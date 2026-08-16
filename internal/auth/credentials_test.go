package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func testCredentials(t *testing.T, username, password string) Credentials {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}
	creds, err := NewCredentials(username, string(hash))
	if err != nil {
		t.Fatalf("new credentials: %v", err)
	}
	return creds
}

func TestNewCredentialsRejectsNonBcryptHash(t *testing.T) {
	if _, err := NewCredentials("admin", "plaintext"); err == nil {
		t.Fatal("expected error for non-bcrypt hash")
	}
}

func TestVerify(t *testing.T) {
	creds := testCredentials(t, "admin", "correct-horse")

	cases := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{"valid", "admin", "correct-horse", true},
		{"wrong password", "admin", "wrong", false},
		{"wrong username", "root", "correct-horse", false},
		{"empty", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := creds.Verify(tc.username, tc.password); got != tc.want {
				t.Errorf("Verify(%q, %q) = %v, want %v", tc.username, tc.password, got, tc.want)
			}
		})
	}
}
