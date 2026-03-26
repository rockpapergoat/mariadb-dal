package handler

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mariadb-dal-api/internal/dal"
	"github.com/mariadb-dal-api/internal/model"
)

// ResourceHandler handles CRUD operations for any resource (table).
type ResourceHandler struct {
	d dal.DAL
}

// NewResourceHandler creates a new ResourceHandler backed by the given DAL.
func NewResourceHandler(d dal.DAL) *ResourceHandler {
	return &ResourceHandler{d: d}
}

// mapDALError maps DAL sentinel errors to HTTP status codes and writes the error response.
func mapDALError(w http.ResponseWriter, err error) {
	switch err {
	case dal.ErrNotFound:
		model.WriteError(w, http.StatusNotFound, err.Error())
	case dal.ErrConflict:
		model.WriteError(w, http.StatusConflict, err.Error())
	default:
		model.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

// Create handles POST /:resource — inserts a new record and returns 201 with the created record.
func (h *ResourceHandler) Create(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	resource := ps.ByName("resource")
	if err := dal.ValidateResourceName(resource); err != nil {
		model.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		model.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	record, err := h.d.Insert(r.Context(), resource, data)
	if err != nil {
		mapDALError(w, err)
		return
	}

	model.WriteJSON(w, http.StatusCreated, record)
}

// GetByID handles GET /:resource/:id — retrieves a single record by ID and returns 200.
func (h *ResourceHandler) GetByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	resource := ps.ByName("resource")
	if err := dal.ValidateResourceName(resource); err != nil {
		model.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	id := ps.ByName("id")
	record, err := h.d.GetByID(r.Context(), resource, id)
	if err != nil {
		mapDALError(w, err)
		return
	}

	model.WriteJSON(w, http.StatusOK, record)
}

// List handles GET /:resource — lists records with optional filters, limit, and offset.
func (h *ResourceHandler) List(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	resource := ps.ByName("resource")
	if err := dal.ValidateResourceName(resource); err != nil {
		model.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	q := r.URL.Query()
	limit, offset, err := dal.ParseLimitOffset(q.Get("limit"), q.Get("offset"))
	if err != nil {
		model.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	filters := make(map[string]string)
	for key, vals := range q {
		if key == "limit" || key == "offset" {
			continue
		}
		filters[key] = vals[0]
	}

	records, err := h.d.List(r.Context(), resource, filters, limit, offset)
	if err != nil {
		mapDALError(w, err)
		return
	}

	if records == nil {
		records = []map[string]any{}
	}

	model.WriteJSON(w, http.StatusOK, records)
}

// Update handles PUT /:resource/:id — fully replaces a record and returns 200 with the updated record.
func (h *ResourceHandler) Update(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	resource := ps.ByName("resource")
	if err := dal.ValidateResourceName(resource); err != nil {
		model.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		model.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	id := ps.ByName("id")
	record, err := h.d.Update(r.Context(), resource, id, data)
	if err != nil {
		mapDALError(w, err)
		return
	}

	model.WriteJSON(w, http.StatusOK, record)
}

// Patch handles PATCH /:resource/:id — partially updates a record and returns 200 with the updated record.
func (h *ResourceHandler) Patch(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	resource := ps.ByName("resource")
	if err := dal.ValidateResourceName(resource); err != nil {
		model.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		model.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	id := ps.ByName("id")
	record, err := h.d.Patch(r.Context(), resource, id, data)
	if err != nil {
		mapDALError(w, err)
		return
	}

	model.WriteJSON(w, http.StatusOK, record)
}

// Delete handles DELETE /:resource/:id — removes a record and returns 204 with no body.
func (h *ResourceHandler) Delete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	resource := ps.ByName("resource")
	if err := dal.ValidateResourceName(resource); err != nil {
		model.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	id := ps.ByName("id")
	if err := h.d.Delete(r.Context(), resource, id); err != nil {
		mapDALError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
