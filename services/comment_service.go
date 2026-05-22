package services

import (
	"database/sql"
	"errors"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrCommentNotFound  = errors.New("comment not found")
	ErrCommentBodyEmpty = errors.New("comment body is required")
)

type CommentService struct {
	Activity *ActivityService
}

func (s *CommentService) ListComments(db *sql.DB, taskID int64) ([]models.Comment, error) {
	rows, err := db.Query(
		`SELECT id, task_id, author, body, created_at, updated_at
		 FROM comments WHERE task_id = $1 ORDER BY created_at`, taskID,
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

func (s *CommentService) AddComment(db *sql.DB, projectID, taskID int64, req models.CreateCommentRequest) (*models.Comment, error) {
	if req.Body == "" {
		return nil, ErrCommentBodyEmpty
	}

	var exists int
	if err := db.QueryRow("SELECT 1 FROM tasks WHERE id = $1", taskID).Scan(&exists); err != nil {
		return nil, ErrTaskNotFound
	}

	var id int64
	err := db.QueryRow(
		`INSERT INTO comments (project_id, task_id, author, body)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		projectID, taskID, req.Author, req.Body,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "comment_added",
			EntityType: "comment",
			EntityID:   id,
			Actor:      req.Author,
			Details:    map[string]any{"task_id": taskID},
		})
	}

	var c models.Comment
	err = db.QueryRow(
		`SELECT id, task_id, author, body, created_at, updated_at FROM comments WHERE id = $1`, id,
	).Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &c.CreatedAt, &c.UpdatedAt)
	return &c, err
}
