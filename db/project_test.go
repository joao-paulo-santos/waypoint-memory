package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestInitProjectDir(t *testing.T) {
	projectDir := t.TempDir()

	uuid, err := InitProjectDir(projectDir, "Test Project", "A test")
	require.NoError(t, err)
	assert.NotEmpty(t, uuid)

	waypointDir := filepath.Join(projectDir, ".waypoint")
	assert.DirExists(t, waypointDir)
	assert.FileExists(t, filepath.Join(waypointDir, "config.yaml"))
	assert.FileExists(t, filepath.Join(waypointDir, "project.db"))
	assert.DirExists(t, filepath.Join(waypointDir, "wiki"))
	assert.FileExists(t, filepath.Join(waypointDir, "wiki", "index.md"))
}

func TestInitProjectDir_ConfigYAML(t *testing.T) {
	projectDir := t.TempDir()

	uuid, err := InitProjectDir(projectDir, "My Project", "Description here")
	require.NoError(t, err)

	cfgData, err := os.ReadFile(filepath.Join(projectDir, ".waypoint", "config.yaml"))
	require.NoError(t, err)

	var cfg ProjectConfig
	require.NoError(t, yaml.Unmarshal(cfgData, &cfg))

	assert.Equal(t, 1, cfg.Version)
	assert.Equal(t, uuid, cfg.Project.UUID)
	assert.Equal(t, "My Project", cfg.Project.Name)
	assert.Equal(t, "Description here", cfg.Project.Description)
	assert.NotEmpty(t, cfg.Project.CreatedAt)
	assert.NotEmpty(t, cfg.Project.RegisteredAt)
}

func TestInitProjectDir_DefaultBuckets(t *testing.T) {
	projectDir := t.TempDir()

	_, err := InitProjectDir(projectDir, "Test", "")
	require.NoError(t, err)

	dbPath := filepath.Join(projectDir, ".waypoint", "project.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	rows, err := db.Query("SELECT title, position, is_done_bucket FROM buckets ORDER BY position")
	require.NoError(t, err)
	defer rows.Close()

	var buckets []struct {
		title        string
		position     float64
		isDoneBucket int
	}
	for rows.Next() {
		var b struct {
			title        string
			position     float64
			isDoneBucket int
		}
		require.NoError(t, rows.Scan(&b.title, &b.position, &b.isDoneBucket))
		buckets = append(buckets, b)
	}

	require.Len(t, buckets, 3)
	assert.Equal(t, "To-Do", buckets[0].title)
	assert.Equal(t, float64(100), buckets[0].position)
	assert.Equal(t, 0, buckets[0].isDoneBucket)

	assert.Equal(t, "In Progress", buckets[1].title)
	assert.Equal(t, float64(200), buckets[1].position)

	assert.Equal(t, "Done", buckets[2].title)
	assert.Equal(t, float64(300), buckets[2].position)
	assert.Equal(t, 1, buckets[2].isDoneBucket)
}

func TestInitProjectDir_Idempotent(t *testing.T) {
	projectDir := t.TempDir()

	uuid1, err := InitProjectDir(projectDir, "Test", "")
	require.NoError(t, err)

	_ = uuid1
}

func TestInitProjectDB_AllTables(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")

	db, err := InitProjectDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	tables := []string{
		"labels", "buckets", "tasks", "task_labels",
		"comments", "sprints", "archived_tasks", "activity_log",
	}
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

func TestInitProjectDB_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")

	db, err := InitProjectDB(dbPath)
	require.NoError(t, err)
	defer db.Close()

	var mode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&mode)
	require.NoError(t, err)
	assert.Equal(t, "wal", mode)
}

func TestGenerateUUID(t *testing.T) {
	u1 := generateUUID()
	u2 := generateUUID()
	assert.NotEmpty(t, u1)
	assert.NotEqual(t, u1, u2)
	assert.Len(t, u1, 36)
}
