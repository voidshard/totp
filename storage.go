package totp

import (
	"fmt"
	"os"

	yaml "gopkg.in/yaml.v3"
)

// Storage for our user data
type Storage interface {
	User(string) (*User, error)
}

// ReadonlyFile is a simple storage backend that reads from a YAML file.
// This is Readonly (obviously).
type ReadonlyFile struct {
	filename string
	users    map[string]*User
}

// NewReadonlyFile creates a new ReadonlyFile storage backend.
func NewReadonlyFile(filename string) (*ReadonlyFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	userdata := []*User{}
	err = yaml.Unmarshal([]byte(data), &userdata)
	if err != nil {
		return nil, err
	}

	users := make(map[string]*User)
	for _, user := range userdata {
		users[user.Username] = user
	}

	return &ReadonlyFile{filename: filename, users: users}, nil
}

// NewDebugStorage creates a new ReadonlyFile storage backend with canned data (see test_data/conf.yaml).
// Obviously, this is for debugging purposes only.
func NewDebugStorage() *ReadonlyFile {
	return &ReadonlyFile{
		users: map[string]*User{
			// canned user data, see test_data/conf.yaml
			"mary":  &User{Username: "mary", Secret: "3UFC3DUK27KESHBWEJDQS4B2HXLHGFZV"},
			"james": &User{Username: "james", Secret: "CV4JDXSYVFRJTHMNG4HUKF3OSTOP6B3H"},
			"test":  &User{Username: "test", Secret: "DSENNVUPIDGLGIH5XE5F7EXPZIZAVZJH"},
		},
	}
}

// User returns a User object by username.
func (r *ReadonlyFile) User(username string) (*User, error) {
	u, ok := r.users[username]
	if !ok {
		return nil, fmt.Errorf("User not found")
	}
	return u, nil
}
