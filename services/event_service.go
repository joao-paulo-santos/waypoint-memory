package services

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrEventNotFound = errors.New("event not found")
)

type EventService struct {
	DB *sql.DB
}

func NewEventService(db *sql.DB) *EventService {
	return &EventService{DB: db}
}

const eventColumns = `id, title, description, date, time, recurrence, category, remind_days_before, created_at`

func scanEvent(scanner interface{ Scan(...interface{}) error }) (*models.Event, error) {
	e := &models.Event{}
	err := scanner.Scan(&e.ID, &e.Title, &e.Description, &e.Date, &e.Time,
		&e.Recurrence, &e.Category, &e.RemindDaysBefore, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *EventService) List(ownerID int64) ([]models.Event, error) {
	rows, err := s.DB.Query(
		`SELECT `+eventColumns+` FROM events WHERE owner_id = $1 ORDER BY date, time`, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *e)
	}
	return events, nil
}

func (s *EventService) GetByID(id int64, ownerID int64) (*models.Event, error) {
	e, err := scanEvent(s.DB.QueryRow(
		`SELECT `+eventColumns+` FROM events WHERE id = $1 AND owner_id = $2`, id, ownerID,
	))
	if err == sql.ErrNoRows {
		return nil, ErrEventNotFound
	}
	return e, err
}

func (s *EventService) Create(req models.CreateEventRequest, ownerID int64) (*models.Event, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}
	if req.Date == "" {
		return nil, errors.New("date is required")
	}

	remind := req.RemindDaysBefore
	if remind == 0 {
		remind = 3
	}

	var id int64
	err := s.DB.QueryRow(
		`INSERT INTO events (owner_id, title, description, date, time, recurrence, category, remind_days_before)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		ownerID, req.Title, req.Description, req.Date, req.Time, req.Recurrence, req.Category, remind,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id, ownerID)
}

func (s *EventService) Update(id int64, ownerID int64, req models.UpdateEventRequest) (*models.Event, error) {
	if _, err := s.GetByID(id, ownerID); err != nil {
		return nil, err
	}

	sets := []string{}
	args := []any{}
	paramIdx := 1

	if req.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", paramIdx))
		args = append(args, *req.Title)
		paramIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", paramIdx))
		args = append(args, *req.Description)
		paramIdx++
	}
	if req.Date != nil {
		sets = append(sets, fmt.Sprintf("date = $%d", paramIdx))
		args = append(args, *req.Date)
		paramIdx++
	}
	if req.Time != nil {
		sets = append(sets, fmt.Sprintf("time = $%d", paramIdx))
		args = append(args, *req.Time)
		paramIdx++
	}
	if req.Recurrence != nil {
		sets = append(sets, fmt.Sprintf("recurrence = $%d", paramIdx))
		args = append(args, *req.Recurrence)
		paramIdx++
	}
	if req.Category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", paramIdx))
		args = append(args, *req.Category)
		paramIdx++
	}
	if req.RemindDaysBefore != nil {
		sets = append(sets, fmt.Sprintf("remind_days_before = $%d", paramIdx))
		args = append(args, *req.RemindDaysBefore)
		paramIdx++
	}

	if len(sets) == 0 {
		return s.GetByID(id, ownerID)
	}

	args = append(args, id, ownerID)
	query := "UPDATE events SET " + strings.Join(sets, ", ") + fmt.Sprintf(" WHERE id = $%d AND owner_id = $%d", paramIdx, paramIdx+1)
	if _, err := s.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return s.GetByID(id, ownerID)
}

func (s *EventService) Delete(id int64, ownerID int64) error {
	result, err := s.DB.Exec("DELETE FROM events WHERE id = $1 AND owner_id = $2", id, ownerID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrEventNotFound
	}
	return nil
}

func (s *EventService) GetOccurrencesInRange(ownerID int64, from, to time.Time) ([]models.CalendarEvent, error) {
	events, err := s.List(ownerID)
	if err != nil {
		return nil, err
	}

	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	var results []models.CalendarEvent
	for _, e := range events {
		if e.Recurrence == nil || *e.Recurrence == "" {
			if e.Date >= fromStr && e.Date <= toStr {
				eid := e.ID
				results = append(results, models.CalendarEvent{
					Type:     "event",
					Date:     e.Date,
					Title:    e.Title,
					Category: e.Category,
					EventID:  &eid,
				})
			}
			continue
		}

		occurrences := expandRecurrence(e, from, to)
		for _, dateStr := range occurrences {
			if dateStr >= fromStr && dateStr <= toStr {
				eid := e.ID
				results = append(results, models.CalendarEvent{
					Type:     "event",
					Date:     dateStr,
					Title:    e.Title,
					Category: e.Category,
					EventID:  &eid,
				})
			}
		}
	}

	return results, nil
}

func expandRecurrence(e models.Event, from, to time.Time) []string {
	var results []string

	switch *e.Recurrence {
	case "yearly":
		parts := strings.SplitN(e.Date, "-", 3)
		if len(parts) < 3 {
			return nil
		}
		mmdd := parts[1] + "-" + parts[2]
		for y := from.Year() - 1; y <= to.Year()+1; y++ {
			dateStr := fmt.Sprintf("%04d-%s", y, mmdd)
			if dateStr >= from.Format("2006-01-02") && dateStr <= to.Format("2006-01-02") {
				results = append(results, dateStr)
			}
		}

	case "monthly":
		parts := strings.SplitN(e.Date, "-", 3)
		if len(parts) < 3 {
			return nil
		}
		day := parts[2]
		d := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := to
		for !d.After(end) {
			dayInt, _ := strconv.Atoi(day)
			date := time.Date(d.Year(), d.Month(), dayInt, 0, 0, 0, 0, time.UTC)
			dateStr := date.Format("2006-01-02")
			if dateStr >= from.Format("2006-01-02") && dateStr <= to.Format("2006-01-02") {
				results = append(results, dateStr)
			}
			d = d.AddDate(0, 1, 0)
		}
	}

	sort.Strings(results)
	return results
}
