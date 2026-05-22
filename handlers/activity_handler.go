package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type ActivityHandler struct {
	DB          *sql.DB
	ProjectSvc  *services.ProjectService
	ActivitySvc *services.ActivityService
}

func NewActivityHandler(db *sql.DB, projectSvc *services.ProjectService, activitySvc *services.ActivityService) *ActivityHandler {
	return &ActivityHandler{
		DB:          db,
		ProjectSvc:  projectSvc,
		ActivitySvc: activitySvc,
	}
}

func (h *ActivityHandler) ProjectActivity(w http.ResponseWriter, r *http.Request) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid project id", http.StatusBadRequest)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	entries, err := h.ActivitySvc.GetProjectActivity(h.DB, projectID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"activity": entries})
}

func (h *ActivityHandler) GlobalActivity(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	entries, err := h.ActivitySvc.GetGlobalActivity(h.DB, h.ProjectSvc, getUserID(r), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"activity": entries})
}
