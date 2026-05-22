package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
)

type UpcomingHandler struct {
	DB         *sql.DB
	ProjectSvc *services.ProjectService
	ContactSvc *services.ContactService
	EventSvc   *services.EventService
}

func NewUpcomingHandler(db *sql.DB, projectSvc *services.ProjectService, contactSvc *services.ContactService, eventSvc *services.EventService) *UpcomingHandler {
	return &UpcomingHandler{DB: db, ProjectSvc: projectSvc, ContactSvc: contactSvc, EventSvc: eventSvc}
}

func (h *UpcomingHandler) GetUpcoming(w http.ResponseWriter, r *http.Request) {
	ownerID := getUserID(r)
	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	future := now.AddDate(0, 0, days).Format("2006-01-02")

	var items []models.UpcomingItem

	projects, _ := h.ProjectSvc.List(ownerID)
	for _, p := range projects {
		if p.IsArchived {
			continue
		}
		rows, err := h.DB.Query(
			`SELECT id, title, done, due_date FROM tasks
			 WHERE project_id = $1 AND due_date IS NOT NULL AND due_date >= $2 AND due_date <= $3`,
			p.ID, today, future,
		)
		if err != nil {
			continue
		}
		for rows.Next() {
			var id int64
			var title string
			var done bool
			var dueDate string
			if rows.Scan(&id, &title, &done, &dueDate) == nil {
				pid := p.ID
				tid := id
				items = append(items, models.UpcomingItem{
					Type:        "task",
					Date:        dueDate,
					Title:       title,
					ProjectID:   &pid,
					ProjectName: p.Name,
					TaskID:      &tid,
					Done:        done,
					DaysUntil:   daysBetween(now, dueDate),
				})
			}
		}
		rows.Close()
	}

	contacts, _ := h.ContactSvc.List(ownerID)
	for _, c := range contacts {
		if c.BirthdayDate == "" {
			continue
		}
		parts := strings.SplitN(c.BirthdayDate, "-", 2)
		if len(parts) != 2 {
			continue
		}
		month, err1 := strconv.Atoi(parts[0])
		day, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}
		occ := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if occ.Before(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)) {
			occ = time.Date(now.Year()+1, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		}
		name := strings.TrimSpace(c.FirstName + " " + c.LastName)
		cid := c.ID
		item := models.UpcomingItem{
			Type:      "birthday",
			Date:      occ.Format("2006-01-02"),
			Title:     name,
			ContactID: &cid,
			Subtitle:  "Birthday",
			DaysUntil: daysBetween(now, occ.Format("2006-01-02")),
		}
		if c.BirthdayYear.Valid {
			age := occ.Year() - int(c.BirthdayYear.Int64)
			item.AgeTurning = &age
		}
		items = append(items, item)
	}

	from := now
	to := now.AddDate(0, 0, days)
	calEvents, _ := h.EventSvc.GetOccurrencesInRange(ownerID, from, to)
	for _, e := range calEvents {
		items = append(items, models.UpcomingItem{
			Type:     "event",
			Date:     e.Date,
			Title:    e.Title,
			Category: e.Category,
			DaysUntil: daysBetween(now, e.Date),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Date != items[j].Date {
			return items[i].Date < items[j].Date
		}
		return items[i].Type < items[j].Type
	})

	if len(items) > 50 {
		items = items[:50]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"items": items})
}

func daysBetween(now time.Time, dateStr string) int {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0
	}
	return int(d.Sub(today).Hours() / 24)
}
