package totp

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"go.opentelemetry.io/otel/attribute"
)

// server is our HTTP server
type server struct {
	// configurable
	csrfKey              []byte
	jwtKey               []byte
	port                 int
	cacheSize            int
	cacheTTL             time.Duration
	jwtSessionTTL        time.Duration
	redirect             string
	authCheckURL         string
	authLoginURL         string
	store                Storage
	secondsBetweenLogins int64
	cookieName           string
	httpReadTimeout      time.Duration
	httpWriteTimeout     time.Duration

	// internal
	sessions  *expirable.LRU[string, bool]
	re        *regexp.Regexp
	lastLogin int64
}

// buildServer creates a new server with the given options - this allows us to track server state
// between handlers.
func buildServer(opts ...WebOption) (*server, error) {
	// set up our server struct
	s := &server{ // default values
		port:                 8080,
		cacheSize:            250,
		cacheTTL:             time.Minute * 2,
		jwtSessionTTL:        time.Hour * 2,
		redirect:             "/auth/check",
		authCheckURL:         "/auth/check",
		authLoginURL:         "/auth/login",
		cookieName:           "totp-auth",
		secondsBetweenLogins: 1,
		re:                   regexp.MustCompile(`^[a-zA-Z0-9]+`),
		httpReadTimeout:      time.Second,
		httpWriteTimeout:     time.Second,
	}
	for _, opt := range opts { // apply options
		opt(s)
	}
	s.sessions = expirable.NewLRU[string, bool](s.cacheSize, nil, s.cacheTTL)

	// validate our configuration
	if s.csrfKey == nil {
		return nil, fmt.Errorf("CSRF key is required")
	}
	if s.jwtKey == nil {
		return nil, fmt.Errorf("JWT key is required")
	}
	if s.store == nil {
		return nil, fmt.Errorf("Storage is required")
	}

	return s, nil
}

// ServeHTTP starts the HTTP server on the given port.
func ServeHTTP(opts ...WebOption) error {
	s, err := buildServer(opts...)
	if err != nil {
		return err
	}

	// set up HTTP routes & opentelemetry (see. https://opentelemetry.io/docs/languages/go/getting-started/)
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return err
	}
	defer otelShutdown()

	// build & start HTTP server.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  s.httpReadTimeout,
		WriteTimeout: s.httpWriteTimeout,
		Handler:      s.newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		log.Println("Server is running at :", s.port)
		srvErr <- srv.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		log.Println("Error starting server:", err)
		// Error when starting HTTP server.
		return err
	case <-ctx.Done():
		log.Println("Shutting down server")
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}

	// When Shutdown is called, ListenAndServe immediately returns ErrServerClosed.
	return srv.Shutdown(context.Background())
}

// newHTTPHandler creates a new HTTP handler for the server.
func (s *server) newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// Add the /auth/check and /auth/login endpoints.
	mux.Handle(s.authCheckURL, otelWrapHandler(http.HandlerFunc(s.authCheck), s.authCheckURL))
	mux.Handle(s.authLoginURL, otelWrapHandler(http.HandlerFunc(s.authLogin), s.authLoginURL))

	return mux
}

// authCheck is the handler for the /auth/check endpoint.
// Validates that a user is logged in (JWT cookie is present and valid).
func (s *server) authCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Println("Method not allowed", r.Method)
		writeError(w, "No", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie(s.cookieName)
	if err != nil {
		log.Println("No cookie found")
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jwt, err := validateJWT(s.jwtKey, cookie.Value)
	if err != nil {
		log.Println("Invalid JWT:", err)
		writeError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	_, span := tracer.Start(r.Context(), "access-approved")
	defer span.End()
	span.AddEvent("Access approved")
	span.SetAttributes(attribute.String("user", jwt.Username))

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome"))
}

// authLogin is the handler for the /auth/login endpoint.
// GET returns a login form.
// POST attempts a login, validating the TOTP and generating a JWT.
func (s *server) authLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.loginGet(w, r)
		return
	} else if r.Method == http.MethodPost {
		if time.Now().Unix() < s.lastLogin+s.secondsBetweenLogins {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		s.lastLogin = time.Now().Unix()

		s.loginPost(w, r)
		return
	}
	log.Println("Method not allowed", r.Method)
	writeError(w, "No", http.StatusMethodNotAllowed)
}

// loginPost handles the POST request for the login form.
// - reads sent values
// - validates the CSRF token
// - checks if the CSRF token has already been used
// - validates the username
// - validates the TOTP
// - generates a JWT
// - sets the JWT cookie
// - redirects to the configured URL
func (s *server) loginPost(w http.ResponseWriter, r *http.Request) {
	// parse the form
	err := r.ParseForm()
	if err != nil {
		log.Println("Error parsing form:", err)
		s.sendLoginPage(w, r, http.StatusBadRequest)
		return
	}

	// read and validate our fields
	csrf := r.Form.Get("csrf")
	_, err = validateJWT(s.csrfKey, csrf)
	if err != nil {
		log.Println("Invalid CSRF token:", err)
		s.sendLoginPage(w, r, http.StatusUnauthorized)
		return
	}

	// check if the CSRF token has already been used
	_, ok := s.sessions.Get(csrf)
	if ok {
		log.Println("CSRF token already used")
		s.sendLoginPage(w, r, http.StatusUnauthorized)
		return
	}

	// remember this token for the session length (after this the JWT will expire anyways)
	s.sessions.Add(csrf, true)

	user := r.Form.Get("user")
	if !s.re.MatchString(user) {
		log.Println("Invalid username:", user)
		s.sendLoginPage(w, r, http.StatusUnauthorized)
		return
	}

	token := strings.Replace(r.Form.Get("token"), " ", "", -1)
	if !s.re.MatchString(token) {
		log.Println("Invalid token:", token)
		s.sendLoginPage(w, r, http.StatusUnauthorized)
		return
	}

	// load the user from the store
	userObj, err := s.store.User(user)
	if err != nil {
		log.Println("Error loading user:", err)
		s.sendLoginPage(w, r, http.StatusUnauthorized)
		return
	}

	// validate the TOTP
	if !validateTOTP(userObj.Secret, token) {
		log.Println("Invalid TOTP")
		s.sendLoginPage(w, r, http.StatusUnauthorized)
		return
	}

	// login successful -- generate JWT
	_, span := tracer.Start(r.Context(), "login-success")
	defer span.End()
	span.AddEvent("Access approved")
	span.SetAttributes(attribute.String("user", userObj.Username))

	jwtKey, err := newJWT(s.jwtKey, user, s.jwtSessionTTL)
	log.Println("User logged in:", userObj.Username)
	w.Header().Set("Location", s.redirect)
	writeCookie(w, s.cookieName, jwtKey)
	w.WriteHeader(http.StatusFound)
}

// loginGet handles the GET request for the login form.
// - generates a session / CSRF token
// - returns the login form with the CSRF token
func (s *server) loginGet(w http.ResponseWriter, r *http.Request) {
	s.sendLoginPage(w, r, http.StatusOK)
}

func (s *server) sendLoginPage(w http.ResponseWriter, r *http.Request, statusOnSend int) {
	// generate a new session
	rng, err := randBytes(64)
	if err != nil {
		log.Println("Error generating random bytes:", err)
		writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	sessID := fmt.Sprintf("%d-%x", time.Now().Unix(), rng)

	// generate a session token
	// ie. this is how long we're willing to accept the CSRF token back
	sessTkn, err := newJWT(s.csrfKey, sessID, s.cacheTTL)
	if err != nil {
		log.Println("Error generating session JWT:", err)
		writeError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// return the login form with the CSRF token
	writeIndex(w, s.authLoginURL, sessTkn, statusOnSend)
}

// writeCookie writes a cookie to the response.
func writeCookie(w http.ResponseWriter, name, value string) {
	cookie := http.Cookie{}
	cookie.Name = name
	cookie.Value = value
	cookie.Secure = true
	cookie.HttpOnly = false
	cookie.Path = "/"
	http.SetCookie(w, &cookie)
}

// writeIndex writes the login form to the response.
func writeIndex(w http.ResponseWriter, loginURL, csrf string, status int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf(`<html><head><title>Please Log In</title></head>
<body><form action="%s" method="POST">
<input placeholder="username" type="text" name="user">
<input placeholder="code" type="text" name="token">
<input type="hidden" name="csrf" value="%s">
<input type="submit" value="Submit">
</form></body></html>`, loginURL, csrf)))
}

// writeError writes an error message to the response.
func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	w.Write([]byte(msg))
}
