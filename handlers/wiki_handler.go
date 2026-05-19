package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type WikiHandler struct {
	ProjectSvc *services.ProjectService
	WikiSvc    *services.WikiService
}

func NewWikiHandler(projectSvc *services.ProjectService) *WikiHandler {
	return &WikiHandler{
		ProjectSvc: projectSvc,
		WikiSvc:    services.NewWikiService(),
	}
}

func (h *WikiHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListWikiPages)
	r.Get("/docs", h.ListDocsPages)
	r.Get("/docs/{slug:.+}", h.ReadDocsPage)
	r.Get("/{slug:.+}", h.ReadWikiPage)
	return r
}

func (h *WikiHandler) ListWikiPages(w http.ResponseWriter, r *http.Request) {
	project, err := h.getProject(r)
	if err != nil {
		writeError(w, err)
		return
	}

	waypointDir := filepath.Join(project.Path, ".waypoint")
	tree, err := h.WikiSvc.ListWikiPages(waypointDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"pages": tree})
}

func (h *WikiHandler) ReadWikiPage(w http.ResponseWriter, r *http.Request) {
	project, err := h.getProject(r)
	if err != nil {
		writeError(w, err)
		return
	}

	slug := chi.URLParam(r, "slug")
	waypointDir := filepath.Join(project.Path, ".waypoint")
	page, err := h.WikiSvc.ReadWikiPage(waypointDir, slug)
	if err != nil {
		writeWikiError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

func (h *WikiHandler) ListDocsPages(w http.ResponseWriter, r *http.Request) {
	project, err := h.getProject(r)
	if err != nil {
		writeError(w, err)
		return
	}

	tree, err := h.WikiSvc.ListDocsPages(project.Path, project.DocsPath)
	if err != nil {
		writeWikiError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"pages": tree})
}

func (h *WikiHandler) ReadDocsPage(w http.ResponseWriter, r *http.Request) {
	project, err := h.getProject(r)
	if err != nil {
		writeError(w, err)
		return
	}

	slug := chi.URLParam(r, "slug")
	page, err := h.WikiSvc.ReadDocsPage(project.Path, project.DocsPath, slug)
	if err != nil {
		writeWikiError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

func (h *WikiHandler) getProject(r *http.Request) (*models.Project, error) {
	projectIDStr := chi.URLParam(r, "projectId")
	projectID, err := strconv.ParseInt(projectIDStr, 10, 64)
	if err != nil {
		return nil, err
	}
	return h.ProjectSvc.GetByID(projectID)
}

func writeWikiError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case services.ErrWikiNotFound:
		code = http.StatusNotFound
	case services.ErrDocsNotEnabled:
		code = http.StatusBadRequest
	case services.ErrPathTraversal:
		code = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
