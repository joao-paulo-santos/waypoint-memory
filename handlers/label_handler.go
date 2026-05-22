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
	DB         *sql.DB
	ProjectSvc *services.ProjectService
	LabelSvc   *services.LabelService
}

func NewLabelHandler(db *sql.DB, projectSvc *services.ProjectService, activitySvc *services.ActivityService) *LabelHandler {
	return &LabelHandler{
		DB:         db,
		ProjectSvc: projectSvc,
		LabelSvc:   &services.LabelService{Activity: activitySvc},
	}
}

func (h *LabelHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Delete("/{lid}", h.Delete)
	return r
}

func (h *LabelHandler) getProjectID(r *http.Request) (int64, error) {
	projectIDStr := chi.URLParam(r, "projectId")
	return strconv.ParseInt(projectIDStr, 10, 64)
}

func (h *LabelHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	labels, err := h.LabelSvc.ListLabels(h.DB, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"labels": labels})
}

func (h *LabelHandler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req models.CreateLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	label, err := h.LabelSvc.CreateLabel(h.DB, projectID, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(label)
}

func (h *LabelHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	lid, err := strconv.ParseInt(chi.URLParam(r, "lid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid label id", http.StatusBadRequest)
		return
	}

	if err := h.LabelSvc.DeleteLabel(h.DB, projectID, lid); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
