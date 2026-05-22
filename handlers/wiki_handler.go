package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type WikiHandler struct {
	DB         *sql.DB
	ProjectSvc *services.ProjectService
	WikiSvc    *services.WikiService
}

func NewWikiHandler(db *sql.DB, projectSvc *services.ProjectService, wikiSvc *services.WikiService) *WikiHandler {
	return &WikiHandler{
		DB:         db,
		ProjectSvc: projectSvc,
		WikiSvc:    wikiSvc,
	}
}

func (h *WikiHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListWikiPages)
	r.Get("/{slug:.+}", h.ReadWikiPage)
	r.Put("/{slug:.+}", h.WriteWikiPage)
	r.Delete("/{slug:.+}", h.DeleteWikiPage)
	return r
}

func (h *WikiHandler) getProjectID(r *http.Request) (int64, error) {
	projectIDStr := chi.URLParam(r, "projectId")
	return strconv.ParseInt(projectIDStr, 10, 64)
}

func (h *WikiHandler) ListWikiPages(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	tree, err := h.WikiSvc.ListWikiPages(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"pages": tree})
}

func (h *WikiHandler) ReadWikiPage(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	pageSlug := chi.URLParam(r, "slug")
	page, err := h.WikiSvc.ReadWikiPage(projectID, pageSlug)
	if err != nil {
		writeWikiError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(page)
}

func (h *WikiHandler) WriteWikiPage(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	pageSlug := chi.URLParam(r, "slug")
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.WikiSvc.WriteWikiPage(projectID, pageSlug, req.Content); err != nil {
		writeWikiError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WikiHandler) DeleteWikiPage(w http.ResponseWriter, r *http.Request) {
	projectID, err := h.getProjectID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	pageSlug := chi.URLParam(r, "slug")
	if err := h.WikiSvc.DeleteWikiPage(projectID, pageSlug); err != nil {
		writeWikiError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeWikiError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case services.ErrWikiNotFound:
		code = http.StatusNotFound
	case services.ErrPathTraversal:
		code = http.StatusBadRequest
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
