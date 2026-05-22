package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrEmptyDoneBucket = errors.New("done bucket is empty, nothing to archive")
)

type SprintService struct{}

func (s *SprintService) EndSprint(db *sql.DB, projectID int64, req models.EndSprintRequest) (*models.EndSprintResponse, error) {
	var doneBucketID int64
	err := db.QueryRow(
		"SELECT id FROM buckets WHERE project_id = $1 AND is_done_bucket = TRUE LIMIT 1", projectID,
	).Scan(&doneBucketID)
	if err == sql.ErrNoRows {
		return nil, ErrNoDoneBucket
	}
	if err != nil {
		return nil, err
	}

	var taskCount int
	if err := db.QueryRow(
		"SELECT count(*) FROM tasks WHERE bucket_id = $1 AND project_id = $2 AND sprint_id IS NULL", doneBucketID, projectID,
	).Scan(&taskCount); err != nil {
		return nil, err
	}
	if taskCount == 0 {
		return nil, ErrEmptyDoneBucket
	}

	sprintName := req.SprintName
	if sprintName == "" {
		sprintName = generateSprintName()
	}

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	sprintName, err = s.resolveSprintName(tx, projectID, sprintName)
	if err != nil {
		return nil, err
	}

	var sprintID int64
	err = tx.QueryRow(
		`INSERT INTO sprints (project_id, name, ended_at, task_count, summary)
		 VALUES ($1, $2, NOW(), 0, $3)
		 RETURNING id`,
		projectID, sprintName, req.Summary,
	).Scan(&sprintID)
	if err != nil {
		return nil, fmt.Errorf("create sprint: %w", err)
	}

	_, err = tx.Exec(
		"UPDATE tasks SET sprint_id = $1 WHERE bucket_id = $2 AND project_id = $3 AND sprint_id IS NULL",
		sprintID, doneBucketID, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("assign sprint to tasks: %w", err)
	}

	_, err = tx.Exec("UPDATE sprints SET task_count = $1 WHERE id = $2", taskCount, sprintID)
	if err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]int{"task_count": taskCount})
	_, err = tx.Exec(
		`INSERT INTO activity_log (project_id, action, entity_type, entity_id, actor, details)
		 VALUES ($1, 'sprint_archived', 'sprint', $2, 'user', $3)`,
		projectID, sprintID, string(details),
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	sprint := models.Sprint{
		ID:        sprintID,
		Name:      sprintName,
		EndedAt:   time.Now().UTC().Format(time.RFC3339),
		TaskCount: taskCount,
		Summary:   req.Summary,
	}

	return &models.EndSprintResponse{
		Sprint:        sprint,
		TasksArchived: taskCount,
	}, nil
}

func (s *SprintService) resolveSprintName(tx *sql.Tx, projectID int64, name string) (string, error) {
	var count int
	err := tx.QueryRow("SELECT count(*) FROM sprints WHERE project_id = $1 AND name = $2", projectID, name).Scan(&count)
	if err != nil {
		return name, err
	}
	if count == 0 {
		return name, nil
	}

	suffix := 2
	for {
		candidate := fmt.Sprintf("%s-%d", name, suffix)
		err := tx.QueryRow("SELECT count(*) FROM sprints WHERE project_id = $1 AND name = $2", projectID, candidate).Scan(&count)
		if err != nil {
			return name, err
		}
		if count == 0 {
			return candidate, nil
		}
		suffix++
	}
}

func generateSprintName() string {
	now := time.Now()
	_, week := now.ISOWeek()
	return fmt.Sprintf("Sprint %d-W%02d", now.Year(), week)
}

func (s *SprintService) ListSprints(db *sql.DB, projectID int64) ([]models.Sprint, error) {
	rows, err := db.Query(
		`SELECT id, name, started_at, ended_at, task_count, summary
		 FROM sprints WHERE project_id = $1 ORDER BY ended_at DESC`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sprints []models.Sprint
	for rows.Next() {
		var sp models.Sprint
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.StartedAt, &sp.EndedAt, &sp.TaskCount, &sp.Summary); err != nil {
			return nil, err
		}
		sprints = append(sprints, sp)
	}
	return sprints, nil
}

func (s *SprintService) GetSprintDetail(db *sql.DB, projectID, sprintID int64) (*models.SprintDetail, error) {
	var sp models.Sprint
	err := db.QueryRow(
		`SELECT id, name, started_at, ended_at, task_count, summary
		 FROM sprints WHERE id = $1 AND project_id = $2`, sprintID, projectID,
	).Scan(&sp.ID, &sp.Name, &sp.StartedAt, &sp.EndedAt, &sp.TaskCount, &sp.Summary)
	if err == sql.ErrNoRows {
		return nil, errors.New("sprint not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT id, project_id, bucket_id, sprint_id, title, description,
		        priority, due_date, done, done_at, created_by, created_at, updated_at
		 FROM tasks WHERE sprint_id = $1 AND project_id = $2 ORDER BY done_at`, sprintID, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.BucketID, &t.SprintID, &t.Title, &t.Description,
			&t.Priority, &t.DueDate, &t.Done, &t.DoneAt, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}

		labelRows, err := db.Query(
			`SELECT l.id, l.title, l.description, l.hex_color, l.created_at
			 FROM labels l
			 JOIN task_labels tl ON tl.label_id = l.id
			 WHERE tl.task_id = $1
			 ORDER BY l.title`, t.ID,
		)
		if err == nil {
			for labelRows.Next() {
				var l models.Label
				if labelRows.Scan(&l.ID, &l.Title, &l.Description, &l.HexColor, &l.CreatedAt) == nil {
					t.Labels = append(t.Labels, l)
				}
			}
			labelRows.Close()
		}

		tasks = append(tasks, t)
	}

	return &models.SprintDetail{Sprint: sp, Tasks: tasks}, nil
}
