package auth

import (
	"net/http"

	"github.com/mariadb-dal-api/internal/model"
)

// NewAuthMiddleware returns middleware that enforces X-API-Key authentication.
// GET /health is exempt from authentication.
func NewAuthMiddleware(keys []string) func(http.Handler) http.Handler {
	keySet := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		keySet[k] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt GET /health from authentication.
			if r.Method == http.MethodGet && r.URL.Path == "/health" {
				next.ServeHTTP(w, r)
				return
			}

			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				model.WriteError(w, http.StatusUnauthorized, "missing API key")
				return
			}

			if _, ok := keySet[apiKey]; !ok {
				model.WriteError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
