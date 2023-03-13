package v1

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/shiroyk/cloudcat/lib/logger"
)

// requestLogger the request logger.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		logger.Info("request", "path", r.URL.Path, "method", r.Method, "status", ww.Status())
	})
}

// auth authentication the secret.
func auth(secret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return handleFunc(func(w http.ResponseWriter, r *http.Request) error {
			if secret != strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ") {
				return JSON(w, http.StatusUnauthorized, msg{http.StatusText(http.StatusUnauthorized)})
			}

			next.ServeHTTP(w, r)

			return nil
		})
	}
}
