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

func setupBirthdayTest(t *testing.T) (*BirthdayService, *ContactService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	return NewBirthdayService(centralDB), NewContactService(centralDB), func() { centralDB.Close() }
}

func TestBirthdayService_Create(t *testing.T) {
	bsvc, _, cleanup := setupBirthdayTest(t)
	defer cleanup()

	b, err := bsvc.Create(models.CreateBirthdayRequest{
		Name: "Jane Doe",
		Date: "05-22",
		Year: func() *int { y := 1990; return &y }(),
	})
	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", b.Name)
	assert.Equal(t, "05-22", b.Date)
	require.NotNil(t, b.Year)
	assert.Equal(t, 1990, *b.Year)
	assert.Equal(t, 3, b.RemindDaysBefore)
}

func TestBirthdayService_Create_InvalidDate(t *testing.T) {
	bsvc, _, cleanup := setupBirthdayTest(t)
	defer cleanup()

	_, err := bsvc.Create(models.CreateBirthdayRequest{
		Name: "Bad",
		Date: "2024-05-22",
	})
	assert.ErrorIs(t, err, ErrBirthdayBadDate)
}

func TestBirthdayService_LinkedToContact(t *testing.T) {
	bsvc, csvc, cleanup := setupBirthdayTest(t)
	defer cleanup()

	c, _ := csvc.Create(models.CreateContactRequest{FirstName: "Jane"})
	b, err := bsvc.Create(models.CreateBirthdayRequest{
		ContactID: &c.ID,
		Name:      "Jane",
		Date:      "12-25",
	})
	require.NoError(t, err)
	require.NotNil(t, b.ContactID)
	assert.Equal(t, c.ID, *b.ContactID)
}

func TestBirthdayService_DeleteContact_BirthdaySurvives(t *testing.T) {
	bsvc, csvc, cleanup := setupBirthdayTest(t)
	defer cleanup()

	c, _ := csvc.Create(models.CreateContactRequest{FirstName: "Jane"})
	b, _ := bsvc.Create(models.CreateBirthdayRequest{
		ContactID: &c.ID,
		Name:      "Jane",
		Date:      "06-15",
	})

	csvc.Delete(c.ID)

	survived, err := bsvc.GetByID(b.ID)
	require.NoError(t, err)
	assert.Equal(t, "Jane", survived.Name)
	assert.Nil(t, survived.ContactID)
}

func TestBirthdayService_StandaloneBirthday(t *testing.T) {
	bsvc, _, cleanup := setupBirthdayTest(t)
	defer cleanup()

	b, err := bsvc.Create(models.CreateBirthdayRequest{
		Name: "Standalone",
		Date: "01-01",
	})
	require.NoError(t, err)
	assert.Nil(t, b.ContactID)
}

func TestBirthdayService_Upcoming(t *testing.T) {
	bsvc, _, cleanup := setupBirthdayTest(t)
	defer cleanup()

	now := time.Now()
	nextMonth := now.AddDate(0, 0, 10)
	dateStr := nextMonth.Format("01-02")

	bsvc.Create(models.CreateBirthdayRequest{
		Name: "Soon",
		Date: dateStr,
	})

	results, err := bsvc.GetUpcoming(30)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "Soon", results[0].Name)
	assert.LessOrEqual(t, results[0].DaysUntil, 30)
}

func TestBirthdayService_Upcoming_AgeCalculation(t *testing.T) {
	bsvc, _, cleanup := setupBirthdayTest(t)
	defer cleanup()

	now := time.Now()
	nextMonth := now.AddDate(0, 0, 10)
	dateStr := nextMonth.Format("01-02")

	year := 1971
	bsvc.Create(models.CreateBirthdayRequest{
		Name: "Old",
		Date: dateStr,
		Year: &year,
	})

	results, err := bsvc.GetUpcoming(30)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotNil(t, results[0].AgeTurning)
}

func TestBirthdayService_Upcoming_NotInRange(t *testing.T) {
	bsvc, _, cleanup := setupBirthdayTest(t)
	defer cleanup()

	bsvc.Create(models.CreateBirthdayRequest{
		Name: "Far",
		Date: "01-01",
	})

	results, err := bsvc.GetUpcoming(7)
	require.NoError(t, err)

	now := time.Now()
	jan1 := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	if jan1.Before(now) {
		jan1 = time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	daysUntil := int(jan1.Sub(now).Hours() / 24)
	if daysUntil <= 7 {
		assert.Len(t, results, 1)
	} else {
		assert.Len(t, results, 0)
	}
}

func TestBirthdayService_CRUD(t *testing.T) {
	bsvc, _, cleanup := setupBirthdayTest(t)
	defer cleanup()

	b, err := bsvc.Create(models.CreateBirthdayRequest{
		Name: "Test",
		Date: "03-15",
	})
	require.NoError(t, err)
	assert.Equal(t, "Test", b.Name)

	got, err := bsvc.GetByID(b.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test", got.Name)

	newName := "Updated"
	updated, err := bsvc.Update(b.ID, models.UpdateBirthdayRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)

	list, err := bsvc.List()
	require.NoError(t, err)
	require.Len(t, list, 1)

	err = bsvc.Delete(b.ID)
	require.NoError(t, err)
	_, err = bsvc.GetByID(b.ID)
	assert.ErrorIs(t, err, ErrBirthdayNotFound)
}
