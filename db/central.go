package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDB(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	db.SetMaxOpenConns(25)

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            SERIAL PRIMARY KEY,
		username      TEXT    NOT NULL UNIQUE,
		password_hash TEXT    NOT NULL,
		created_at    TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id         SERIAL PRIMARY KEY,
		user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token      TEXT    NOT NULL UNIQUE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		expires_at TIMESTAMP NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);

	CREATE TABLE IF NOT EXISTS projects (
		id          SERIAL PRIMARY KEY,
		slug        TEXT    NOT NULL UNIQUE,
		name        TEXT    NOT NULL,
		description TEXT    NOT NULL DEFAULT '',
		color       TEXT    NOT NULL DEFAULT '',
		icon        TEXT    NOT NULL DEFAULT '',
		owner_id    INTEGER NOT NULL REFERENCES users(id),
		is_archived BOOLEAN NOT NULL DEFAULT FALSE,
		last_opened TIMESTAMP,
		created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_projects_owner ON projects(owner_id);

	CREATE TABLE IF NOT EXISTS contacts (
		id                   SERIAL PRIMARY KEY,
		owner_id             INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		first_name           TEXT    NOT NULL DEFAULT '',
		last_name            TEXT    NOT NULL DEFAULT '',
		email                TEXT    NOT NULL DEFAULT '',
		phone                TEXT    NOT NULL DEFAULT '',
		company              TEXT    NOT NULL DEFAULT '',
		role                 TEXT    NOT NULL DEFAULT '',
		notes                TEXT    NOT NULL DEFAULT '',
		tags                 JSONB   NOT NULL DEFAULT '[]',
		birthday_date        TEXT    NOT NULL DEFAULT '',
		birthday_year        INTEGER,
		birthday_remind_days INTEGER NOT NULL DEFAULT 3,
		birthday_notes       TEXT    NOT NULL DEFAULT '',
		created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at           TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_contacts_owner ON contacts(owner_id);

	CREATE TABLE IF NOT EXISTS events (
		id                 SERIAL PRIMARY KEY,
		owner_id           INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title              TEXT    NOT NULL,
		description        TEXT    NOT NULL DEFAULT '',
		date               TEXT    NOT NULL,
		time               TEXT,
		recurrence         TEXT,
		category           TEXT    NOT NULL DEFAULT '',
		remind_days_before INTEGER NOT NULL DEFAULT 3,
		created_at         TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_events_date ON events(date);
	CREATE INDEX IF NOT EXISTS idx_events_owner ON events(owner_id);

	CREATE TABLE IF NOT EXISTS app_config (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS api_tokens (
		id          SERIAL PRIMARY KEY,
		user_id     INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name        TEXT    NOT NULL,
		token_hash  TEXT    NOT NULL UNIQUE,
		prefix      TEXT    NOT NULL,
		permissions JSONB   NOT NULL DEFAULT '["read","write"]',
		last_used   TIMESTAMP,
		expires_at  TIMESTAMP,
		created_at  TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_api_tokens_hash ON api_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_api_tokens_user ON api_tokens(user_id);

	-- Project-scoped tables

	CREATE TABLE IF NOT EXISTS labels (
		id          SERIAL PRIMARY KEY,
		project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		title       TEXT    NOT NULL,
		description TEXT    NOT NULL DEFAULT '',
		hex_color   TEXT    NOT NULL DEFAULT '',
		created_at  TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_labels_project_title ON labels(project_id, title);

	CREATE TABLE IF NOT EXISTS buckets (
		id             SERIAL PRIMARY KEY,
		project_id     INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		title          TEXT    NOT NULL,
		position       DOUBLE PRECISION NOT NULL DEFAULT 0,
		is_done_bucket BOOLEAN NOT NULL DEFAULT FALSE,
		wip_limit      INTEGER NOT NULL DEFAULT 0,
		created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at     TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS sprints (
		id         SERIAL PRIMARY KEY,
		project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		name       TEXT    NOT NULL,
		started_at TIMESTAMP,
		ended_at   TIMESTAMP NOT NULL DEFAULT NOW(),
		task_count INTEGER NOT NULL DEFAULT 0,
		summary    TEXT    NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id          SERIAL PRIMARY KEY,
		project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		bucket_id   INTEGER NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
		sprint_id   INTEGER REFERENCES sprints(id) ON DELETE SET NULL,
		title       TEXT    NOT NULL,
		description TEXT    NOT NULL DEFAULT '',
		position    DOUBLE PRECISION NOT NULL DEFAULT 0,
		priority    INTEGER NOT NULL DEFAULT 0,
		due_date    TEXT,
		done        BOOLEAN NOT NULL DEFAULT FALSE,
		done_at     TIMESTAMP,
		created_by  TEXT    NOT NULL DEFAULT '',
		created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS task_labels (
		task_id  INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		label_id INTEGER NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
		PRIMARY KEY (task_id, label_id)
	);

	CREATE TABLE IF NOT EXISTS comments (
		id         SERIAL PRIMARY KEY,
		project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		author     TEXT    NOT NULL DEFAULT '',
		body       TEXT    NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_comments_task ON comments(task_id);

	CREATE TABLE IF NOT EXISTS activity_log (
		id          SERIAL PRIMARY KEY,
		project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		action      TEXT    NOT NULL,
		entity_type TEXT    NOT NULL,
		entity_id   INTEGER,
		actor       TEXT    NOT NULL DEFAULT 'user',
		details     JSONB   NOT NULL DEFAULT '{}',
		created_at  TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE INDEX IF NOT EXISTS idx_activity_log_created ON activity_log(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_activity_log_project ON activity_log(project_id);

	CREATE TABLE IF NOT EXISTS wiki_pages (
		id          SERIAL PRIMARY KEY,
		project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
		path        TEXT    NOT NULL,
		content     TEXT    NOT NULL DEFAULT '',
		created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMP NOT NULL DEFAULT NOW()
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_wiki_pages_project_path ON wiki_pages(project_id, path);
	`
	_, err := db.Exec(schema)
	return err
}
