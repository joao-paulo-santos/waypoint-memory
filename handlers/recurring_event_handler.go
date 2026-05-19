package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type RecurringEventHandler struct {
	Svc *services.RecurringEventService
}

func NewRecurringEventHandler(svc *services.RecurringEventService) *RecurringEventHandler {
	return &RecurringEventHandler{Svc: svc}
}

func (h *RecurringEventHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/upcoming", h.Upcoming)
	r.Post("/", h.Create)
	r.Route("/{eid}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

func (h *RecurringEventHandler) List(w http.ResponseWriter, r *http.Request) {
	events, err := h.Svc.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"events": events})
}

func (h *RecurringEventHandler) Upcoming(w http.ResponseWriter, r *http.Request) {
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}
	results, err := h.Svc.GetUpcoming(days)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"events": results})
}

func (h *RecurringEventHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRecurringEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	event, err := h.Svc.Create(req)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

func (h *RecurringEventHandler) Get(w http.ResponseWriter, r *http.Request) {
	eid, err := strconv.ParseInt(chi.URLParam(r, "eid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	event, err := h.Svc.GetByID(eid)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (h *RecurringEventHandler) Update(w http.ResponseWriter, r *http.Request) {
	eid, err := strconv.ParseInt(chi.URLParam(r, "eid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	var req models.UpdateRecurringEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	event, err := h.Svc.Update(eid, req)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

func (h *RecurringEventHandler) Delete(w http.ResponseWriter, r *http.Request) {
	eid, err := strconv.ParseInt(chi.URLParam(r, "eid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid event id", http.StatusBadRequest)
		return
	}
	if err := h.Svc.Delete(eid); err != nil {
		writeCentralError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
