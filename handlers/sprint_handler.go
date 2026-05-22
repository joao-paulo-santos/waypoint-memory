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

type SprintHandler struct {
	DB         *sql.DB
	ProjectSvc *services.ProjectService
	SprintSvc  *services.SprintService
}

func NewSprintHandler(db *sql.DB, projectSvc *services.ProjectService) *SprintHandler {
	return &SprintHandler{
		DB:         db,
		ProjectSvc: projectSvc,
		SprintSvc:  &services.SprintService{},
	}
}

func (h *SprintHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/end", h.EndSprint)
	r.Get("/", h.ListSprints)
	r.Get("/{sid}", h.GetSprint)
	return r
}

func (h *SprintHandler) getProjectID(r *http.Request) (int64, error) {
	projectIDStr := chi.URLParam(r, "projectId")
	return strconv.ParseInt(projectIDStr, 10, 64)
}

func (h *SprintHandler) EndSprint(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	var req models.EndSprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.SprintSvc.EndSprint(h.DB, projectID, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *SprintHandler) ListSprints(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	sprints, err := h.SprintSvc.ListSprints(h.DB, projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"sprints": sprints})
}

func (h *SprintHandler) GetSprint(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	sid, err := strconv.ParseInt(chi.URLParam(r, "sid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid sprint id", http.StatusBadRequest)
		return
	}

	detail, err := h.SprintSvc.GetSprintDetail(h.DB, projectID, sid)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}
