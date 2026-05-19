package services

import (
	"database/sql"
	"errors"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrCommentNotFound = errors.New("comment not found")
	ErrCommentBodyEmpty = errors.New("comment body is required")
)

type CommentService struct{}

func (s *CommentService) ListComments(db *sql.DB, taskID int64) ([]models.Comment, error) {
	rows, err := db.Query(
		`SELECT id, task_id, author, body, created_at, updated_at
		 FROM comments WHERE task_id = ? ORDER BY created_at`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var c models.Comment
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, nil
}

func (s *CommentService) AddComment(db *sql.DB, taskID int64, req models.CreateCommentRequest) (*models.Comment, error) {
	if req.Body == "" {
		return nil, ErrCommentBodyEmpty
	}

	var exists int
	if err := db.QueryRow("SELECT 1 FROM tasks WHERE id = ?", taskID).Scan(&exists); err != nil {
		return nil, ErrTaskNotFound
	}

	result, err := db.Exec(
		`INSERT INTO comments (task_id, author, body) VALUES (?, ?, ?)`,
		taskID, req.Author, req.Body,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	var c models.Comment
	err = db.QueryRow(
		`SELECT id, task_id, author, body, created_at, updated_at FROM comments WHERE id = ?`, id,
	).Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}
