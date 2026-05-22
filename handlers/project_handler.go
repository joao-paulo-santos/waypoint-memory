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

type ProjectHandler struct {
	DB      *sql.DB
	Service *services.ProjectService
}

func NewProjectHandler(db *sql.DB, svc *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{DB: db, Service: svc}
}

func (h *ProjectHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	project, err := h.Service.Create(req, getUserID(r))
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	projects, err := h.Service.List(getUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"projects": projects})
}

func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	detail, err := h.Service.GetProjectDetail(h.DB, id)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(detail)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	var req models.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	project, err := h.Service.Update(id, req)
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.Service.Delete(id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case services.ErrProjectNotFound:
		code = http.StatusNotFound
	case services.ErrProjectAlreadyRegistered:
		code = http.StatusConflict
	case services.ErrBucketNotFound:
		code = http.StatusNotFound
	case services.ErrBucketHasTasks:
		code = http.StatusConflict
	case services.ErrTaskNotFound:
		code = http.StatusNotFound
	case services.ErrLabelNotFound:
		code = http.StatusNotFound
	case services.ErrLabelDuplicate:
		code = http.StatusConflict
	case services.ErrNoDoneBucket:
		code = http.StatusBadRequest
	case services.ErrEmptyDoneBucket:
		code = http.StatusBadRequest
	case services.ErrBucketOnlyDone:
		code = http.StatusConflict
	case services.ErrCommentBodyEmpty:
		code = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
