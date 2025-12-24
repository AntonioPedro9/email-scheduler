package middleware

import (
	"net/http"
)

func AuthMiddleware(apiToken string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.Header.Get("Authorization")
			if len(tokenString) < 7 || tokenString[:7] != "Bearer " {
				http.Error(w, "Unauthorized: Missing or invalid Bearer token", http.StatusUnauthorized)
				return
			}

			token := tokenString[7:]
			if token != apiToken {
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			next(w, r)
		}
	}
}
