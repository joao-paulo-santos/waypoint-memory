package services

import (
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthTest(t *testing.T) (*AuthService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	return NewAuthService(centralDB), func() { centralDB.Close() }
}

func TestAuthService_NoPassword(t *testing.T) {
	svc, cleanup := setupAuthTest(t)
	defer cleanup()

	has, err := svc.HasPassword()
	require.NoError(t, err)
	assert.False(t, has)

	err = svc.VerifyPassword("anything")
	assert.NoError(t, err)
}

func TestAuthService_SetPassword(t *testing.T) {
	svc, cleanup := setupAuthTest(t)
	defer cleanup()

	err := svc.SetPassword("test123")
	require.NoError(t, err)

	has, err := svc.HasPassword()
	require.NoError(t, err)
	assert.True(t, has)
}

func TestAuthService_VerifyPassword_Correct(t *testing.T) {
	svc, cleanup := setupAuthTest(t)
	defer cleanup()

	err := svc.SetPassword("test123")
	require.NoError(t, err)

	err = svc.VerifyPassword("test123")
	assert.NoError(t, err)
}

func TestAuthService_VerifyPassword_Wrong(t *testing.T) {
	svc, cleanup := setupAuthTest(t)
	defer cleanup()

	err := svc.SetPassword("test123")
	require.NoError(t, err)

	err = svc.VerifyPassword("wrong")
	assert.ErrorIs(t, err, ErrInvalidPassword)
}

func TestAuthService_SetPassword_Overwrite(t *testing.T) {
	svc, cleanup := setupAuthTest(t)
	defer cleanup()

	err := svc.SetPassword("first")
	require.NoError(t, err)

	err = svc.SetPassword("second")
	require.NoError(t, err)

	err = svc.VerifyPassword("first")
	assert.ErrorIs(t, err, ErrInvalidPassword)

	err = svc.VerifyPassword("second")
	assert.NoError(t, err)
}
