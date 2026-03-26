package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// Feature: mariadb-dal-api, Property 2: Auth rejects missing or invalid API keys

// TestPropertyAuthRejectsMissingOrInvalidKey verifies that for any protected endpoint,
// requests with a missing or invalid X-API-Key header always receive a 401 response.
// Validates: Requirements 2.2, 2.3, 2.4
func TestPropertyAuthRejectsMissingOrInvalidKey(t *testing.T) {
	// Generator for a non-empty set of valid API keys (1-5 keys)
	validKeySetGen := rapid.Custom(func(t *rapid.T) []string {
		n := rapid.IntRange(1, 5).Draw(t, "num_keys")
		keys := make([]string, n)
		for i := range keys {
			keys[i] = rapid.StringMatching(`[a-zA-Z0-9_\-]{8,32}`).Draw(t, "key")
		}
		return keys
	})

	// Generator for a protected endpoint path (not /health)
	protectedPathGen := rapid.Custom(func(t *rapid.T) string {
		segments := []string{"users", "orders", "items", "products", "records", "data"}
		resource := rapid.SampledFrom(segments).Draw(t, "resource")
		// Optionally append an ID
		if rapid.Bool().Draw(t, "has_id") {
			id := rapid.IntRange(1, 9999).Draw(t, "id")
			return "/" + resource + "/" + rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).Draw(t, "id_str")[:0] + string(rune('0'+id%10))
		}
		return "/" + resource
	})

	// Generator for HTTP methods
	methodGen := rapid.SampledFrom([]string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	})

	// Dummy next handler that should never be called for rejected requests
	dummyNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("missing X-API-Key header yields 401", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Feature: mariadb-dal-api, Property 2: Auth rejects missing or invalid API keys
			keys := validKeySetGen.Draw(t, "keys")
			path := protectedPathGen.Draw(t, "path")
			method := methodGen.Draw(t, "method")

			middleware := NewAuthMiddleware(keys)
			handler := middleware(dummyNext)

			req := httptest.NewRequest(method, path, nil)
			// No X-API-Key header set
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 for missing key on %s %s, got %d", method, path, rr.Code)
			}
		})
	})

	t.Run("invalid X-API-Key header yields 401", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Feature: mariadb-dal-api, Property 2: Auth rejects missing or invalid API keys
			keys := validKeySetGen.Draw(t, "keys")
			path := protectedPathGen.Draw(t, "path")
			method := methodGen.Draw(t, "method")

			// Generate a key that is guaranteed not to be in the valid key set
			invalidKey := rapid.StringMatching(`[a-zA-Z0-9_\-]{8,32}`).Draw(t, "invalid_key")
			// Ensure it's not accidentally in the valid set
			for _, k := range keys {
				if k == invalidKey {
					t.Skip()
				}
			}

			middleware := NewAuthMiddleware(keys)
			handler := middleware(dummyNext)

			req := httptest.NewRequest(method, path, nil)
			req.Header.Set("X-API-Key", invalidKey)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 for invalid key %q on %s %s, got %d", invalidKey, method, path, rr.Code)
			}
		})
	})
}

// Feature: mariadb-dal-api, Property 3: Health endpoint is exempt from authentication

// TestPropertyHealthEndpointAuthExemption verifies that GET /health always passes through
// the auth middleware regardless of API key configuration or X-API-Key header value.
// Validates: Requirements 2.5
func TestPropertyHealthEndpointAuthExemption(t *testing.T) {
	// Generator for API key configurations (0-5 keys, including empty set)
	keyConfigGen := rapid.Custom(func(t *rapid.T) []string {
		n := rapid.IntRange(0, 5).Draw(t, "num_keys")
		keys := make([]string, n)
		for i := range keys {
			keys[i] = rapid.StringMatching(`[a-zA-Z0-9_\-]{8,32}`).Draw(t, "key")
		}
		return keys
	})

	// Generator for X-API-Key header: present with random value, or absent
	apiKeyHeaderGen := rapid.Custom(func(t *rapid.T) string {
		// "" means absent; any other value means present with that value
		if rapid.Bool().Draw(t, "header_present") {
			return rapid.StringMatching(`[a-zA-Z0-9_\-]{0,40}`).Draw(t, "header_value")
		}
		return ""
	})

	dummyNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rapid.Check(t, func(t *rapid.T) {
		// Feature: mariadb-dal-api, Property 3: Health endpoint is exempt from authentication
		keys := keyConfigGen.Draw(t, "keys")
		headerValue := apiKeyHeaderGen.Draw(t, "header_value")

		middleware := NewAuthMiddleware(keys)
		handler := middleware(dummyNext)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		if headerValue != "" {
			req.Header.Set("X-API-Key", headerValue)
		}
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusUnauthorized {
			t.Fatalf("expected non-401 for GET /health (keys=%v, X-API-Key=%q), got 401", keys, headerValue)
		}
	})
}
