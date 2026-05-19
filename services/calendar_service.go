package services

import (
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

type CalendarService struct {
	CentralDB    *sql.DB
	ProjectSvc   *ProjectService
	BirthdaySvc  *BirthdayService
	RecurringSvc *RecurringEventService
}

func NewCalendarService(
	centralDB *sql.DB,
	projectSvc *ProjectService,
	birthdaySvc *BirthdayService,
	recurringSvc *RecurringEventService,
) *CalendarService {
	return &CalendarService{
		CentralDB:    centralDB,
		ProjectSvc:   projectSvc,
		BirthdaySvc:  birthdaySvc,
		RecurringSvc: recurringSvc,
	}
}

func (s *CalendarService) GetCalendar(from, to string) (*models.CalendarResponse, error) {
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

	daysAhead := int(toDate.Sub(fromDate).Hours()/24) + 1
	var events []models.CalendarEvent

	taskEvents, err := s.getTaskEvents(fromDate, toDate)
	if err != nil {
		return nil, err
	}
	events = append(events, taskEvents...)

	birthdayEvents, err := s.getBirthdayEvents(daysAhead)
	if err != nil {
		return nil, err
	}
	events = append(events, birthdayEvents...)

	recurringEvents, err := s.getRecurringEvents(daysAhead)
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

func (s *CalendarService) GetToday() (*models.CalendarResponse, error) {
	today := time.Now().Format("2006-01-02")
	return s.GetCalendar(today, today)
}

func (s *CalendarService) getTaskEvents(from, to time.Time) ([]models.CalendarEvent, error) {
	projects, err := s.ProjectSvc.List()
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

		projDB, err := s.ProjectSvc.GetProjectDB(p.ID)
		if err != nil {
			continue
		}

		rows, err := projDB.Query(
			`SELECT id, title, due_date FROM tasks
			 WHERE due_date IS NOT NULL AND due_date >= ? AND due_date <= ?`,
			fromStr, toStr,
		)
		if err == nil {
			for rows.Next() {
				var id int64
				var title, dueDate string
				if rows.Scan(&id, &title, &dueDate) == nil {
					pid := p.ID
					tid := id
					events = append(events, models.CalendarEvent{
						Type:        "task_due",
						Date:        dueDate,
						Title:       title,
						TaskID:      &tid,
						ProjectID:   &pid,
						ProjectName: p.Name,
					})
				}
			}
			rows.Close()
		}

		rows2, err := projDB.Query(
			`SELECT id, title, done_at, due_date FROM tasks
			 WHERE done = 1 AND done_at IS NOT NULL AND date(done_at) >= ? AND date(done_at) <= ?`,
			fromStr, toStr,
		)
		if err == nil {
			for rows2.Next() {
				var id int64
				var title, doneAt string
				var dueDate sql.NullString
				if rows2.Scan(&id, &title, &doneAt, &dueDate) == nil {
					hadDue := dueDate.Valid && dueDate.String != ""
					pid := p.ID
					tid := id
					dateStr := doneAt
					if len(dateStr) > 10 {
						dateStr = dateStr[:10]
					}
					events = append(events, models.CalendarEvent{
						Type:        "task_done",
						Date:        dateStr,
						Title:       title,
						TaskID:      &tid,
						ProjectID:   &pid,
						ProjectName: p.Name,
						HadDueDate:  &hadDue,
					})
				}
			}
			rows2.Close()
		}

		projDB.Close()
	}

	return events, nil
}

func (s *CalendarService) getBirthdayEvents(daysAhead int) ([]models.CalendarEvent, error) {
	upcoming, err := s.BirthdaySvc.GetUpcoming(daysAhead)
	if err != nil {
		return nil, err
	}

	var events []models.CalendarEvent
	for _, b := range upcoming {
		days := b.DaysUntil
		events = append(events, models.CalendarEvent{
			Type:       "birthday",
			Date:       b.OccurrenceDate,
			Name:       b.Name,
			AgeTurning: b.AgeTurning,
			DaysUntil:  &days,
		})
	}

	return events, nil
}

func (s *CalendarService) getRecurringEvents(daysAhead int) ([]models.CalendarEvent, error) {
	upcoming, err := s.RecurringSvc.GetUpcoming(daysAhead)
	if err != nil {
		return nil, err
	}

	var events []models.CalendarEvent
	for _, e := range upcoming {
		days := e.DaysUntil
		events = append(events, models.CalendarEvent{
			Type:       "recurring_event",
			Date:       e.Date,
			Title:      e.Title,
			Recurrence: e.Recurrence,
			Category:   e.Category,
			DaysUntil:  &days,
		})
	}

	return events, nil
}
