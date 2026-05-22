package services

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

type CalendarService struct {
	DB         *sql.DB
	ProjectSvc *ProjectService
	ContactSvc *ContactService
	EventSvc   *EventService
}

func NewCalendarService(
	db *sql.DB,
	projectSvc *ProjectService,
	contactSvc *ContactService,
	eventSvc *EventService,
) *CalendarService {
	return &CalendarService{
		DB:         db,
		ProjectSvc: projectSvc,
		ContactSvc: contactSvc,
		EventSvc:   eventSvc,
	}
}

func (s *CalendarService) GetCalendar(ownerID int64, from, to string) (*models.CalendarResponse, error) {
	if from == "" || to == "" {
		now := time.Now()
		if from == "" {
			from = now.Format("2006-01-02")
		}
		if to == "" {
			to = now.AddDate(0, 0, 7).Format("2006-01-02")
		}
	}

	fromDate, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("invalid from date: %w", err)
	}
	toDate, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("invalid to date: %w", err)
	}

	var events []models.CalendarEvent

	taskEvents, err := s.getTaskEvents(ownerID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	events = append(events, taskEvents...)

	birthdayEvents, err := s.getBirthdayEvents(ownerID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	events = append(events, birthdayEvents...)

	recurringEvents, err := s.getRecurringEvents(ownerID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	events = append(events, recurringEvents...)

	var filtered []models.CalendarEvent
	for _, e := range events {
		if e.Date >= from && e.Date <= to {
			filtered = append(filtered, e)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Date < filtered[j].Date
	})

	return &models.CalendarResponse{
		Events: filtered,
		From:   from,
		To:     to,
	}, nil
}

func (s *CalendarService) GetToday(ownerID int64) (*models.CalendarResponse, error) {
	today := time.Now().Format("2006-01-02")
	return s.GetCalendar(ownerID, today, today)
}

func (s *CalendarService) getTaskEvents(ownerID int64, from, to time.Time) ([]models.CalendarEvent, error) {
	projects, err := s.ProjectSvc.List(ownerID)
	if err != nil {
		return nil, err
	}

	var events []models.CalendarEvent
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	for _, p := range projects {
		if p.IsArchived {
			continue
		}

		rows, err := s.DB.Query(
			`SELECT id, title, done, due_date, done_at FROM tasks
			 WHERE project_id = $1
			 AND (
				 (due_date IS NOT NULL AND due_date >= $2 AND due_date <= $3)
				 OR
				 (done = TRUE AND done_at IS NOT NULL AND due_date IS NULL AND done_at >= $2::timestamp AND done_at <= $3::timestamp)
			 )`,
			p.ID, fromStr, toStr,
		)
		if err != nil {
		} else {
			for rows.Next() {
				var id int64
				var title string
				var done bool
				var dueDate sql.NullString
				var doneAt sql.NullTime
				if scanErr := rows.Scan(&id, &title, &done, &dueDate, &doneAt); scanErr != nil {
					fmt.Printf("calendar task scan error: %v\n", scanErr)
					continue
				} else {
					pid := p.ID
					tid := id
					dateStr := ""
					if dueDate.Valid && dueDate.String != "" {
						dateStr = dueDate.String
					} else if doneAt.Valid {
						dateStr = doneAt.Time.Format("2006-01-02")
					}
					if dateStr == "" {
						continue
					}
					events = append(events, models.CalendarEvent{
						Type:        "task_due",
						Date:        dateStr,
						Title:       title,
						TaskID:      &tid,
						ProjectID:   &pid,
						ProjectName: p.Name,
						Done:        done,
					})
				}
			}
			rows.Close()
		}
	}

	return events, nil
}

func (s *CalendarService) getBirthdayEvents(ownerID int64, from, to time.Time) ([]models.CalendarEvent, error) {
	contacts, err := s.ContactSvc.List(ownerID)
	if err != nil {
		return nil, err
	}

	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	var events []models.CalendarEvent
	for _, c := range contacts {
		if c.BirthdayDate == "" {
			continue
		}
		mmdd := c.BirthdayDate
		name := strings.TrimSpace(c.FirstName + " " + c.LastName)
		for y := from.Year() - 1; y <= to.Year()+1; y++ {
			dateStr := fmt.Sprintf("%04d-%s", y, mmdd)
			if dateStr < fromStr || dateStr > toStr {
				continue
			}
			var ageTurning *int
			if c.BirthdayYear.Valid {
				age := y - int(c.BirthdayYear.Int64)
				ageTurning = &age
			}
			events = append(events, models.CalendarEvent{
				Type:       "birthday",
				Date:       dateStr,
				Title:      "Birthday: " + name,
				Name:       name,
				AgeTurning: ageTurning,
			})
		}
	}

	return events, nil
}

func (s *CalendarService) getRecurringEvents(ownerID int64, from, to time.Time) ([]models.CalendarEvent, error) {
	return s.EventSvc.GetOccurrencesInRange(ownerID, from, to)
}
