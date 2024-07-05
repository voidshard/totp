### TOTP Auth Server

Dead simple no-frills attached totp auth server with two commands
```
Usage: totp generate <account> [flags]

Generate a TOTP QR code

Arguments:
  <account>    Account name

Flags:
  -h, --help                    Show context-sensitive help.

  -i, --issuer="example.org"    Issuer name for TOTP ($ISSUER)
  -o, --output="qr.png"         Path to save QR code
```
Generate the TOTP QR code & secret


```
Usage: main serve [flags]

Serve the API

Flags:
  -h, --help                                                                  Show context-sensitive help.

      --port=8080                                                             Port to listen on ($PORT)
      --config="conf.yaml"                                                    Config file path ($USER_CONFIG)
      --debug                                                                 Enable debug mode ($DEBUG).
      --jwt-key=STRING                                                        JWT signing key (required when not in debug mode) ($JWT_KEY)
      --csrf-key=STRING                                                       CSRF signing key (recommended) ($CSRF_KEY)
      --redirect="/auth/check"                                                Redirect URL after login ($REDIRECT)
      --lru-size=250                                                          LRU cache size (used for remembering CSRF tokens) ($LRU_SIZE)
      --lruttl=120                                                            LRU cache TTL in seconds (used for remembering CSRF tokens) ($LRU_TTL)
      --jwtttl=7200                                                           JWT session TTL in seconds ($JWT_TTL)
      --login-url="/auth/login"                                               Auth URL ($AUTH_URL)
      --check-url="/auth/check"                                               Check URL ($CHECK_URL)
      --cookie="totp-auth"                                                    Cookie name ($COOKIE)
      --otel-resource-attributes="service.name=totp,service.version=0.0.0"    OpenTelemetry resource attributes ($OTEL_RESOURCE_ATTRIBUTES)
      --seconds-between-logins=1                                              Minimum time between logins in seconds ($SECONDS_BETWEEN_LOGINS)
      --http-read-timeout=1                                                   HTTP read timeout in seconds ($HTTP_READ_TIMEOUT)
      --http-write-timeout=1                                                  HTTP write timeout in seconds ($HTTP_WRITE_TIMEOUT)
```
Run a HTTP server with 
  - /auth/login
        Writes out a simple HTTP page with a user, TOTP code challenge. A successful login sets a Cookie (JWT) and redirects the user. The server limits login attempts to 1 per second and injects a CSRF token into each index page. JWT cookies expire in two hours.
  - /auth/check
        Check makes sure that the JWT Cookie is set & signed (returning HTTP 401 or HTTP 200).


Currently 'users' are added via a read-only YAML file (see test_data/conf.yaml for an example), but the web server takes an interface if you wanted to implement something more complex.


Intended to work alongside a reverse proxy like nginx, with some config akin to
```
        location /auth {
                proxy_pass http://127.0.0.1:8080; # This is the TOTP Server
                proxy_set_header X-Original-URI $request_uri;
        }

        # This ensures that if the TOTP server returns 401 we redirect to login
        error_page 401 = @error401;
        location @error401 {
            return 302 /auth/login;
        }

        location / {
                auth_request /auth/check;
                proxy_pass http://127.0.0.1:8888; # whatever you're redirecting to
        }
```
The idea is to protect route(s) behind this TOTP login.


Idea taken from https://github.com/newhouseb/simpleotp
