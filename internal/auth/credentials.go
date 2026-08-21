package auth

import (
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Credentials struct {
	username     string
	passwordHash []byte
}

func NewCredentials(username, passwordHash string) (Credentials, error) {
	if _, err := bcrypt.Cost([]byte(passwordHash)); err != nil {
		return Credentials{}, fmt.Errorf("admin password hash is not a valid bcrypt hash: %w", err)
	}
	return Credentials{username: username, passwordHash: []byte(passwordHash)}, nil
}

func (c Credentials) Verify(username, password string) bool {
	userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(c.username)) == 1
	passMatch := bcrypt.CompareHashAndPassword(c.passwordHash, []byte(password)) == nil
	return userMatch && passMatch
}
