package model

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse is the standard error envelope for all 4xx/5xx responses.
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteJSON sets Content-Type: application/json, writes the status code, and encodes v as JSON.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// WriteError writes a JSON error response with the given status code and message.
func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, ErrorResponse{Error: msg})
}
