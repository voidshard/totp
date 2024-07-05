package totp

import (
	"bytes"
	"image"
	"image/png"

	// "github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// NewTOTP creates a new TOTP key for the given account.
func NewTOTP(issuer, account string) (string, image.Image, []byte, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
	})
	if err != nil {
		return "", nil, nil, err
	}

	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		panic(err)
	}
	err = png.Encode(&buf, img)

	return key.Secret(), img, buf.Bytes(), err
}

// validateTOTP validates the given TOTP code against the secret.
func validateTOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}
