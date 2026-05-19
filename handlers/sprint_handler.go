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
	ProjectSvc *services.ProjectService
	SprintSvc  *services.SprintService
}

func NewSprintHandler(projectSvc *services.ProjectService) *SprintHandler {
	return &SprintHandler{
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

func (h *SprintHandler) EndSprint(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	var req models.EndSprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.SprintSvc.EndSprint(db, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(result)
}

func (h *SprintHandler) ListSprints(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	sprints, err := h.SprintSvc.ListSprints(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"sprints": sprints})
}

func (h *SprintHandler) GetSprint(w http.ResponseWriter, r *http.Request) {
	db, close, err := h.getProjectDB(r)
	if err != nil {
		writeError(w, err)
		return
	}
	defer close()

	sid, err := strconv.ParseInt(chi.URLParam(r, "sid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid sprint id", http.StatusBadRequest)
		return
	}

	detail, err := h.SprintSvc.GetSprintDetail(db, sid)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *SprintHandler) getProjectDB(r *http.Request) (*sql.DB, func(), error) {
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
