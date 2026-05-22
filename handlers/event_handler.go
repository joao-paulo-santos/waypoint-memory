package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type EventHandler struct {
	Svc *services.EventService
}

func NewEventHandler(svc *services.EventService) *EventHandler {
	return &EventHandler{Svc: svc}
}

func (h *EventHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Route("/{eid}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

func (h *EventHandler) List(w http.ResponseWriter, r *http.Request) {
	events, err := h.Svc.List(getUserID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"events": events})
}

func (h *EventHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	event, err := h.Svc.Create(req, getUserID(r))
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandler) Get(w http.ResponseWriter, r *http.Request) {
	eid, err := strconv.ParseInt(chi.URLParam(r, "eid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	event, err := h.Svc.GetByID(eid, getUserID(r))
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandler) Update(w http.ResponseWriter, r *http.Request) {
	eid, err := strconv.ParseInt(chi.URLParam(r, "eid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	var req models.UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	event, err := h.Svc.Update(eid, getUserID(r), req)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (h *EventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	eid, err := strconv.ParseInt(chi.URLParam(r, "eid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	if err := h.Svc.Delete(eid, getUserID(r)); err != nil {
		writeCentralError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
