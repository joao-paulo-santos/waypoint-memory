package db

import (
	"crypto/rand"
	"database/sql"
	"fmt"
)

func EnsureDefaultBuckets(db *sql.DB, projectID int64) error {
	var count int
	err := db.QueryRow("SELECT count(*) FROM buckets WHERE project_id = $1", projectID).Scan(&count)
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
			"INSERT INTO buckets (project_id, title, position, is_done_bucket) VALUES ($1, $2, $3, $4)",
			projectID, b.title, b.position, b.isDoneBucket,
		)
		if err != nil {
			return fmt.Errorf("insert default bucket %q: %w", b.title, err)
		}
	}
	return nil
}

func EnsureWikiIndex(db *sql.DB, projectID int64) error {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM wiki_pages WHERE project_id = $1 AND path = 'index')", projectID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		_, err = db.Exec("INSERT INTO wiki_pages (project_id, path, content) VALUES ($1, 'index', '')", projectID)
		return err
	}
	return nil
}

func GenerateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func GenerateToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
