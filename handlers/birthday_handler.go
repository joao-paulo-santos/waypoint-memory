package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type BirthdayHandler struct {
	Svc *services.BirthdayService
}

func NewBirthdayHandler(svc *services.BirthdayService) *BirthdayHandler {
	return &BirthdayHandler{Svc: svc}
}

func (h *BirthdayHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/upcoming", h.Upcoming)
	r.Post("/", h.Create)
	r.Route("/{bid}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})
	return r
}

func (h *BirthdayHandler) List(w http.ResponseWriter, r *http.Request) {
	birthdays, err := h.Svc.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"birthdays": birthdays})
}

func (h *BirthdayHandler) Upcoming(w http.ResponseWriter, r *http.Request) {
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
	json.NewEncoder(w).Encode(map[string]any{"birthdays": results})
}

func (h *BirthdayHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateBirthdayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	birthday, err := h.Svc.Create(req)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(birthday)
}

func (h *BirthdayHandler) Get(w http.ResponseWriter, r *http.Request) {
	bid, err := strconv.ParseInt(chi.URLParam(r, "bid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid birthday id", http.StatusBadRequest)
		return
	}
	birthday, err := h.Svc.GetByID(bid)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(birthday)
}

func (h *BirthdayHandler) Update(w http.ResponseWriter, r *http.Request) {
	bid, err := strconv.ParseInt(chi.URLParam(r, "bid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid birthday id", http.StatusBadRequest)
		return
	}
	var req models.UpdateBirthdayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	birthday, err := h.Svc.Update(bid, req)
	if err != nil {
		writeCentralError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(birthday)
}

func (h *BirthdayHandler) Delete(w http.ResponseWriter, r *http.Request) {
	bid, err := strconv.ParseInt(chi.URLParam(r, "bid"), 10, 64)
	if err != nil {
		http.Error(w, "invalid birthday id", http.StatusBadRequest)
		return
	}
	if err := h.Svc.Delete(bid); err != nil {
		writeCentralError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
