package services

import (
	"database/sql"
	"errors"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrLabelNotFound  = errors.New("label not found")
	ErrLabelDuplicate = errors.New("label title already exists")
)

type LabelService struct {
	Activity *ActivityService
}

func (s *LabelService) ListLabels(db *sql.DB, projectID int64) ([]models.Label, error) {
	rows, err := db.Query(
		`SELECT id, title, description, hex_color, created_at FROM labels WHERE project_id = $1 ORDER BY title`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []models.Label
	for rows.Next() {
		var l models.Label
		if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.HexColor, &l.CreatedAt); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, nil
}

func (s *LabelService) CreateLabel(db *sql.DB, projectID int64, req models.CreateLabelRequest) (*models.Label, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	var id int64
	err := db.QueryRow(
		`INSERT INTO labels (project_id, title, description, hex_color)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id`,
		projectID, req.Title, req.Description, req.HexColor,
	).Scan(&id)
	if err != nil {
		return nil, ErrLabelDuplicate
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "label_created",
			EntityType: "label",
			EntityID:   id,
			Details:    map[string]any{"title": req.Title},
		})
	}

	return s.GetLabel(db, id)
}

func (s *LabelService) GetLabel(db *sql.DB, id int64) (*models.Label, error) {
	l := &models.Label{}
	err := db.QueryRow(
		`SELECT id, title, description, hex_color, created_at FROM labels WHERE id = $1`, id,
	).Scan(&l.ID, &l.Title, &l.Description, &l.HexColor, &l.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrLabelNotFound
	}
	return l, err
}

func (s *LabelService) DeleteLabel(db *sql.DB, projectID, id int64) error {
	result, err := db.Exec("DELETE FROM labels WHERE id = $1", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrLabelNotFound
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "label_deleted",
			EntityType: "label",
			EntityID:   id,
		})
	}

	return nil
}

func (s *LabelService) GetTaskLabels(db *sql.DB, taskID int64) ([]models.Label, error) {
	rows, err := db.Query(
		`SELECT l.id, l.title, l.description, l.hex_color, l.created_at
		 FROM labels l
		 JOIN task_labels tl ON tl.label_id = l.id
		 WHERE tl.task_id = $1
		 ORDER BY l.title`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labels []models.Label
	for rows.Next() {
		var l models.Label
		if err := rows.Scan(&l.ID, &l.Title, &l.Description, &l.HexColor, &l.CreatedAt); err != nil {
			return nil, err
		}
		labels = append(labels, l)
	}
	return labels, nil
}
