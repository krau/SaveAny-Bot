package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/krau/SaveAny-Bot/config"
)

func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.C().API

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header format")
				return
			}

			token := parts[1]

			if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.Token)) != 1 {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
