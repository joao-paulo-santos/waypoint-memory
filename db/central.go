package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func InitCentralDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open central db: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := createCentralTables(db); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return db, nil
}

func createCentralTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid          TEXT    NOT NULL UNIQUE,
		name          TEXT    NOT NULL,
		description   TEXT    NOT NULL DEFAULT '',
		path          TEXT    NOT NULL UNIQUE,
		docs_path     TEXT    NOT NULL DEFAULT '',
		color         TEXT    NOT NULL DEFAULT '',
		icon          TEXT    NOT NULL DEFAULT '',
		is_archived   INTEGER NOT NULL DEFAULT 0,
		last_opened   TEXT    NOT NULL DEFAULT '',
		created_at    TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at    TEXT    NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS contacts (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name  TEXT    NOT NULL DEFAULT '',
		last_name   TEXT    NOT NULL DEFAULT '',
		email       TEXT    NOT NULL DEFAULT '',
		phone       TEXT    NOT NULL DEFAULT '',
		company     TEXT    NOT NULL DEFAULT '',
		role        TEXT    NOT NULL DEFAULT '',
		notes       TEXT    NOT NULL DEFAULT '',
		tags        TEXT    NOT NULL DEFAULT '[]',
		created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
		updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS birthdays (
		id                 INTEGER PRIMARY KEY AUTOINCREMENT,
		contact_id         INTEGER REFERENCES contacts(id) ON DELETE SET NULL,
		name               TEXT    NOT NULL,
		date               TEXT    NOT NULL,
		year               INTEGER,
		remind_days_before INTEGER NOT NULL DEFAULT 3,
		notes              TEXT    NOT NULL DEFAULT '',
		created_at         TEXT    NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_birthdays_date ON birthdays(date);

	CREATE TABLE IF NOT EXISTS recurring_events (
		id                 INTEGER PRIMARY KEY AUTOINCREMENT,
		title              TEXT    NOT NULL,
		description        TEXT    NOT NULL DEFAULT '',
		start_date         TEXT    NOT NULL,
		end_date           TEXT,
		date               TEXT    NOT NULL,
		year               INTEGER,
		recurrence         TEXT    NOT NULL DEFAULT 'yearly',
		category           TEXT    NOT NULL DEFAULT '',
		remind_days_before INTEGER NOT NULL DEFAULT 3,
		created_at         TEXT    NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_recurring_events_date ON recurring_events(date);

	CREATE TABLE IF NOT EXISTS app_config (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS api_tokens (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT    NOT NULL,
		token_hash  TEXT    NOT NULL UNIQUE,
		prefix      TEXT    NOT NULL,
		permissions TEXT    NOT NULL DEFAULT '["read","write"]',
		last_used   TEXT,
		expires_at  TEXT,
		created_at  TEXT    NOT NULL DEFAULT (datetime('now'))
	);
	CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash);
	`

	_, err := db.Exec(schema)
	return err
}
