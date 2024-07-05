package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alecthomas/kong"

	"github.com/voidshard/totp"
)

var cli struct {
	Serve    cmdServe    `cmd help:"Serve the API"`
	Generate cmdGenerate `cmd help:"Generate a TOTP QR code"`
}

type cmdServe struct {
	Port     int    `long:"port" default:"8080" help:"Port to listen on" env:"PORT"`
	Config   string `long:"config" default:"conf.yaml" help:"Config file path" env:"USER_CONFIG"`
	Debug    bool   `long:"debug" help:"Enable debug mode." env:"DEBUG"`
	JWTKey   string `long:"jwt-key" env:"JWT_KEY" help:"JWT signing key (required when not in debug mode)"`
	CSRFKey  string `long:"csrf-key" env:"CSRF_KEY" help:"CSRF signing key (recommended)"`
	Redirect string `long:"redirect" default:"/auth/check" env:"REDIRECT" help:"Redirect URL after login"`
	LRUSize  int    `long:"lru-size" default:"250" env:"LRU_SIZE" help:"LRU cache size (used for remembering CSRF tokens)"`
	LRUTTL   int    `long:"lru-ttl" default:"120" env:"LRU_TTL" help:"LRU cache TTL in seconds (used for remembering CSRF tokens)"` // 2 mins
	JWTTTL   int    `long:"jwt-ttl" default:"7200" env:"JWT_TTL" help:"JWT session TTL in seconds"`                                 // 2 hours
	LoginURL string `long:"auth-url" default:"/auth/login" env:"LOGIN_URL" help:"Auth URL"`
	CheckURL string `long:"check-url" default:"/auth/check" env:"CHECK_URL" help:"Check URL"`
	Cookie   string `long:"cookie" default:"totp-auth" env:"COOKIE" help:"Cookie name"`

	OtelResourceAttributes string `long:"otel-resource-attributes" env:"OTEL_RESOURCE_ATTRIBUTES" help:"OpenTelemetry resource attributes" default:"service.name=totp,service.version=0.0.0"`
	SecondsBetweenLogins   int64  `long:"seconds-between-logins" default:"1" env:"SECONDS_BETWEEN_LOGINS" help:"Minimum time between logins in seconds"`

	HTTPReadTimeout  int `long:"http-read-timeout" default:"1" env:"HTTP_READ_TIMEOUT" help:"HTTP read timeout in seconds"`
	HTTPWriteTimeout int `long:"http-write-timeout" default:"1" env:"HTTP_WRITE_TIMEOUT" help:"HTTP write timeout in seconds"`
}

// defaults sets up some default values for the server, generating keys if needed (debug mode only)
func (c *cmdServe) defaults() error {
	if c.JWTKey == "" {
		if c.Debug {
			if c.JWTKey == "" {
				log.Println("No JWT key provided, generating a random one")
				rb, err := randBytes(64)
				if err != nil {
					return err
				}
				c.JWTKey = string(rb)
			}
		} else {
			return fmt.Errorf("JWT key is required")
		}
	}
	if c.CSRFKey == "" {
		if c.Debug {
			log.Println("No CSRF key provided, generating a random one")
			rb, err := randBytes(64)
			if err != nil {
				return err
			}
			c.CSRFKey = string(rb)
		} else {
			return fmt.Errorf("CSRF key is required")
		}
	}
	return nil
}

// Run starts the TOTP server.
func (c *cmdServe) Run() error {
	err := c.defaults()
	if err != nil {
		return err
	}

	var store totp.Storage
	if c.Debug {
		log.Println("Debug mode enabled, loading test user only")
		store = totp.NewDebugStorage()
	} else {
		store, err = totp.NewReadonlyFile(c.Config)
		if err != nil {
			return err
		}
	}
	return totp.ServeHTTP(
		totp.WithCSRFKey([]byte(c.JWTKey)),
		totp.WithJWTKey([]byte(c.CSRFKey)),
		totp.WithPort(c.Port),
		totp.WithStorage(store),
		totp.WithLRUCacheSize(c.LRUSize),
		totp.WithLRUCacheTTL(time.Duration(c.LRUTTL)*time.Second),
		totp.WithJWTSessionTTL(time.Duration(c.JWTTTL)*time.Second),
		totp.WithRedirect(c.Redirect),
		totp.WithAuthCheckURL(c.CheckURL),
		totp.WithAuthLoginURL(c.LoginURL),
		totp.WithCookieName(c.Cookie),
		totp.WithSecondsBetweenLogins(c.SecondsBetweenLogins),
		totp.WithHTTPReadTimeout(time.Duration(c.HTTPReadTimeout)*time.Second),
		totp.WithHTTPWriteTimeout(time.Duration(c.HTTPWriteTimeout)*time.Second),
	)
}

type cmdGenerate struct {
	Issuer  string `short:"i" long:"issuer" default:"example.org" env:"ISSUER" help:"Issuer name for TOTP"`
	Account string `arg:"" help:"Account name"`
	Output  string `long:"output" short:"o" default:"qr.png" help:"Path to save QR code"`
}

// Run generates a new TOTP secret and saves a QR code to the output path.
// Intended for an admin creating a user account
func (c *cmdGenerate) Run() error {
	secret, _, qrData, err := totp.NewTOTP(c.Issuer, c.Account)
	if err != nil {
		return err
	}

	fmt.Println("Secret:", secret)
	fmt.Println("QR code saved to:", c.Output)
	return os.WriteFile(c.Output, qrData, 0644)
}

// randBytes generates n random bytes.
// Only used to be helpful & generate keys for debug style mode.
func randBytes(n int) ([]byte, error) {
	data := make([]byte, n)
	_, err := rand.Read(data)
	return data, err
}

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
