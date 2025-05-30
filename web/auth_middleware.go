package web

import "net/http"

func (handler *HttpRouteHandler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	if handler.UseAuth {
		return func(w http.ResponseWriter, r *http.Request) {
			if !isAuthenticated(r, handler.SecretKey) {
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
