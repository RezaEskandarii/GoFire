package web

import "net/http"

func authMiddleware(useAuth bool, next http.HandlerFunc) http.HandlerFunc {
	if useAuth {
		return func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("auth")
			if err != nil || !isValidAuthToken(cookie.Value) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next(w, r)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
	}
}
