package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type ActivityHandler struct {
	ProjectSvc  *services.ProjectService
	ActivitySvc *services.ActivityService
}

func NewActivityHandler(projectSvc *services.ProjectService, activitySvc *services.ActivityService) *ActivityHandler {
	return &ActivityHandler{
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

	projDB, err := h.ProjectSvc.GetProjectDB(projectID)
	if err != nil {
		writeError(w, err)
		return
	}
	defer projDB.Close()

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	entries, err := h.ActivitySvc.GetProjectActivity(projDB, limit)
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

	entries, err := h.ActivitySvc.GetGlobalActivity(h.ProjectSvc.CentralDB, h.ProjectSvc, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"activity": entries})
}
