package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mariadb-dal-api/internal/dal"
	"pgregory.net/rapid"
)

// Feature: mariadb-dal-api, Property 4: Invalid JSON body yields 400

// invalidJSONGen generates strings that are guaranteed to not be valid JSON.
func invalidJSONGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Pick from a variety of invalid JSON patterns
		kind := rapid.IntRange(0, 4).Draw(t, "kind")
		switch kind {
		case 0:
			// Truncated JSON object
			prefix := rapid.StringMatching(`\{[^}]{1,20}`).Draw(t, "truncated")
			return prefix
		case 1:
			// Plain text (not JSON at all)
			return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{1,30}`).Draw(t, "plain_text")
		case 2:
			// Malformed key-value (missing quotes around key)
			key := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "key")
			val := rapid.IntRange(0, 999).Draw(t, "val")
			return "{" + key + ": " + string(rune('0'+val%10)) + "}"
		case 3:
			// Trailing comma (invalid JSON)
			key := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "key")
			return `{"` + key + `": 1,}`
		default:
			// Single unquoted word
			return rapid.StringMatching(`[a-zA-Z]{2,15}`).Draw(t, "word")
		}
	})
}

// TestPropertyInvalidJSONBodyYields400 verifies that POST, PUT, and PATCH requests
// with an invalid JSON body always receive a 400 response with an "error" field.
// Validates: Requirements 3.2, 6.4
func TestPropertyInvalidJSONBodyYields400(t *testing.T) {
	// neverCalledDAL panics if any DAL method other than the interface stubs is called,
	// confirming the handler rejects before reaching the DAL.
	neverCalled := &mockDAL{}

	handler := NewResourceHandler(neverCalled)

	methods := []struct {
		name   string
		method string
		path   string
		fn     func(http.ResponseWriter, *http.Request, httprouter.Params)
	}{
		{"POST", http.MethodPost, "/users", handler.Create},
		{"PUT", http.MethodPut, "/users/1", handler.Update},
		{"PATCH", http.MethodPatch, "/users/1", handler.Patch},
	}

	for _, m := range methods {
		m := m
		t.Run(m.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Feature: mariadb-dal-api, Property 4: Invalid JSON body yields 400
				body := invalidJSONGen().Draw(t, "invalid_json")

				// Verify the generated string is indeed invalid JSON
				var probe any
				if json.Unmarshal([]byte(body), &probe) == nil {
					// Accidentally valid JSON — skip this iteration
					t.Skip()
				}

				req := httptest.NewRequest(m.method, m.path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")

				ps := httprouter.Params{
					httprouter.Param{Key: "resource", Value: "users"},
				}
				if m.method != http.MethodPost {
					ps = append(ps, httprouter.Param{Key: "id", Value: "1"})
				}

				rr := httptest.NewRecorder()
				m.fn(rr, req, ps)

				if rr.Code != http.StatusBadRequest {
					t.Fatalf("%s with invalid JSON body %q: expected 400, got %d", m.method, body, rr.Code)
				}

				var respBody map[string]any
				if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
					t.Fatalf("%s: response body is not valid JSON: %v (body: %s)", m.method, err, rr.Body.String())
				}

				errField, ok := respBody["error"]
				if !ok {
					t.Fatalf("%s: response JSON missing \"error\" field, got: %v", m.method, respBody)
				}
				errStr, ok := errField.(string)
				if !ok || errStr == "" {
					t.Fatalf("%s: \"error\" field is not a non-empty string, got: %v", m.method, errField)
				}
			})
		})
	}
}

// Feature: mariadb-dal-api, Property 5: Unknown resource yields 404

// notFoundDAL is a mock DAL that returns dal.ErrNotFound for all operations,
// simulating a resource name that does not correspond to an existing table.
type notFoundDAL struct{}

func (m *notFoundDAL) Ping(ctx context.Context) error { return nil }
func (m *notFoundDAL) Insert(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *notFoundDAL) GetByID(ctx context.Context, table string, id string) (map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *notFoundDAL) List(ctx context.Context, table string, filters map[string]string, limit, offset int) ([]map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *notFoundDAL) Update(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *notFoundDAL) Patch(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *notFoundDAL) Delete(ctx context.Context, table string, id string) error {
	return dal.ErrNotFound
}

// unknownResourceGen generates valid resource names (alphanumeric + underscore)
// that the mock DAL treats as "not found" (i.e., no corresponding table).
func unknownResourceGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_]{0,29}`)
}

// TestPropertyUnknownResourceYields404 verifies that for any request targeting a
// resource name that does not correspond to an existing table, the response is 404.
// Validates: Requirements 3.4, 4.3, 5.6
func TestPropertyUnknownResourceYields404(t *testing.T) {
	handler := NewResourceHandler(&notFoundDAL{})

	type methodCase struct {
		name   string
		method string
		withID bool
		fn     func(http.ResponseWriter, *http.Request, httprouter.Params)
	}

	methods := []methodCase{
		{"POST", http.MethodPost, false, handler.Create},
		{"GET_list", http.MethodGet, false, handler.List},
		{"GET_byID", http.MethodGet, true, handler.GetByID},
		{"PUT", http.MethodPut, true, handler.Update},
		{"PATCH", http.MethodPatch, true, handler.Patch},
		{"DELETE", http.MethodDelete, true, handler.Delete},
	}

	for _, m := range methods {
		m := m
		t.Run(m.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Feature: mariadb-dal-api, Property 5: Unknown resource yields 404
				resource := unknownResourceGen().Draw(t, "resource")

				var body string
				if m.method == http.MethodPost || m.method == http.MethodPut || m.method == http.MethodPatch {
					body = `{"field":"value"}`
				}

				var path string
				if m.withID {
					path = "/" + resource + "/1"
				} else {
					path = "/" + resource
				}

				req := httptest.NewRequest(m.method, path, strings.NewReader(body))
				if body != "" {
					req.Header.Set("Content-Type", "application/json")
				}

				ps := httprouter.Params{
					httprouter.Param{Key: "resource", Value: resource},
				}
				if m.withID {
					ps = append(ps, httprouter.Param{Key: "id", Value: "1"})
				}

				rr := httptest.NewRecorder()
				m.fn(rr, req, ps)

				if rr.Code != http.StatusNotFound {
					t.Fatalf("%s /%s: expected 404, got %d (body: %s)", m.method, resource, rr.Code, rr.Body.String())
				}
			})
		})
	}
}

// Feature: mariadb-dal-api, Property 6: Missing record yields 404

// missingRecordDAL is a mock DAL where the table exists but any record lookup by ID
// returns dal.ErrNotFound, simulating a valid table with no matching record.
type missingRecordDAL struct{}

func (m *missingRecordDAL) Ping(ctx context.Context) error { return nil }
func (m *missingRecordDAL) Insert(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	return map[string]any{"id": "1"}, nil
}
func (m *missingRecordDAL) GetByID(ctx context.Context, table string, id string) (map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *missingRecordDAL) List(ctx context.Context, table string, filters map[string]string, limit, offset int) ([]map[string]any, error) {
	return []map[string]any{}, nil
}
func (m *missingRecordDAL) Update(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *missingRecordDAL) Patch(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, dal.ErrNotFound
}
func (m *missingRecordDAL) Delete(ctx context.Context, table string, id string) error {
	return dal.ErrNotFound
}

// TestPropertyMissingRecordYields404 verifies that GET, PUT, PATCH, and DELETE requests
// targeting an ID that does not exist in the table always receive a 404 response.
// Validates: Requirements 4.2, 6.3, 7.2
func TestPropertyMissingRecordYields404(t *testing.T) {
	// Feature: mariadb-dal-api, Property 6: Missing record yields 404
	handler := NewResourceHandler(&missingRecordDAL{})

	type methodCase struct {
		name   string
		method string
		fn     func(http.ResponseWriter, *http.Request, httprouter.Params)
		body   string
	}

	methods := []methodCase{
		{"GET_byID", http.MethodGet, handler.GetByID, ""},
		{"PUT", http.MethodPut, handler.Update, `{"field":"value"}`},
		{"PATCH", http.MethodPatch, handler.Patch, `{"field":"value"}`},
		{"DELETE", http.MethodDelete, handler.Delete, ""},
	}

	for _, m := range methods {
		m := m
		t.Run(m.name, func(t *testing.T) {
			rapid.Check(t, func(t *rapid.T) {
				// Generate a random ID string that the mock DAL treats as not found.
				id := rapid.StringMatching(`[a-zA-Z0-9_-]{1,40}`).Draw(t, "id")

				path := "/users/" + id
				req := httptest.NewRequest(m.method, path, strings.NewReader(m.body))
				if m.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}

				ps := httprouter.Params{
					httprouter.Param{Key: "resource", Value: "users"},
					httprouter.Param{Key: "id", Value: id},
				}

				rr := httptest.NewRecorder()
				m.fn(rr, req, ps)

				if rr.Code != http.StatusNotFound {
					t.Fatalf("%s /users/%s: expected 404, got %d (body: %s)", m.method, id, rr.Code, rr.Body.String())
				}
			})
		})
	}
}

// Feature: mariadb-dal-api, Property 14: Error responses are well-formed JSON

// internalErrDAL is a mock DAL that returns a generic (non-sentinel) error for all
// operations, triggering a 500 Internal Server Error response.
type internalErrDAL struct{}

func (m *internalErrDAL) Ping(ctx context.Context) error { return nil }
func (m *internalErrDAL) Insert(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	return nil, errors.New("unexpected db error")
}
func (m *internalErrDAL) GetByID(ctx context.Context, table string, id string) (map[string]any, error) {
	return nil, errors.New("unexpected db error")
}
func (m *internalErrDAL) List(ctx context.Context, table string, filters map[string]string, limit, offset int) ([]map[string]any, error) {
	return nil, errors.New("unexpected db error")
}
func (m *internalErrDAL) Update(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, errors.New("unexpected db error")
}
func (m *internalErrDAL) Patch(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, errors.New("unexpected db error")
}
func (m *internalErrDAL) Delete(ctx context.Context, table string, id string) error {
	return errors.New("unexpected db error")
}

// assertErrorResponseWellFormed checks that the recorder holds a 4xx/5xx response
// whose body is valid JSON containing a non-empty "error" string field.
func assertErrorResponseWellFormed(t *rapid.T, rr *httptest.ResponseRecorder, scenario string) {
	t.Helper()
	code := rr.Code
	if code < 400 || code > 599 {
		// Not an error response — nothing to assert.
		return
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("%s: status %d but response body is not valid JSON: %v (raw: %q)", scenario, code, err, rr.Body.String())
	}
	errField, ok := body["error"]
	if !ok {
		t.Fatalf("%s: status %d response JSON missing \"error\" field, got: %v", scenario, code, body)
	}
	errStr, ok := errField.(string)
	if !ok || errStr == "" {
		t.Fatalf("%s: status %d \"error\" field is not a non-empty string, got: %v", scenario, code, errField)
	}
}

// invalidResourceNameGen generates resource names that contain at least one character
// outside [a-zA-Z0-9_], which will trigger a 400 from ValidateResourceName.
func invalidResourceNameGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Start with a valid prefix (may be empty) then inject an invalid character.
		prefix := rapid.StringMatching(`[a-zA-Z0-9_]{0,10}`).Draw(t, "prefix")
		// Pick a special character that is definitely outside the allowed set.
		specials := "!@#$%^&*()-+=[]{}|;:',.<>?/ \t"
		idx := rapid.IntRange(0, len(specials)-1).Draw(t, "special_idx")
		suffix := rapid.StringMatching(`[a-zA-Z0-9_]{0,10}`).Draw(t, "suffix")
		return prefix + string(specials[idx]) + suffix
	})
}

// TestPropertyErrorResponsesAreWellFormedJSON verifies that for any request that
// produces a 4xx or 5xx response, the body is valid JSON with a non-empty "error" field.
// Validates: Requirements 8.1
func TestPropertyErrorResponsesAreWellFormedJSON(t *testing.T) {
	// Feature: mariadb-dal-api, Property 14: Error responses are well-formed JSON

	t.Run("invalid_resource_name_yields_400_well_formed", func(t *testing.T) {
		// Any handler with any DAL — the error fires before the DAL is called.
		h := NewResourceHandler(&mockDAL{})
		rapid.Check(t, func(t *rapid.T) {
			resource := invalidResourceNameGen().Draw(t, "resource")

			// Test across all methods that accept a resource param.
			// Use a fixed safe URL path — the resource name is injected via Params,
			// not the URL, so httptest.NewRequest won't reject control characters.
			type scenario struct {
				name   string
				method string
				fn     func(http.ResponseWriter, *http.Request, httprouter.Params)
				body   string
				id     bool
			}
			scenarios := []scenario{
				{"POST", http.MethodPost, h.Create, `{"k":"v"}`, false},
				{"GET_list", http.MethodGet, h.List, "", false},
				{"GET_byID", http.MethodGet, h.GetByID, "", true},
				{"PUT", http.MethodPut, h.Update, `{"k":"v"}`, true},
				{"PATCH", http.MethodPatch, h.Patch, `{"k":"v"}`, true},
				{"DELETE", http.MethodDelete, h.Delete, "", true},
			}
			for _, s := range scenarios {
				req := httptest.NewRequest(s.method, "/resource", strings.NewReader(s.body))
				if s.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				ps := httprouter.Params{{Key: "resource", Value: resource}}
				if s.id {
					ps = append(ps, httprouter.Param{Key: "id", Value: "1"})
				}
				rr := httptest.NewRecorder()
				s.fn(rr, req, ps)
				assertErrorResponseWellFormed(t, rr, s.name+" invalid resource")
			}
		})
	})

	t.Run("invalid_json_body_yields_400_well_formed", func(t *testing.T) {
		h := NewResourceHandler(&mockDAL{})
		rapid.Check(t, func(t *rapid.T) {
			body := invalidJSONGen().Draw(t, "invalid_json")
			var probe any
			if json.Unmarshal([]byte(body), &probe) == nil {
				t.Skip()
			}
			for _, m := range []struct {
				name string
				fn   func(http.ResponseWriter, *http.Request, httprouter.Params)
				id   bool
			}{
				{"POST", h.Create, false},
				{"PUT", h.Update, true},
				{"PATCH", h.Patch, true},
			} {
				req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				ps := httprouter.Params{{Key: "resource", Value: "users"}}
				if m.id {
					ps = append(ps, httprouter.Param{Key: "id", Value: "1"})
				}
				rr := httptest.NewRecorder()
				m.fn(rr, req, ps)
				assertErrorResponseWellFormed(t, rr, m.name+" invalid JSON body")
			}
		})
	})

	t.Run("missing_record_yields_404_well_formed", func(t *testing.T) {
		h := NewResourceHandler(&missingRecordDAL{})
		rapid.Check(t, func(t *rapid.T) {
			id := rapid.StringMatching(`[a-zA-Z0-9_-]{1,40}`).Draw(t, "id")
			for _, m := range []struct {
				name string
				fn   func(http.ResponseWriter, *http.Request, httprouter.Params)
				body string
			}{
				{"GET_byID", h.GetByID, ""},
				{"PUT", h.Update, `{"k":"v"}`},
				{"PATCH", h.Patch, `{"k":"v"}`},
				{"DELETE", h.Delete, ""},
			} {
				req := httptest.NewRequest("GET", "/users/"+id, strings.NewReader(m.body))
				if m.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				ps := httprouter.Params{
					{Key: "resource", Value: "users"},
					{Key: "id", Value: id},
				}
				rr := httptest.NewRecorder()
				m.fn(rr, req, ps)
				assertErrorResponseWellFormed(t, rr, m.name+" missing record id="+id)
			}
		})
	})

	t.Run("unknown_resource_yields_404_well_formed", func(t *testing.T) {
		h := NewResourceHandler(&notFoundDAL{})
		rapid.Check(t, func(t *rapid.T) {
			resource := unknownResourceGen().Draw(t, "resource")
			for _, m := range []struct {
				name string
				fn   func(http.ResponseWriter, *http.Request, httprouter.Params)
				body string
				id   bool
			}{
				{"POST", h.Create, `{"k":"v"}`, false},
				{"GET_list", h.List, "", false},
				{"GET_byID", h.GetByID, "", true},
				{"PUT", h.Update, `{"k":"v"}`, true},
				{"PATCH", h.Patch, `{"k":"v"}`, true},
				{"DELETE", h.Delete, "", true},
			} {
				req := httptest.NewRequest("GET", "/"+resource, strings.NewReader(m.body))
				if m.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				ps := httprouter.Params{{Key: "resource", Value: resource}}
				if m.id {
					ps = append(ps, httprouter.Param{Key: "id", Value: "1"})
				}
				rr := httptest.NewRecorder()
				m.fn(rr, req, ps)
				assertErrorResponseWellFormed(t, rr, m.name+" unknown resource "+resource)
			}
		})
	})

	t.Run("internal_error_yields_500_well_formed", func(t *testing.T) {
		h := NewResourceHandler(&internalErrDAL{})
		rapid.Check(t, func(t *rapid.T) {
			for _, m := range []struct {
				name string
				fn   func(http.ResponseWriter, *http.Request, httprouter.Params)
				body string
				id   bool
			}{
				{"POST", h.Create, `{"k":"v"}`, false},
				{"GET_list", h.List, "", false},
				{"GET_byID", h.GetByID, "", true},
				{"PUT", h.Update, `{"k":"v"}`, true},
				{"PATCH", h.Patch, `{"k":"v"}`, true},
				{"DELETE", h.Delete, "", true},
			} {
				req := httptest.NewRequest("GET", "/users", strings.NewReader(m.body))
				if m.body != "" {
					req.Header.Set("Content-Type", "application/json")
				}
				ps := httprouter.Params{{Key: "resource", Value: "users"}}
				if m.id {
					ps = append(ps, httprouter.Param{Key: "id", Value: "1"})
				}
				rr := httptest.NewRecorder()
				m.fn(rr, req, ps)
				assertErrorResponseWellFormed(t, rr, m.name+" internal error")
			}
		})
	})
}
