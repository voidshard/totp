package totp

import (
	"crypto/rand"
)

// randBytes generates a random byte slice of length n.
func randBytes(n int) ([]byte, error) {
	data := make([]byte, n)
	_, err := rand.Read(data)
	return data, err
}
