package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCalendarTest(t *testing.T) (*CalendarService, *ProjectService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)

	projectsDir := filepath.Join(tmpDir, "projects")
	os.MkdirAll(projectsDir, 0755)

	projectSvc := NewProjectService(centralDB, projectsDir)
	birthdaySvc := NewBirthdayService(centralDB)
	recurringSvc := NewRecurringEventService(centralDB)
	calendarSvc := NewCalendarService(centralDB, projectSvc, birthdaySvc, recurringSvc)

	return calendarSvc, projectSvc, func() { centralDB.Close() }
}

func TestCalendarService_TaskDueInRange(t *testing.T) {
	calSvc, projectSvc, cleanup := setupCalendarTest(t)
	defer cleanup()

	p, _ := projectSvc.Create(models.CreateProjectRequest{Name: "CalTest"})
	projDB, _ := projectSvc.GetProjectDB(p.ID)
	defer projDB.Close()

	boardSvc := &BoardService{}
	dueDate := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Due task",
		DueDate:  &dueDate,
	})

	from := time.Now().Format("2006-01-02")
	to := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	result, err := calSvc.GetCalendar(from, to)
	require.NoError(t, err)

	var found bool
	for _, e := range result.Events {
		if e.Type == "task_due" && e.Title == "Due task" {
			found = true
			assert.Equal(t, dueDate, e.Date)
			assert.Equal(t, p.Name, e.ProjectName)
		}
	}
	assert.True(t, found, "expected task_due event in calendar")
}

func TestCalendarService_TaskCompletedInRange(t *testing.T) {
	calSvc, projectSvc, cleanup := setupCalendarTest(t)
	defer cleanup()

	p, _ := projectSvc.Create(models.CreateProjectRequest{Name: "DoneTest"})
	projDB, _ := projectSvc.GetProjectDB(p.ID)
	defer projDB.Close()

	boardSvc := &BoardService{}
	boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 3,
		Title:    "Done task",
	})

	from := time.Now().Format("2006-01-02")
	to := time.Now().Format("2006-01-02")

	result, err := calSvc.GetCalendar(from, to)
	require.NoError(t, err)

	var found bool
	for _, e := range result.Events {
		if e.Type == "task_done" && e.Title == "Done task" {
			found = true
			assert.NotNil(t, e.TaskID)
			assert.Equal(t, p.Name, e.ProjectName)
		}
	}
	assert.True(t, found, "expected task_done event in calendar")
}

func TestCalendarService_BirthdayInRange(t *testing.T) {
	calSvc, _, cleanup := setupCalendarTest(t)
	defer cleanup()

	now := time.Now()
	dateStr := now.AddDate(0, 0, 2).Format("01-02")

	bsvc := NewBirthdayService(calSvc.CentralDB)
	bsvc.Create(models.CreateBirthdayRequest{
		Name: "Birthday Person",
		Date: dateStr,
	})

	from := now.Format("2006-01-02")
	to := now.AddDate(0, 0, 7).Format("2006-01-02")

	result, err := calSvc.GetCalendar(from, to)
	require.NoError(t, err)

	var found bool
	for _, e := range result.Events {
		if e.Type == "birthday" && e.Name == "Birthday Person" {
			found = true
		}
	}
	assert.True(t, found, "expected birthday event in calendar")
}

func TestCalendarService_RecurringEventInRange(t *testing.T) {
	calSvc, _, cleanup := setupCalendarTest(t)
	defer cleanup()

	now := time.Now()
	dateStr := now.AddDate(0, 0, 3).Format("01-02")

	rsvc := NewRecurringEventService(calSvc.CentralDB)
	rsvc.Create(models.CreateRecurringEventRequest{
		Title:     "Meeting",
		StartDate: "2020-01-01",
		Date:      dateStr,
		Category:  "work",
	})

	from := now.Format("2006-01-02")
	to := now.AddDate(0, 0, 7).Format("2006-01-02")

	result, err := calSvc.GetCalendar(from, to)
	require.NoError(t, err)

	var found bool
	for _, e := range result.Events {
		if e.Type == "recurring_event" && e.Title == "Meeting" {
			found = true
			assert.Equal(t, "yearly", e.Recurrence)
			assert.Equal(t, "work", e.Category)
		}
	}
	assert.True(t, found, "expected recurring_event in calendar")
}

func TestCalendarService_DefaultDateRange(t *testing.T) {
	calSvc, _, cleanup := setupCalendarTest(t)
	defer cleanup()

	result, err := calSvc.GetCalendar("", "")
	require.NoError(t, err)

	today := time.Now().Format("2006-01-02")
	nextWeek := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	assert.Equal(t, today, result.From)
	assert.Equal(t, nextWeek, result.To)
}

func TestCalendarService_Today(t *testing.T) {
	calSvc, _, cleanup := setupCalendarTest(t)
	defer cleanup()

	result, err := calSvc.GetToday()
	require.NoError(t, err)

	today := time.Now().Format("2006-01-02")
	assert.Equal(t, today, result.From)
	assert.Equal(t, today, result.To)
}

func TestCalendarService_EventsSortedByDate(t *testing.T) {
	calSvc, _, cleanup := setupCalendarTest(t)
	defer cleanup()

	now := time.Now()
	bsvc := NewBirthdayService(calSvc.CentralDB)
	bsvc.Create(models.CreateBirthdayRequest{Name: "B1", Date: now.AddDate(0, 0, 5).Format("01-02")})
	bsvc.Create(models.CreateBirthdayRequest{Name: "B2", Date: now.AddDate(0, 0, 2).Format("01-02")})

	from := now.Format("2006-01-02")
	to := now.AddDate(0, 0, 7).Format("2006-01-02")

	result, err := calSvc.GetCalendar(from, to)
	require.NoError(t, err)

	for i := 1; i < len(result.Events); i++ {
		assert.True(t, result.Events[i-1].Date <= result.Events[i].Date,
			"events should be sorted by date: %s <= %s", result.Events[i-1].Date, result.Events[i].Date)
	}
}
