package httpx

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/syamxm/debian-watch/internal/auth"
	"github.com/syamxm/debian-watch/internal/collect"
	"github.com/syamxm/debian-watch/internal/config"
)

const (
	testUser     = "admin"
	testPassword = "correct-horse"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("generate hash: %v", err)
	}
	creds, err := auth.NewCredentials(testUser, string(hash))
	if err != nil {
		t.Fatalf("new credentials: %v", err)
	}

	cfg := config.Config{
		Addr:         ":0",
		AdminUser:    testUser,
		SessionTTL:   time.Hour,
		CookieSecure: false,
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	server, err := NewServer(
		cfg, log,
		collect.New(context.Background(), log),
		auth.NewSessionStore(cfg.SessionTTL),
		auth.NewLoginLimiter(5, 15*time.Minute),
		creds,
	)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return server.Handler()
}

func csrfToken(t *testing.T, handler http.Handler) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/signin", nil))
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == auth.CSRFCookieName {
			return cookie
		}
	}
	t.Fatal("sign-in page did not set a CSRF cookie")
	return nil
}

func postSignIn(t *testing.T, handler http.Handler, username, password string) *httptest.ResponseRecorder {
	t.Helper()
	csrf := csrfToken(t, handler)

	form := url.Values{
		"username":         {username},
		"password":         {password},
		auth.CSRFFieldName: {csrf.Value},
	}
	req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrf)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestProtectedRouteRedirectsWithoutSession(t *testing.T) {
	handler := newTestHandler(t)

	for _, path := range []string{"/dashboard", "/memory", "/system", "/api/live-stats"} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if rec.Code != http.StatusSeeOther {
			t.Errorf("GET %s = %d, want %d", path, rec.Code, http.StatusSeeOther)
		}
		if location := rec.Header().Get("Location"); location != "/signin" {
			t.Errorf("GET %s redirected to %q, want /signin", path, location)
		}
	}
}

func TestProtectedRouteRejectsHTMXWithoutSession(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/live-stats", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if rec.Header().Get("HX-Redirect") != "/signin" {
		t.Errorf("HX-Redirect = %q, want /signin", rec.Header().Get("HX-Redirect"))
	}
}

func TestSignInRejectsMissingCSRFToken(t *testing.T) {
	handler := newTestHandler(t)

	form := url.Values{"username": {testUser}, "password": {testPassword}}
	req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestSignInRejectsWrongPassword(t *testing.T) {
	handler := newTestHandler(t)
	rec := postSignIn(t, handler, testUser, "wrong")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == auth.SessionCookieName && cookie.Value != "" {
			t.Fatal("failed login must not issue a session cookie")
		}
	}
}

func TestSignInGrantsAccess(t *testing.T) {
	handler := newTestHandler(t)
	rec := postSignIn(t, handler, testUser, testPassword)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	var session *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == auth.SessionCookieName {
			session = cookie
		}
	}
	if session == nil {
		t.Fatal("successful login did not set a session cookie")
	}
	if !session.HttpOnly || session.SameSite != http.SameSiteStrictMode {
		t.Errorf("session cookie flags = HttpOnly:%v SameSite:%v", session.HttpOnly, session.SameSite)
	}

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(session)
	dashboard := httptest.NewRecorder()
	handler.ServeHTTP(dashboard, req)

	if dashboard.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, want %d", dashboard.Code, http.StatusOK)
	}
	if !strings.Contains(dashboard.Body.String(), "performance overview") {
		t.Error("dashboard body missing expected heading")
	}
}

func TestSignInRateLimits(t *testing.T) {
	handler := newTestHandler(t)

	for range 5 {
		postSignIn(t, handler, testUser, "wrong")
	}

	rec := postSignIn(t, handler, testUser, testPassword)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTooManyRequests)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/signin", nil))

	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	}
	for header, value := range want {
		if got := rec.Header().Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Error("CSP missing frame-ancestors directive")
	}
}

func TestHealthEndpointIsPublic(t *testing.T) {
	handler := newTestHandler(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("healthz = %d %q", rec.Code, rec.Body.String())
	}
}
