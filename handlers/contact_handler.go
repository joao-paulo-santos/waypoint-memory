package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type ContactHandler struct {
	Svc *services.ContactService
}

func NewContactHandler(svc *services.ContactService) *ContactHandler {
	return &ContactHandler{Svc: svc}
}

func (h *ContactHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Route("/{cid}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	contacts, err := h.Svc.List(getUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"contacts": contacts})
}

func (h *ContactHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	contact, err := h.Svc.Create(req, getUserID(r))
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(contact)
}

func (h *ContactHandler) Get(w http.ResponseWriter, r *http.Request) {
	cid, err := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid contact id", http.StatusBadRequest)
		return
	}
	contact, err := h.Svc.GetByID(cid, getUserID(r))
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contact)
}

func (h *ContactHandler) Update(w http.ResponseWriter, r *http.Request) {
	cid, err := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid contact id", http.StatusBadRequest)
		return
	}
	var req models.UpdateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	contact, err := h.Svc.Update(cid, getUserID(r), req)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contact)
}

func (h *ContactHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cid, err := strconv.ParseInt(chi.URLParam(r, "cid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid contact id", http.StatusBadRequest)
		return
	}
	if err := h.Svc.Delete(cid, getUserID(r)); err != nil {
		writeCentralError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeCentralError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case services.ErrContactNotFound:
		code = http.StatusNotFound
	case services.ErrEventNotFound:
		code = http.StatusNotFound
	case services.ErrTokenNotFound:
		code = http.StatusNotFound
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
