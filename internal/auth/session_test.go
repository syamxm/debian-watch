package auth

import (
	"testing"
	"time"
)

func TestSessionLifecycle(t *testing.T) {
	store := NewSessionStore(time.Hour)

	id, err := store.Create()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if !store.Valid(id) {
		t.Fatal("new session should be valid")
	}

	store.Destroy(id)
	if store.Valid(id) {
		t.Fatal("destroyed session should be invalid")
	}
}

func TestSessionExpires(t *testing.T) {
	store := NewSessionStore(-time.Second)

	id, err := store.Create()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if store.Valid(id) {
		t.Fatal("expired session should be invalid")
	}
}

func TestSessionRejectsUnknownID(t *testing.T) {
	store := NewSessionStore(time.Hour)
	if store.Valid("") || store.Valid("not-a-session") {
		t.Fatal("unknown ids should be invalid")
	}
}

func TestSessionIDsAreUnique(t *testing.T) {
	store := NewSessionStore(time.Hour)
	first, _ := store.Create()
	second, _ := store.Create()
	if first == second {
		t.Fatal("session ids must not repeat")
	}
}
