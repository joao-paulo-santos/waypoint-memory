package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Version int `yaml:"version"`
	Project struct {
		UUID         string `yaml:"uuid"`
		Name         string `yaml:"name"`
		Description  string `yaml:"description"`
		CreatedAt    string `yaml:"created_at"`
		RegisteredAt string `yaml:"registered_at"`
	} `yaml:"project"`
}

func InitProjectDir(projectPath, name, description string) (uuid string, err error) {
	waypointDir := filepath.Join(projectPath, ".waypoint")

	if err := os.MkdirAll(waypointDir, 0755); err != nil {
		return "", fmt.Errorf("create .waypoint dir: %w", err)
	}

	wikiDir := filepath.Join(waypointDir, "wiki")
	if err := os.MkdirAll(wikiDir, 0755); err != nil {
		return "", fmt.Errorf("create wiki dir: %w", err)
	}

	indexPath := filepath.Join(wikiDir, "index.md")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		if err := os.WriteFile(indexPath, []byte(""), 0644); err != nil {
			return "", fmt.Errorf("create index.md: %w", err)
		}
	}

	uuid = generateUUID()

	now := time.Now().UTC().Format(time.RFC3339)
	cfg := ProjectConfig{}
	cfg.Version = 1
	cfg.Project.UUID = uuid
	cfg.Project.Name = name
	cfg.Project.Description = description
	cfg.Project.CreatedAt = now
	cfg.Project.RegisteredAt = now

	cfgData, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	cfgPath := filepath.Join(waypointDir, "config.yaml")
	if err := os.WriteFile(cfgPath, cfgData, 0644); err != nil {
		return "", fmt.Errorf("write config.yaml: %w", err)
	}

	dbPath := filepath.Join(waypointDir, "project.db")
	db, err := InitProjectDB(dbPath)
	if err != nil {
		return "", fmt.Errorf("init project db: %w", err)
	}
	defer db.Close()

	return uuid, nil
}

func InitProjectDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open project db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := createProjectTables(db); err != nil {
		return nil, fmt.Errorf("create project tables: %w", err)
	}

	if err := ensureDefaultBuckets(db); err != nil {
		return nil, fmt.Errorf("ensure default buckets: %w", err)
	}

	return db, nil
}

func createProjectTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS labels (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		title       TEXT    NOT NULL,
		description TEXT    NOT NULL DEFAULT '',
		hex_color   TEXT    NOT NULL DEFAULT '',
		created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_labels_title ON labels(title);

	CREATE TABLE IF NOT EXISTS buckets (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		title          TEXT    NOT NULL,
		position       REAL    NOT NULL DEFAULT 0,
		is_done_bucket INTEGER NOT NULL DEFAULT 0,
		wip_limit      INTEGER NOT NULL DEFAULT 0,
		created_at     TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at     TEXT    NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		bucket_id   INTEGER NOT NULL REFERENCES buckets(id),
		title       TEXT    NOT NULL,
		description TEXT    NOT NULL DEFAULT '',
		position    REAL    NOT NULL DEFAULT 0,
		priority    INTEGER NOT NULL DEFAULT 0,
		due_date    TEXT,
		done        INTEGER NOT NULL DEFAULT 0,
		done_at     TEXT,
		created_by  TEXT    NOT NULL DEFAULT '',
		created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS task_labels (
		task_id  INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		label_id INTEGER NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
		PRIMARY KEY (task_id, label_id)
	);

	CREATE TABLE IF NOT EXISTS comments (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		author     TEXT    NOT NULL DEFAULT '',
		body       TEXT    NOT NULL,
		created_at TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at TEXT    NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_comments_task ON comments(task_id);

	CREATE TABLE IF NOT EXISTS sprints (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT    NOT NULL,
		started_at TEXT,
		ended_at   TEXT    NOT NULL DEFAULT (datetime('now')),
		task_count INTEGER NOT NULL DEFAULT 0,
		summary    TEXT    NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS archived_tasks (
		id                INTEGER PRIMARY KEY AUTOINCREMENT,
		sprint_id         INTEGER NOT NULL REFERENCES sprints(id) ON DELETE CASCADE,
		original_task_id  INTEGER NOT NULL,
		bucket_title      TEXT    NOT NULL,
		title             TEXT    NOT NULL,
		description       TEXT    NOT NULL DEFAULT '',
		priority          INTEGER NOT NULL DEFAULT 0,
		created_by        TEXT    NOT NULL DEFAULT '',
		due_date          TEXT,
		labels            TEXT    NOT NULL DEFAULT '[]',
		comments          TEXT    NOT NULL DEFAULT '[]',
		done_at           TEXT,
		original_created  TEXT    NOT NULL,
		archived_at       TEXT    NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_archived_tasks_sprint ON archived_tasks(sprint_id);

	CREATE TABLE IF NOT EXISTS activity_log (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		action      TEXT    NOT NULL,
		entity_type TEXT    NOT NULL,
		entity_id   INTEGER,
		actor       TEXT    NOT NULL DEFAULT 'user',
		details     TEXT    NOT NULL DEFAULT '{}',
		created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_activity_log_created ON activity_log(created_at DESC);
	`

	_, err := db.Exec(schema)
	return err
}

func ensureDefaultBuckets(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT count(*) FROM buckets").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaults := []struct {
		title        string
		position     float64
		isDoneBucket bool
	}{
		{"To-Do", 100, false},
		{"In Progress", 200, false},
		{"Done", 300, true},
	}

	for _, b := range defaults {
		_, err := db.Exec(
			"INSERT INTO buckets (title, position, is_done_bucket) VALUES (?, ?, ?)",
			b.title, b.position, b.isDoneBucket,
		)
		if err != nil {
			return fmt.Errorf("insert default bucket %q: %w", b.title, err)
		}
	}
	return nil
}

func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
