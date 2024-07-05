package totp

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
)

func expiredJWT(key []byte, username string) (string, error) {
	claims := &JWTClaim{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: 1,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

func TestJWT(t *testing.T) {
	secret := []byte("test-secret")

	token, err := newJWT(secret, "good-user", 1*time.Hour)
	assert.Nil(t, err)

	tokenBad, err := newJWT([]byte("some other"), "user-smart", 1*time.Hour)
	assert.Nil(t, err)

	tokenExp, err := expiredJWT(secret, "expired-user")
	assert.Nil(t, err)

	cases := []struct {
		Name        string
		Token       string
		ExpectError bool
	}{
		{"good-user", token, false},
		{"bad-user", "bad-token", true},
		{"bad-smarter-user", tokenBad, true},
		{"expired-user", tokenExp, true},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			result, err := validateJWT(secret, c.Token)
			if c.ExpectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, c.Name, result.Username)
			}
		})
	}
}
