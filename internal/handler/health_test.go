package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
)

// mockDAL is a minimal DAL implementation for testing the health handler.
type mockDAL struct {
	pingErr error
}

func (m *mockDAL) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockDAL) Insert(ctx context.Context, table string, data map[string]any) (map[string]any, error) {
	return nil, nil
}

func (m *mockDAL) GetByID(ctx context.Context, table string, id string) (map[string]any, error) {
	return nil, nil
}

func (m *mockDAL) List(ctx context.Context, table string, filters map[string]string, limit, offset int) ([]map[string]any, error) {
	return nil, nil
}

func (m *mockDAL) Update(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, nil
}

func (m *mockDAL) Patch(ctx context.Context, table string, id string, data map[string]any) (map[string]any, error) {
	return nil, nil
}

func (m *mockDAL) Delete(ctx context.Context, table string, id string) error {
	return nil
}

func TestHealthHandler_DBUp(t *testing.T) {
	h := NewHealthHandler(&mockDAL{pingErr: nil})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Handle(w, req, httprouter.Params{})

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected body {\"status\":\"ok\"}, got %v", body)
	}
}

func TestHealthHandler_DBDown(t *testing.T) {
	dbErr := errors.New("connection refused")
	h := NewHealthHandler(&mockDAL{pingErr: dbErr})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Handle(w, req, httprouter.Params{})

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if _, ok := body["error"]; !ok {
		t.Errorf("expected response body to contain \"error\" field, got %v", body)
	}

	if body["error"] != dbErr.Error() {
		t.Errorf("expected error message %q, got %q", dbErr.Error(), body["error"])
	}
}
