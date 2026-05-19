package services

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEventTest(t *testing.T) (*RecurringEventService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	return NewRecurringEventService(centralDB), func() { centralDB.Close() }
}

func TestRecurringEventService_CreateYearly(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	e, err := svc.Create(models.CreateRecurringEventRequest{
		Title:     "Anniversary",
		StartDate: "2020-08-15",
		Date:      "08-15",
	})
	require.NoError(t, err)
	assert.Equal(t, "Anniversary", e.Title)
	assert.Equal(t, "yearly", e.Recurrence)
	assert.Equal(t, 3, e.RemindDaysBefore)
}

func TestRecurringEventService_CreateMonthly(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	e, err := svc.Create(models.CreateRecurringEventRequest{
		Title:      "Haircut",
		StartDate:  "2026-01-15",
		Date:       "15",
		Recurrence: "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, "monthly", e.Recurrence)
	assert.Equal(t, "15", e.Date)
}

func TestRecurringEventService_InvalidRecurrence(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	_, err := svc.Create(models.CreateRecurringEventRequest{
		Title:      "Bad",
		StartDate:  "2020-01-01",
		Date:       "01-01",
		Recurrence: "weekly",
	})
	assert.ErrorIs(t, err, ErrInvalidRecurrence)
}

func TestRecurringEventService_InvalidDate(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	_, err := svc.Create(models.CreateRecurringEventRequest{
		Title:     "Bad",
		StartDate: "2020-01-01",
		Date:      "15",
	})
	assert.ErrorIs(t, err, ErrInvalidDate)

	_, err = svc.Create(models.CreateRecurringEventRequest{
		Title:      "Bad",
		StartDate:  "2020-01-01",
		Date:       "08-15",
		Recurrence: "monthly",
	})
	assert.ErrorIs(t, err, ErrInvalidDate)
}

func TestRecurringEventService_StartDateBlocksBackwards(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	futureYear := time.Now().Year() + 2
	svc.Create(models.CreateRecurringEventRequest{
		Title:     "Future",
		StartDate: time.Date(futureYear, 6, 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
		Date:      "08-15",
	})

	results, err := svc.GetUpcoming(400)
	require.NoError(t, err)

	for _, r := range results {
		assert.True(t, r.DaysUntil > 0)
	}
}

func TestRecurringEventService_EndDateStopsRecurrence(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	now := time.Now()
	endDate := now.AddDate(0, 0, 20).Format("2006-01-02")

	svc.Create(models.CreateRecurringEventRequest{
		Title:     "Limited",
		StartDate: "2020-01-01",
		EndDate:   &endDate,
		Date:      "08-15",
	})

	results, err := svc.GetUpcoming(400)
	require.NoError(t, err)

	for _, r := range results {
		d, _ := time.Parse("2006-01-02", r.Date)
		ed, _ := time.Parse("2006-01-02", endDate)
		assert.True(t, !d.After(ed), "event %s should not be after end_date %s", r.Date, endDate)
	}
}

func TestRecurringEventService_NoEndDate(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	svc.Create(models.CreateRecurringEventRequest{
		Title:     "Forever",
		StartDate: "2020-01-01",
		Date:      "08-15",
	})

	results, err := svc.GetUpcoming(400)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestRecurringEventService_UpcomingMonthly(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	now := time.Now()
	dayStr := now.Format("02")

	svc.Create(models.CreateRecurringEventRequest{
		Title:      "Monthly",
		StartDate:  "2020-01-01",
		Date:       dayStr,
		Recurrence: "monthly",
	})

	results, err := svc.GetUpcoming(60)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(results), 1)
}

func TestRecurringEventService_CRUD(t *testing.T) {
	svc, cleanup := setupEventTest(t)
	defer cleanup()

	e, err := svc.Create(models.CreateRecurringEventRequest{
		Title:     "Test",
		StartDate: "2020-01-01",
		Date:      "03-15",
	})
	require.NoError(t, err)
	assert.Equal(t, "Test", e.Title)

	got, err := svc.GetByID(e.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test", got.Title)

	newTitle := "Updated"
	updated, err := svc.Update(e.ID, models.UpdateRecurringEventRequest{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Title)

	list, err := svc.List()
	require.NoError(t, err)
	require.Len(t, list, 1)

	err = svc.Delete(e.ID)
	require.NoError(t, err)
	_, err = svc.GetByID(e.ID)
	assert.ErrorIs(t, err, ErrRecurringEventNotFound)
}
