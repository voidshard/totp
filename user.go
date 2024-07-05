package totp

// User object holds the bare minimum
type User struct {
	// Some uniqe string
	Username string `yaml:"username"`

	// TOTP secret
	Secret string `yaml:"secret"`
}
