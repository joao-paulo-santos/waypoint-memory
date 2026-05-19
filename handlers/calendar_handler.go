package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type CalendarHandler struct {
	CalendarSvc *services.CalendarService
}

func NewCalendarHandler(calendarSvc *services.CalendarService) *CalendarHandler {
	return &CalendarHandler{CalendarSvc: calendarSvc}
}

func (h *CalendarHandler) GetCalendar(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	result, err := h.CalendarSvc.GetCalendar(from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *CalendarHandler) GetToday(w http.ResponseWriter, r *http.Request) {
	result, err := h.CalendarSvc.GetToday()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
