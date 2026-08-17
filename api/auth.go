package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/krau/SaveAny-Bot/config"
)

// AuthMiddleware 返回认证中间件
func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cfg := config.C().API

			// 从请求头获取 token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "missing authorization header")
				return
			}

			// 提取 Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid authorization header format")
				return
			}

			token := parts[1]

			// 验证 token
			if subtle.ConstantTimeCompare([]byte(token), []byte(cfg.Token)) != 1 {
				WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
