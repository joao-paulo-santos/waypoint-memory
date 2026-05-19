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

func (s *SprintService) EndSprint(db *sql.DB, req models.EndSprintRequest) (*models.EndSprintResponse, error) {
	var doneBucketID int64
	var doneBucketTitle string
	err := db.QueryRow(
		"SELECT id, title FROM buckets WHERE is_done_bucket = 1 LIMIT 1",
	).Scan(&doneBucketID, &doneBucketTitle)
	if err == sql.ErrNoRows {
		return nil, ErrNoDoneBucket
	}
	if err != nil {
		return nil, err
	}

	var taskCount int
	if err := db.QueryRow(
		"SELECT count(*) FROM tasks WHERE bucket_id = ?", doneBucketID,
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

	sprintName, err = s.resolveSprintName(tx, sprintName)
	if err != nil {
		return nil, err
	}

	var sprintID int64
	result, err := tx.Exec(
		`INSERT INTO sprints (name, ended_at, task_count, summary) VALUES (?, datetime('now'), 0, ?)`,
		sprintName, req.Summary,
	)
	if err != nil {
		return nil, fmt.Errorf("create sprint: %w", err)
	}
	sprintID, _ = result.LastInsertId()

	rows, err := tx.Query(
		`SELECT id, title, description, priority, created_by, due_date, done_at, created_at
		 FROM tasks WHERE bucket_id = ?`, doneBucketID,
	)
	if err != nil {
		return nil, err
	}

	type taskRow struct {
		ID          int64
		Title       string
		Description string
		Priority    int
		CreatedBy   string
		DueDate     sql.NullString
		DoneAt      sql.NullString
		CreatedAt   string
	}

	var tasks []taskRow
	for rows.Next() {
		var tr taskRow
		if err := rows.Scan(&tr.ID, &tr.Title, &tr.Description, &tr.Priority,
			&tr.CreatedBy, &tr.DueDate, &tr.DoneAt, &tr.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		tasks = append(tasks, tr)
	}
	rows.Close()

	for _, task := range tasks {
		labelsJSON, _ := s.snapshotLabels(tx, task.ID)
		commentsJSON, _ := s.snapshotComments(tx, task.ID)

		var dueDate, doneAt any
		if task.DueDate.Valid {
			dueDate = task.DueDate.String
		}
		if task.DoneAt.Valid {
			doneAt = task.DoneAt.String
		}

		_, err := tx.Exec(
			`INSERT INTO archived_tasks (
				sprint_id, original_task_id, bucket_title, title, description,
				priority, created_by, due_date, labels, comments, done_at, original_created
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			sprintID, task.ID, doneBucketTitle, task.Title, task.Description,
			task.Priority, task.CreatedBy, dueDate, labelsJSON, commentsJSON,
			doneAt, task.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("archive task %d: %w", task.ID, err)
		}
	}

	_, err = tx.Exec("UPDATE sprints SET task_count = ? WHERE id = ?", len(tasks), sprintID)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("DELETE FROM tasks WHERE bucket_id = ?", doneBucketID)
	if err != nil {
		return nil, err
	}

	details, _ := json.Marshal(map[string]int{"task_count": len(tasks)})
	_, err = tx.Exec(
		`INSERT INTO activity_log (action, entity_type, entity_id, actor, details)
		 VALUES ('sprint_archived', 'sprint', ?, 'user', ?)`,
		sprintID, string(details),
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
		EndedAt:   time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		TaskCount: len(tasks),
		Summary:   req.Summary,
	}

	return &models.EndSprintResponse{
		Sprint:        sprint,
		TasksArchived: len(tasks),
	}, nil
}

func (s *SprintService) resolveSprintName(tx *sql.Tx, name string) (string, error) {
	var count int
	err := tx.QueryRow("SELECT count(*) FROM sprints WHERE name = ?", name).Scan(&count)
	if err != nil {
		return name, err
	}
	if count == 0 {
		return name, nil
	}

	suffix := 2
	for {
		candidate := fmt.Sprintf("%s-%d", name, suffix)
		err := tx.QueryRow("SELECT count(*) FROM sprints WHERE name = ?", candidate).Scan(&count)
		if err != nil {
			return name, err
		}
		if count == 0 {
			return candidate, nil
		}
		suffix++
	}
}

func (s *SprintService) snapshotLabels(tx *sql.Tx, taskID int64) (string, error) {
	rows, err := tx.Query(
		`SELECT l.id, l.title, l.hex_color
		 FROM labels l
		 JOIN task_labels tl ON tl.label_id = l.id
		 WHERE tl.task_id = ?`, taskID,
	)
	if err != nil {
		return "[]", nil
	}
	defer rows.Close()

	type labelSnapshot struct {
		ID       int64  `json:"id"`
		Title    string `json:"title"`
		HexColor string `json:"hex_color"`
	}

	var snapshots []labelSnapshot
	for rows.Next() {
		var ls labelSnapshot
		if err := rows.Scan(&ls.ID, &ls.Title, &ls.HexColor); err != nil {
			return "[]", nil
		}
		snapshots = append(snapshots, ls)
	}

	if snapshots == nil {
		return "[]", nil
	}

	data, _ := json.Marshal(snapshots)
	return string(data), nil
}

func (s *SprintService) snapshotComments(tx *sql.Tx, taskID int64) (string, error) {
	rows, err := tx.Query(
		`SELECT author, body, created_at FROM comments WHERE task_id = ? ORDER BY created_at`,
		taskID,
	)
	if err != nil {
		return "[]", nil
	}
	defer rows.Close()

	type commentSnapshot struct {
		Author    string `json:"author"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
	}

	var snapshots []commentSnapshot
	for rows.Next() {
		var cs commentSnapshot
		if err := rows.Scan(&cs.Author, &cs.Body, &cs.CreatedAt); err != nil {
			return "[]", nil
		}
		snapshots = append(snapshots, cs)
	}

	if snapshots == nil {
		return "[]", nil
	}

	data, _ := json.Marshal(snapshots)
	return string(data), nil
}

func generateSprintName() string {
	now := time.Now()
	_, week := now.ISOWeek()
	return fmt.Sprintf("Sprint %d-W%02d", now.Year(), week)
}

func (s *SprintService) ListSprints(db *sql.DB) ([]models.Sprint, error) {
	rows, err := db.Query(
		`SELECT id, name, started_at, ended_at, task_count, summary
		 FROM sprints ORDER BY ended_at DESC`,
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

func (s *SprintService) GetSprintDetail(db *sql.DB, sprintID int64) (*models.SprintDetail, error) {
	var sp models.Sprint
	err := db.QueryRow(
		`SELECT id, name, started_at, ended_at, task_count, summary
		 FROM sprints WHERE id = ?`, sprintID,
	).Scan(&sp.ID, &sp.Name, &sp.StartedAt, &sp.EndedAt, &sp.TaskCount, &sp.Summary)
	if err == sql.ErrNoRows {
		return nil, errors.New("sprint not found")
	}
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`SELECT id, sprint_id, original_task_id, bucket_title, title, description,
		        priority, created_by, due_date, labels, comments, done_at,
		        original_created, archived_at
		 FROM archived_tasks WHERE sprint_id = ? ORDER BY archived_at`, sprintID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.ArchivedTask
	for rows.Next() {
		var at models.ArchivedTask
		if err := rows.Scan(&at.ID, &at.SprintID, &at.OriginalTaskID, &at.BucketTitle,
			&at.Title, &at.Description, &at.Priority, &at.CreatedBy, &at.DueDate,
			&at.Labels, &at.Comments, &at.DoneAt, &at.OriginalCreated, &at.ArchivedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, at)
	}

	return &models.SprintDetail{Sprint: sp, Tasks: tasks}, nil
}
