package totp

import "time"

type WebOption func(*server)

// WithCSRFKey sets the CSRF key for the server, used to sign CSRF tokens.
// If this changes between page load and form submission, the form submission will be rejected.
// (Required).
func WithCSRFKey(key []byte) WebOption {
	return func(s *server) {
		s.csrfKey = key
	}
}

// WithJWTKey sets the JWT key for the server, used to sign JWT tokens.
// If this is changed existing tokens will be invalid.
// (Required).
func WithJWTKey(key []byte) WebOption {
	return func(s *server) {
		s.jwtKey = key
	}
}

// WithPort sets the port the server will listen on.
func WithPort(port int) WebOption {
	return func(s *server) {
		s.port = port
	}
}

// WithLRUCacheSize sets the size of the LRU cache used to recall CSRF tokens.
// Used to enforce that a token can only be used once.
func WithLRUCacheSize(size int) WebOption {
	return func(s *server) {
		s.cacheSize = size
	}
}

// WithLRUCacheTTL sets the TTL of the LRU cache used to recall CSRF tokens.
// Used to enforce that a token can only be used once.
func WithLRUCacheTTL(ttl time.Duration) WebOption {
	return func(s *server) {
		s.cacheTTL = ttl
	}
}

// WithJWTSessionTTL sets the TTL of the JWT session. After this time has elapsed the token is invalid
func WithJWTSessionTTL(ttl time.Duration) WebOption {
	return func(s *server) {
		s.jwtSessionTTL = ttl
	}
}

// WithRedirect sets the URL to redirect to after a successful login
func WithRedirect(redirect string) WebOption {
	return func(s *server) {
		s.redirect = redirect
	}
}

// WithAuthCheckURL sets the URL to check the authentication status (eg. the cookie is set & JWT is valid)
func WithAuthCheckURL(url string) WebOption {
	return func(s *server) {
		s.authCheckURL = url
	}
}

// WithAuthLoginURL sets the URL to login to the authentication system
func WithAuthLoginURL(url string) WebOption {
	return func(s *server) {
		s.authLoginURL = url
	}
}

// WithStorage sets the storage backend for the server (required)
func WithStorage(store Storage) WebOption {
	return func(s *server) {
		s.store = store
	}
}

// WithCookieName sets the name of the cookie used to store the JWT token
func WithCookieName(name string) WebOption {
	return func(s *server) {
		s.cookieName = name
	}
}

// WithSecondsBetweenLogins sets the minimum time between logins.
// That is, we ratelimit attempts to POST to /auth/login
func WithSecondsBetweenLogins(seconds int64) WebOption {
	return func(s *server) {
		s.secondsBetweenLogins = seconds
	}
}

// WithHTTPReadTimeout sets the read timeout for the HTTP server
func WithHTTPReadTimeout(timeout time.Duration) WebOption {
	return func(s *server) {
		s.httpReadTimeout = timeout
	}
}

// WithHTTPWriteTimeout sets the write timeout for the HTTP server
func WithHTTPWriteTimeout(timeout time.Duration) WebOption {
	return func(s *server) {
		s.httpWriteTimeout = timeout
	}
}
