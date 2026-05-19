package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCentralDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")

	db, err := InitCentralDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	assert.FileExists(t, dbPath)

	tables := []string{"projects", "contacts", "birthdays", "recurring_events", "api_tokens"}
	for _, table := range tables {
		var count int
		err := db.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "table %s should exist", table)
	}
}

func TestInitCentralDB_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")

	db, err := InitCentralDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	require.NoError(t, err)
	assert.Equal(t, "wal", mode)
}

func TestInitCentralDB_ForeignKeysEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")

	db, err := InitCentralDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	var enabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&enabled)
	require.NoError(t, err)
	assert.Equal(t, 1, enabled)
}

func TestInitCentralDB_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")

	db1, err := InitCentralDB(dbPath)
	require.NoError(t, err)
	db1.Close()

	db2, err := InitCentralDB(dbPath)
	require.NoError(t, err)
	defer db2.Close()

	tables := []string{"projects", "contacts", "birthdays", "recurring_events", "api_tokens"}
	for _, table := range tables {
		var count int
		err := db2.QueryRow(
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "table %s should still exist after re-open", table)
	}
}

func TestInitCentralDB_InsertProject(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")

	db, err := InitCentralDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	result, err := db.Exec(
		`INSERT INTO projects (uuid, name, path) VALUES (?, ?, ?)`,
		"test-uuid-1", "Test Project", "/tmp/test",
	)
	require.NoError(t, err)

	id, _ := result.LastInsertId()
	assert.Equal(t, int64(1), id)

	var name string
	err = db.QueryRow("SELECT name FROM projects WHERE id = ?", id).Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "Test Project", name)
}
