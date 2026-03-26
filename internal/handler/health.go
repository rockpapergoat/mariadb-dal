package handler

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mariadb-dal-api/internal/dal"
	"github.com/mariadb-dal-api/internal/model"
)

// HealthHandler handles GET /health requests.
type HealthHandler struct {
	d dal.DAL
}

// NewHealthHandler creates a new HealthHandler backed by the given DAL.
func NewHealthHandler(d dal.DAL) *HealthHandler {
	return &HealthHandler{d: d}
}

// Handle responds with 200 {"status":"ok"} if the DB is reachable, or 503 on failure.
func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if err := h.d.Ping(r.Context()); err != nil {
		model.WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	model.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
