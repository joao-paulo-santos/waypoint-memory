package services

import (
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTokenTest(t *testing.T) (*TokenService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	return NewTokenService(centralDB), func() { centralDB.Close() }
}

func TestTokenService_Create(t *testing.T) {
	svc, cleanup := setupTokenTest(t)
	defer cleanup()

	result, err := svc.Create("test-token", "")
	require.NoError(t, err)
	assert.Contains(t, result.Token, "wpt_")
	assert.Equal(t, "test-token", result.TokenInfo.Name)
	assert.Equal(t, 8, len(result.TokenInfo.Prefix))
}

func TestTokenService_List(t *testing.T) {
	svc, cleanup := setupTokenTest(t)
	defer cleanup()

	svc.Create("token-1", "")
	svc.Create("token-2", "")

	tokens, err := svc.List()
	require.NoError(t, err)
	require.Len(t, tokens, 2)
	names := []string{tokens[0].Name, tokens[1].Name}
	assert.Contains(t, names, "token-1")
	assert.Contains(t, names, "token-2")
}

func TestTokenService_Delete(t *testing.T) {
	svc, cleanup := setupTokenTest(t)
	defer cleanup()

	result, _ := svc.Create("to-delete", "")
	err := svc.Delete(result.TokenInfo.ID)
	require.NoError(t, err)

	tokens, _ := svc.List()
	assert.Len(t, tokens, 0)
}

func TestTokenService_Delete_NotFound(t *testing.T) {
	svc, cleanup := setupTokenTest(t)
	defer cleanup()

	err := svc.Delete(999)
	assert.ErrorIs(t, err, ErrTokenNotFound)
}

func TestTokenService_Create_NoName(t *testing.T) {
	svc, cleanup := setupTokenTest(t)
	defer cleanup()

	_, err := svc.Create("", "")
	assert.EqualError(t, err, "name is required")
}
