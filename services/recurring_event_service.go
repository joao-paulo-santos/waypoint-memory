package services

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrRecurringEventNotFound = errors.New("recurring event not found")
	ErrInvalidRecurrence      = errors.New("recurrence must be 'yearly' or 'monthly'")
	ErrInvalidDate            = errors.New("yearly date must be MM-DD, monthly date must be DD")
)

type RecurringEventService struct {
	DB *sql.DB
}

func NewRecurringEventService(db *sql.DB) *RecurringEventService {
	return &RecurringEventService{DB: db}
}

func (s *RecurringEventService) List() ([]models.RecurringEvent, error) {
	rows, err := s.DB.Query(
		`SELECT id, title, description, start_date, end_date, date, year,
		        recurrence, category, remind_days_before, created_at
		 FROM recurring_events ORDER BY title`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (s *RecurringEventService) GetByID(id int64) (*models.RecurringEvent, error) {
	e := &models.RecurringEvent{}
	err := s.DB.QueryRow(
		`SELECT id, title, description, start_date, end_date, date, year,
		        recurrence, category, remind_days_before, created_at
		 FROM recurring_events WHERE id = ?`, id,
	).Scan(&e.ID, &e.Title, &e.Description, &e.StartDate, &e.EndDate,
		&e.Date, &e.Year, &e.Recurrence, &e.Category, &e.RemindDaysBefore, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrRecurringEventNotFound
	}
	return e, err
}

func (s *RecurringEventService) Create(req models.CreateRecurringEventRequest) (*models.RecurringEvent, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	recurrence := req.Recurrence
	if recurrence == "" {
		recurrence = "yearly"
	}
	if recurrence != "yearly" && recurrence != "monthly" {
		return nil, ErrInvalidRecurrence
	}

	if err := validateEventDate(req.Date, recurrence); err != nil {
		return nil, err
	}

	remind := req.RemindDaysBefore
	if remind == 0 {
		remind = 3
	}

	var endDate any
	if req.EndDate != nil {
		endDate = *req.EndDate
	}
	var year any
	if req.Year != nil {
		year = *req.Year
	}

	result, err := s.DB.Exec(
		`INSERT INTO recurring_events (title, description, start_date, end_date, date, year, recurrence, category, remind_days_before)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Title, req.Description, req.StartDate, endDate, req.Date, year, recurrence, req.Category, remind,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return s.GetByID(id)
}

func (s *RecurringEventService) Update(id int64, req models.UpdateRecurringEventRequest) (*models.RecurringEvent, error) {
	if _, err := s.GetByID(id); err != nil {
		return nil, err
	}

	sets := []string{}
	args := []any{}

	if req.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *req.Description)
	}
	if req.StartDate != nil {
		sets = append(sets, "start_date = ?")
		args = append(args, *req.StartDate)
	}
	if req.EndDate != nil {
		sets = append(sets, "end_date = ?")
		args = append(args, *req.EndDate)
	}
	if req.Date != nil {
		sets = append(sets, "date = ?")
		args = append(args, *req.Date)
	}
	if req.Year != nil {
		sets = append(sets, "year = ?")
		args = append(args, *req.Year)
	}
	if req.Recurrence != nil {
		sets = append(sets, "recurrence = ?")
		args = append(args, *req.Recurrence)
	}
	if req.Category != nil {
		sets = append(sets, "category = ?")
		args = append(args, *req.Category)
	}
	if req.RemindDaysBefore != nil {
		sets = append(sets, "remind_days_before = ?")
		args = append(args, *req.RemindDaysBefore)
	}

	if len(sets) == 0 {
		return s.GetByID(id)
	}

	args = append(args, id)
	query := "UPDATE recurring_events SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := s.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *RecurringEventService) Delete(id int64) error {
	result, err := s.DB.Exec("DELETE FROM recurring_events WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrRecurringEventNotFound
	}
	return nil
}

func (s *RecurringEventService) GetUpcoming(daysAhead int) ([]models.UpcomingEvent, error) {
	if daysAhead <= 0 {
		daysAhead = 30
	}

	events, err := s.List()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var results []models.UpcomingEvent

	for _, e := range events {
		nextDates := getNextOccurrences(e, now, daysAhead)
		for _, d := range nextDates {
			daysUntil := int(d.Sub(now).Hours() / 24)
			results = append(results, models.UpcomingEvent{
				ID:         e.ID,
				Title:      e.Title,
				Date:       d.Format("2006-01-02"),
				Recurrence: e.Recurrence,
				Category:   e.Category,
				DaysUntil:  daysUntil,
			})
		}
	}

	return results, nil
}

func getNextOccurrences(e models.RecurringEvent, now time.Time, daysAhead int) []time.Time {
	var results []time.Time
	end := now.AddDate(0, 0, daysAhead)

	startDate, _ := time.Parse("2006-01-02", e.StartDate)
	var endDate *time.Time
	if e.EndDate != nil {
		ed, _ := time.Parse("2006-01-02", *e.EndDate)
		endDate = &ed
	}

	switch e.Recurrence {
	case "yearly":
		month, err := strconv.Atoi(e.Date[0:2])
		if err != nil {
			return nil
		}
		day, err := strconv.Atoi(e.Date[3:5])
		if err != nil {
			return nil
		}
		for year := now.Year(); year <= end.Year()+1; year++ {
			d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			if !startDate.IsZero() && d.Before(startDate) {
				continue
			}
			if endDate != nil && d.After(*endDate) {
				continue
			}
			if d.After(end) {
				break
			}
			if !d.Before(now) {
				results = append(results, d)
			}
		}

	case "monthly":
		day, err := strconv.Atoi(e.Date[0:2])
		if err != nil {
			return nil
		}
		d := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, time.UTC)
		if d.Before(now) {
			d = d.AddDate(0, 1, 0)
		}
		for !d.After(end) {
			if !startDate.IsZero() && d.Before(startDate) {
				d = d.AddDate(0, 1, 0)
				continue
			}
			if endDate != nil && d.After(*endDate) {
				break
			}
			results = append(results, d)
			d = d.AddDate(0, 1, 0)
		}
	}

	return results
}

func validateEventDate(date, recurrence string) error {
	if recurrence == "yearly" {
		if len(date) != 5 || date[2] != '-' {
			return ErrInvalidDate
		}
	} else if recurrence == "monthly" {
		if len(date) != 2 {
			return ErrInvalidDate
		}
	}
	return nil
}

func scanEvents(rows *sql.Rows) ([]models.RecurringEvent, error) {
	var events []models.RecurringEvent
	for rows.Next() {
		var e models.RecurringEvent
		if err := rows.Scan(&e.ID, &e.Title, &e.Description, &e.StartDate, &e.EndDate,
			&e.Date, &e.Year, &e.Recurrence, &e.Category, &e.RemindDaysBefore, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
