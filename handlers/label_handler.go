package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type LabelHandler struct {
	ProjectSvc *services.ProjectService
	LabelSvc   *services.LabelService
}

func NewLabelHandler(projectSvc *services.ProjectService) *LabelHandler {
	return &LabelHandler{
		ProjectSvc: projectSvc,
		LabelSvc:   &services.LabelService{},
	}
}

func (h *LabelHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Delete("/{lid}", h.Delete)
	return r
}

func (h *LabelHandler) List(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	labels, err := h.LabelSvc.ListLabels(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"labels": labels})
}

func (h *LabelHandler) Create(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	var req models.CreateLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	label, err := h.LabelSvc.CreateLabel(db, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(label)
}

func (h *LabelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	lid, err := strconv.ParseInt(chi.URLParam(r, "lid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid label id", http.StatusBadRequest)
		return
	}

	if err := h.LabelSvc.DeleteLabel(db, lid); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *LabelHandler) getProjectDB(r *http.Request) (*sql.DB, func(), error) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return nil, nil, err
	}
	db, err := h.ProjectSvc.GetProjectDB(projectID)
	if err != nil {
		return nil, nil, err
	}
	return db, func() { db.Close() }, nil
}
