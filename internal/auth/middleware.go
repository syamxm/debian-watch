package auth

import "net/http"

// RequireSession rejects unauthenticated requests. HTMX requests get an
// HX-Redirect header, since a 303 to the sign-in page would otherwise be
// swapped into the panel it was polling.
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
