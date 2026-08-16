// Package auth provides single-user credential checks, in-memory sessions,
// CSRF protection and the middleware that guards the dashboard routes.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

const SessionCookieName = "dw_session"

// SessionStore keeps sessions in memory only. A restart invalidates every
// session, which is acceptable for a single-user dashboard.
type SessionStore struct {
	mu       sync.Mutex
	ttl      time.Duration
	sessions map[string]time.Time
}

func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{
		ttl:      ttl,
		sessions: make(map[string]time.Time),
	}
}

func (s *SessionStore) Create() (string, error) {
	id, err := randomToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = time.Now().Add(s.ttl)
	return id, nil
}

func (s *SessionStore) Valid(id string) bool {
	if id == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.sessions[id]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(s.sessions, id)
		return false
	}
	return true
}

func (s *SessionStore) Destroy(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *SessionStore) Sweep(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.mu.Lock()
			for id, expiry := range s.sessions {
				if now.After(expiry) {
					delete(s.sessions, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

func SetSessionCookie(w http.ResponseWriter, id string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func ClearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
