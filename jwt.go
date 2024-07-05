package totp

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

// Claims we want to store in the JWT
type JWTClaim struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// newJWT creates a new JWT token with the given username and expiration time.
func newJWT(key []byte, username string, ttl time.Duration) (string, error) {
	claims := &JWTClaim{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(ttl).Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

// validateJWT checks the given token and returns the claims if it's valid.
func validateJWT(key []byte, signedToken string) (*JWTClaim, error) {
	token, err := jwt.ParseWithClaims(
		signedToken, &JWTClaim{},
		func(token *jwt.Token) (interface{}, error) {
			if token.Method == nil {
				return nil, errors.New("unexpected signing method")
			} else if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(key), nil
		},
	)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaim)
	if !ok {
		return nil, errors.New("couldn't parse claims")
	}
	if claims.ExpiresAt < time.Now().Local().Unix() {
		return nil, errors.New("token expired")
	}

	return claims, nil
}
