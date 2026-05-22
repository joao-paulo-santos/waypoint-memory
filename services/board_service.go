package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrBucketNotFound = errors.New("bucket not found")
	ErrBucketHasTasks = errors.New("bucket has tasks, cannot delete")
	ErrBucketOnlyDone = errors.New("cannot delete the only done bucket")
	ErrTaskNotFound   = errors.New("task not found")
	ErrNoDoneBucket   = errors.New("no done bucket configured")
	ErrBucketWIPLimit = errors.New("bucket has reached WIP limit")
)

type BoardService struct {
	Activity *ActivityService
}

func (s *BoardService) GetBoard(db *sql.DB, projectID int64) (*models.Board, error) {
	buckets, err := s.ListBuckets(db, projectID)
	if err != nil {
		return nil, err
	}

	var result []models.BucketWithTasks
	for _, b := range buckets {
		tasks, err := s.listTasksByBucket(db, b.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, models.BucketWithTasks{Bucket: b, Tasks: tasks})
	}

	return &models.Board{Buckets: result}, nil
}

func (s *BoardService) ListBuckets(db *sql.DB, projectID int64) ([]models.Bucket, error) {
	rows, err := db.Query(
		`SELECT id, title, position, is_done_bucket, wip_limit, created_at, updated_at
		 FROM buckets WHERE project_id = $1 ORDER BY position`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []models.Bucket
	for rows.Next() {
		var b models.Bucket
		if err := rows.Scan(&b.ID, &b.Title, &b.Position, &b.IsDoneBucket, &b.WIPLimit, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, nil
}

func (s *BoardService) CreateBucket(db *sql.DB, projectID int64, req models.CreateBucketRequest) (*models.Bucket, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	if req.Position == 0 {
		var maxPos sql.NullFloat64
		_ = db.QueryRow("SELECT MAX(position) FROM buckets WHERE project_id = $1", projectID).Scan(&maxPos)
		if maxPos.Valid {
			req.Position = maxPos.Float64 + 100
		} else {
			req.Position = 100
		}
	}

	if req.IsDoneBucket {
		if err := s.clearDoneBucket(db, projectID); err != nil {
			return nil, err
		}
	}

	var id int64
	err := db.QueryRow(
		`INSERT INTO buckets (project_id, title, position, is_done_bucket, wip_limit)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		projectID, req.Title, req.Position, req.IsDoneBucket, req.WIPLimit,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "bucket_created",
			EntityType: "bucket",
			EntityID:   id,
			Details:    map[string]any{"title": req.Title},
		})
	}

	return s.GetBucket(db, id)
}

func (s *BoardService) GetBucket(db *sql.DB, id int64) (*models.Bucket, error) {
	b := &models.Bucket{}
	err := db.QueryRow(
		`SELECT id, title, position, is_done_bucket, wip_limit, created_at, updated_at
		 FROM buckets WHERE id = $1`, id,
	).Scan(&b.ID, &b.Title, &b.Position, &b.IsDoneBucket, &b.WIPLimit, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrBucketNotFound
	}
	return b, err
}

func (s *BoardService) UpdateBucket(db *sql.DB, projectID, id int64, req models.UpdateBucketRequest) (*models.Bucket, error) {
	if _, err := s.GetBucket(db, id); err != nil {
		return nil, err
	}

	sets := []string{"updated_at = NOW()"}
	args := []any{id}
	paramIdx := 2

	if req.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", paramIdx))
		args = append(args, *req.Title)
		paramIdx++
	}
	if req.Position != nil {
		sets = append(sets, fmt.Sprintf("position = $%d", paramIdx))
		args = append(args, *req.Position)
		paramIdx++
	}
	if req.IsDoneBucket != nil && *req.IsDoneBucket {
		if err := s.clearDoneBucket(db, projectID); err != nil {
			return nil, err
		}
		sets = append(sets, "is_done_bucket = TRUE")
	}
	if req.WIPLimit != nil {
		sets = append(sets, fmt.Sprintf("wip_limit = $%d", paramIdx))
		args = append(args, *req.WIPLimit)
		paramIdx++
	}

	query := "UPDATE buckets SET " + strings.Join(sets, ", ") + " WHERE id = $1"
	if _, err := db.Exec(query, args...); err != nil {
		return nil, err
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "bucket_updated",
			EntityType: "bucket",
			EntityID:   id,
		})
	}

	return s.GetBucket(db, id)
}

func (s *BoardService) DeleteBucket(db *sql.DB, projectID, id int64) error {
	bucket, err := s.GetBucket(db, id)
	if err != nil {
		return err
	}

	var taskCount int
	if err := db.QueryRow("SELECT count(*) FROM tasks WHERE bucket_id = $1 AND sprint_id IS NULL", id).Scan(&taskCount); err != nil {
		return err
	}
	if taskCount > 0 {
		return ErrBucketHasTasks
	}

	if bucket.IsDoneBucket {
		var doneCount int
		if err := db.QueryRow("SELECT count(*) FROM buckets WHERE project_id = $1 AND is_done_bucket = TRUE", projectID).Scan(&doneCount); err != nil {
			return err
		}
		if doneCount <= 1 {
			return ErrBucketOnlyDone
		}
	}

	_, err = db.Exec("DELETE FROM buckets WHERE id = $1", id)
	if err != nil {
		return err
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "bucket_deleted",
			EntityType: "bucket",
			EntityID:   id,
		})
	}

	return nil
}

func (s *BoardService) ReorderBuckets(db *sql.DB, projectID int64, req models.ReorderBucketsRequest) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range req.BucketIDs {
		pos := float64((i + 1) * 100)
		if _, err := tx.Exec("UPDATE buckets SET position = $1, updated_at = NOW() WHERE id = $2 AND project_id = $3", pos, id, projectID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *BoardService) clearDoneBucket(db *sql.DB, projectID int64) error {
	_, err := db.Exec("UPDATE buckets SET is_done_bucket = FALSE WHERE project_id = $1 AND is_done_bucket = TRUE", projectID)
	return err
}

func (s *BoardService) GetDoneBucketID(db *sql.DB, projectID int64) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM buckets WHERE project_id = $1 AND is_done_bucket = TRUE LIMIT 1", projectID).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNoDoneBucket
	}
	return id, err
}

func (s *BoardService) CreateTask(db *sql.DB, projectID int64, req models.CreateTaskRequest) (*models.Task, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	var position float64
	var maxPos sql.NullFloat64
	_ = db.QueryRow("SELECT MAX(position) FROM tasks WHERE bucket_id = $1 AND project_id = $2 AND sprint_id IS NULL", req.BucketID, projectID).Scan(&maxPos)
	if maxPos.Valid {
		position = maxPos.Float64 + 1
	} else {
		position = 1
	}

	var isDone bool
	err := db.QueryRow("SELECT is_done_bucket FROM buckets WHERE id = $1 AND project_id = $2", req.BucketID, projectID).Scan(&isDone)
	if err != nil {
		return nil, ErrBucketNotFound
	}

	var done bool
	var doneAt any
	if isDone {
		done = true
		doneAt = time.Now().UTC()
	}

	var id int64
	err = db.QueryRow(
		`INSERT INTO tasks (project_id, bucket_id, title, description, position, priority, due_date, done, done_at, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id`,
		projectID, req.BucketID, req.Title, req.Description, position, req.Priority,
		req.DueDate, done, doneAt, req.CreatedBy,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	if req.Labels != "" {
		s.setTaskLabels(db, id, req.Labels)
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "task_created",
			EntityType: "task",
			EntityID:   id,
			Actor:      req.CreatedBy,
			Details:    map[string]any{"title": req.Title, "bucket_id": req.BucketID},
		})
	}

	return s.GetTask(db, id)
}

func (s *BoardService) GetTask(db *sql.DB, id int64) (*models.Task, error) {
	t := &models.Task{}
	err := db.QueryRow(
		`SELECT id, project_id, bucket_id, sprint_id, title, description, position, priority, due_date,
		        done, done_at, created_by, created_at, updated_at
		 FROM tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.ProjectID, &t.BucketID, &t.SprintID, &t.Title, &t.Description, &t.Position, &t.Priority,
		&t.DueDate, &t.Done, &t.DoneAt, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrTaskNotFound
	}

	labels, err := (&LabelService{}).GetTaskLabels(db, t.ID)
	if err != nil {
		return nil, err
	}
	if labels != nil {
		t.Labels = labels
	}
	return t, err
}

func (s *BoardService) UpdateTask(db *sql.DB, projectID, id int64, req models.UpdateTaskRequest) (*models.Task, error) {
	existing, err := s.GetTask(db, id)
	if err != nil {
		return nil, err
	}

	sets := []string{"updated_at = NOW()"}
	args := []any{}
	paramIdx := 1

	if req.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", paramIdx))
		args = append(args, *req.Title)
		paramIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", paramIdx))
		args = append(args, *req.Description)
		paramIdx++
	}
	if req.Priority != nil {
		sets = append(sets, fmt.Sprintf("priority = $%d", paramIdx))
		args = append(args, *req.Priority)
		paramIdx++
	}
	if req.DueDate != nil {
		sets = append(sets, fmt.Sprintf("due_date = $%d", paramIdx))
		args = append(args, *req.DueDate)
		paramIdx++
	}
	if req.BucketID != nil {
		sets = append(sets, fmt.Sprintf("bucket_id = $%d", paramIdx))
		args = append(args, *req.BucketID)
		paramIdx++
	}

	if req.Done != nil {
		if *req.Done && !existing.Done {
			doneBucketID, err := s.GetDoneBucketID(db, projectID)
			if err != nil {
				return nil, err
			}
			sets = append(sets, "done = TRUE", "done_at = NOW()", fmt.Sprintf("bucket_id = $%d", paramIdx))
			args = append(args, doneBucketID)
			paramIdx++
		} else if !*req.Done && existing.Done {
			sets = append(sets, "done = FALSE", "done_at = NULL")
		}
	} else if req.BucketID != nil && !existing.Done {
		var isDone bool
		err := db.QueryRow("SELECT is_done_bucket FROM buckets WHERE id = $1", *req.BucketID).Scan(&isDone)
		if err == nil && isDone {
			sets = append(sets, "done = TRUE", "done_at = NOW()")
		}
	} else if req.BucketID != nil && existing.Done {
		var isDone bool
		err := db.QueryRow("SELECT is_done_bucket FROM buckets WHERE id = $1", *req.BucketID).Scan(&isDone)
		if err == nil && !isDone {
			sets = append(sets, "done = FALSE", "done_at = NULL")
		}
	}

	args = append(args, id)
	query := "UPDATE tasks SET " + strings.Join(sets, ", ") + fmt.Sprintf(" WHERE id = $%d", paramIdx)
	if _, err := db.Exec(query, args...); err != nil {
		return nil, err
	}

	if req.Labels != nil {
		s.setTaskLabels(db, id, *req.Labels)
	}

	if s.Activity != nil {
		action := "task_updated"
		details := map[string]any{"task_id": id}
		if req.Done != nil && *req.Done && !existing.Done {
			action = "task_completed"
		}
		if req.BucketID != nil {
			action = "task_moved"
			details["from_bucket"] = existing.BucketID
			details["to_bucket"] = *req.BucketID
		}
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     action,
			EntityType: "task",
			EntityID:   id,
			Details:    details,
		})
	}

	return s.GetTask(db, id)
}

func (s *BoardService) MoveTask(db *sql.DB, projectID, taskID, targetBucketID int64) (*models.Task, error) {
	req := models.UpdateTaskRequest{BucketID: &targetBucketID}
	return s.UpdateTask(db, projectID, taskID, req)
}

func (s *BoardService) DeleteTask(db *sql.DB, projectID, id int64) error {
	result, err := db.Exec("DELETE FROM tasks WHERE id = $1", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrTaskNotFound
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, projectID, LogActivityParams{
			Action:     "task_deleted",
			EntityType: "task",
			EntityID:   id,
		})
	}

	return nil
}

func (s *BoardService) ReorderTasks(db *sql.DB, projectID int64, req models.ReorderTasksRequest) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range req.TaskIDs {
		pos := float64(i + 1)
		if _, err := tx.Exec("UPDATE tasks SET position = $1, updated_at = NOW() WHERE id = $2 AND project_id = $3", pos, id, projectID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *BoardService) listTasksByBucket(db *sql.DB, bucketID int64) ([]models.Task, error) {
	rows, err := db.Query(
		`SELECT id, project_id, bucket_id, sprint_id, title, description, position, priority, due_date,
		        done, done_at, created_by, created_at, updated_at
		 FROM tasks WHERE bucket_id = $1 AND sprint_id IS NULL ORDER BY position`, bucketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelSvc := &LabelService{}
	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.BucketID, &t.SprintID, &t.Title, &t.Description, &t.Position,
			&t.Priority, &t.DueDate, &t.Done, &t.DoneAt, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}

		labels, err := labelSvc.GetTaskLabels(db, t.ID)
		if err != nil {
			return nil, err
		}
		if labels != nil {
			t.Labels = labels
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *BoardService) setTaskLabels(db *sql.DB, taskID int64, labelsStr string) {
	db.Exec("DELETE FROM task_labels WHERE task_id = $1", taskID)
	if labelsStr == "" {
		return
	}
	parts := strings.Split(labelsStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		var labelID int64
		if _, err := fmt.Sscanf(p, "%d", &labelID); err == nil {
			db.Exec("INSERT INTO task_labels (task_id, label_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", taskID, labelID)
		}
	}
}
