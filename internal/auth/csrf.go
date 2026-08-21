package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
)

const (
	CSRFCookieName = "dw_csrf"
	CSRFFieldName  = "csrf_token"
)

type contextKey struct{}

var csrfContextKey contextKey

func CSRF(secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := ensureCSRFToken(w, r, secure)
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !isSafeMethod(r.Method) && !validCSRFRequest(r, token) {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), csrfContextKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CSRFToken(ctx context.Context) string {
	token, _ := ctx.Value(csrfContextKey).(string)
	return token
}

func RotateCSRFToken(w http.ResponseWriter, secure bool) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	setCSRFCookie(w, token, secure)
	return nil
}

func ensureCSRFToken(w http.ResponseWriter, r *http.Request, secure bool) (string, error) {
	if cookie, err := r.Cookie(CSRFCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	setCSRFCookie(w, token, secure)
	return token, nil
}

func setCSRFCookie(w http.ResponseWriter, token string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func validCSRFRequest(r *http.Request, token string) bool {
	cookie, err := r.Cookie(CSRFCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	submitted := r.PostFormValue(CSRFFieldName)
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submitted)) == 1 &&
		subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(token)) == 1
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}
