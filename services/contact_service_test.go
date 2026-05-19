package services

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupContactTest(t *testing.T) (*ContactService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	return NewContactService(centralDB), func() { centralDB.Close() }
}

func TestContactService_CRUD(t *testing.T) {
	svc, cleanup := setupContactTest(t)
	defer cleanup()

	c, err := svc.Create(models.CreateContactRequest{
		FirstName: "Jane",
		LastName:  "Doe",
		Email:     "jane@example.com",
		Company:   "Acme",
	})
	require.NoError(t, err)
	assert.Equal(t, "Jane", c.FirstName)
	assert.Equal(t, "jane@example.com", c.Email)

	got, err := svc.GetByID(c.ID)
	require.NoError(t, err)
	assert.Equal(t, "Jane", got.FirstName)

	contacts, err := svc.List()
	require.NoError(t, err)
	require.Len(t, contacts, 1)

	newCompany := "New Corp"
	updated, err := svc.Update(c.ID, models.UpdateContactRequest{Company: &newCompany})
	require.NoError(t, err)
	assert.Equal(t, "New Corp", updated.Company)
	assert.Equal(t, "Jane", updated.FirstName)

	err = svc.Delete(c.ID)
	require.NoError(t, err)
	_, err = svc.GetByID(c.ID)
	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestContactService_Tags(t *testing.T) {
	svc, cleanup := setupContactTest(t)
	defer cleanup()

	c, err := svc.Create(models.CreateContactRequest{
		FirstName: "Tagged",
		Tags:      "recruiter,flutter",
	})
	require.NoError(t, err)

	var tags []string
	require.NoError(t, json.Unmarshal([]byte(c.Tags), &tags))
	assert.Equal(t, []string{"recruiter", "flutter"}, tags)
}

func TestContactService_UpdateTags(t *testing.T) {
	svc, cleanup := setupContactTest(t)
	defer cleanup()

	c, _ := svc.Create(models.CreateContactRequest{FirstName: "A"})
	newTags := "x,y,z"
	updated, err := svc.Update(c.ID, models.UpdateContactRequest{Tags: &newTags})
	require.NoError(t, err)

	var tags []string
	require.NoError(t, json.Unmarshal([]byte(updated.Tags), &tags))
	assert.Equal(t, []string{"x", "y", "z"}, tags)
}

func TestContactService_Delete_NotFound(t *testing.T) {
	svc, cleanup := setupContactTest(t)
	defer cleanup()

	err := svc.Delete(999)
	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestContactService_Create_NoFirstName(t *testing.T) {
	svc, cleanup := setupContactTest(t)
	defer cleanup()

	_, err := svc.Create(models.CreateContactRequest{})
	assert.EqualError(t, err, "first_name is required")
}
