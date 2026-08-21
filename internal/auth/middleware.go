package auth

import "net/http"

func RequireSession(store *SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasSession(r, store) {
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/signin")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func HasSession(r *http.Request, store *SessionStore) bool {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return false
	}
	return store.Valid(cookie.Value)
}
