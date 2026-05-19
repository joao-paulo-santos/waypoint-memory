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

func (s *BoardService) GetBoard(db *sql.DB) (*models.Board, error) {
	buckets, err := s.ListBuckets(db)
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

func (s *BoardService) ListBuckets(db *sql.DB) ([]models.Bucket, error) {
	rows, err := db.Query(
		`SELECT id, title, position, is_done_bucket, wip_limit, created_at, updated_at
		 FROM buckets ORDER BY position`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []models.Bucket
	for rows.Next() {
		var b models.Bucket
		var isDone int
		if err := rows.Scan(&b.ID, &b.Title, &b.Position, &isDone, &b.WIPLimit, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		b.IsDoneBucket = isDone == 1
		buckets = append(buckets, b)
	}
	return buckets, nil
}

func (s *BoardService) CreateBucket(db *sql.DB, req models.CreateBucketRequest) (*models.Bucket, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	if req.Position == 0 {
		var maxPos sql.NullFloat64
		_ = db.QueryRow("SELECT MAX(position) FROM buckets").Scan(&maxPos)
		if maxPos.Valid {
			req.Position = maxPos.Float64 + 100
		} else {
			req.Position = 100
		}
	}

	if req.IsDoneBucket {
		if err := s.clearDoneBucket(db); err != nil {
			return nil, err
		}
	}

	result, err := db.Exec(
		`INSERT INTO buckets (title, position, is_done_bucket, wip_limit)
		 VALUES (?, ?, ?, ?)`,
		req.Title, req.Position, req.IsDoneBucket, req.WIPLimit,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	if s.Activity != nil {
		s.Activity.LogActivity(db, LogActivityParams{
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
	var isDone int
	err := db.QueryRow(
		`SELECT id, title, position, is_done_bucket, wip_limit, created_at, updated_at
		 FROM buckets WHERE id = ?`, id,
	).Scan(&b.ID, &b.Title, &b.Position, &isDone, &b.WIPLimit, &b.CreatedAt, &b.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrBucketNotFound
	}
	b.IsDoneBucket = isDone == 1
	return b, err
}

func (s *BoardService) UpdateBucket(db *sql.DB, id int64, req models.UpdateBucketRequest) (*models.Bucket, error) {
	if _, err := s.GetBucket(db, id); err != nil {
		return nil, err
	}

	sets := []string{"updated_at = datetime('now')"}
	args := []any{}

	if req.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Position != nil {
		sets = append(sets, "position = ?")
		args = append(args, *req.Position)
	}
	if req.IsDoneBucket != nil && *req.IsDoneBucket {
		if err := s.clearDoneBucket(db); err != nil {
			return nil, err
		}
		sets = append(sets, "is_done_bucket = 1")
	}
	if req.WIPLimit != nil {
		sets = append(sets, "wip_limit = ?")
		args = append(args, *req.WIPLimit)
	}

	args = append(args, id)
	query := "UPDATE buckets SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := db.Exec(query, args...); err != nil {
		return nil, err
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, LogActivityParams{
			Action:     "bucket_updated",
			EntityType: "bucket",
			EntityID:   id,
		})
	}

	return s.GetBucket(db, id)
}

func (s *BoardService) DeleteBucket(db *sql.DB, id int64) error {
	bucket, err := s.GetBucket(db, id)
	if err != nil {
		return err
	}

	var taskCount int
	if err := db.QueryRow("SELECT count(*) FROM tasks WHERE bucket_id = ?", id).Scan(&taskCount); err != nil {
		return err
	}
	if taskCount > 0 {
		return ErrBucketHasTasks
	}

	if bucket.IsDoneBucket {
		var doneCount int
		if err := db.QueryRow("SELECT count(*) FROM buckets WHERE is_done_bucket = 1").Scan(&doneCount); err != nil {
			return err
		}
		if doneCount <= 1 {
			return ErrBucketOnlyDone
		}
	}

	_, err = db.Exec("DELETE FROM buckets WHERE id = ?", id)
	if err != nil {
		return err
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, LogActivityParams{
			Action:     "bucket_deleted",
			EntityType: "bucket",
			EntityID:   id,
		})
	}

	return nil
}

func (s *BoardService) ReorderBuckets(db *sql.DB, req models.ReorderBucketsRequest) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range req.BucketIDs {
		pos := float64((i + 1) * 100)
		if _, err := tx.Exec("UPDATE buckets SET position = ?, updated_at = datetime('now') WHERE id = ?", pos, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *BoardService) clearDoneBucket(db *sql.DB) error {
	_, err := db.Exec("UPDATE buckets SET is_done_bucket = 0 WHERE is_done_bucket = 1")
	return err
}

func (s *BoardService) GetDoneBucketID(db *sql.DB) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM buckets WHERE is_done_bucket = 1 LIMIT 1").Scan(&id)
	if err == sql.ErrNoRows {
		return 0, ErrNoDoneBucket
	}
	return id, err
}

func (s *BoardService) CreateTask(db *sql.DB, req models.CreateTaskRequest) (*models.Task, error) {
	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	var position float64
	var maxPos sql.NullFloat64
	_ = db.QueryRow("SELECT MAX(position) FROM tasks WHERE bucket_id = ?", req.BucketID).Scan(&maxPos)
	if maxPos.Valid {
		position = maxPos.Float64 + 1
	} else {
		position = 1
	}

	var isDone int
	err := db.QueryRow("SELECT is_done_bucket FROM buckets WHERE id = ?", req.BucketID).Scan(&isDone)
	if err != nil {
		return nil, ErrBucketNotFound
	}

	var done int
	var doneAt any
	if isDone == 1 {
		done = 1
		doneAt = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	}

	result, err := db.Exec(
		`INSERT INTO tasks (bucket_id, title, description, position, priority, due_date, done, done_at, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.BucketID, req.Title, req.Description, position, req.Priority,
		req.DueDate, done, doneAt, req.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	if req.Labels != "" {
		s.setTaskLabels(db, id, req.Labels)
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, LogActivityParams{
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
	var done int
	err := db.QueryRow(
		`SELECT id, bucket_id, title, description, position, priority, due_date,
		        done, done_at, created_by, created_at, updated_at
		 FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.BucketID, &t.Title, &t.Description, &t.Position, &t.Priority,
		&t.DueDate, &done, &t.DoneAt, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrTaskNotFound
	}
	t.Done = done == 1

	labels, err := (&LabelService{}).GetTaskLabels(db, t.ID)
	if err != nil {
		return nil, err
	}
	if labels != nil {
		t.Labels = labels
	}
	return t, err
}

func (s *BoardService) UpdateTask(db *sql.DB, id int64, req models.UpdateTaskRequest) (*models.Task, error) {
	existing, err := s.GetTask(db, id)
	if err != nil {
		return nil, err
	}

	sets := []string{"updated_at = datetime('now')"}
	args := []any{}

	if req.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *req.Title)
	}
	if req.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *req.Description)
	}
	if req.Priority != nil {
		sets = append(sets, "priority = ?")
		args = append(args, *req.Priority)
	}
	if req.DueDate != nil {
		sets = append(sets, "due_date = ?")
		args = append(args, *req.DueDate)
	}
	if req.BucketID != nil {
		sets = append(sets, "bucket_id = ?")
		args = append(args, *req.BucketID)
	}

	if req.Done != nil {
		if *req.Done && !existing.Done {
			doneBucketID, err := s.GetDoneBucketID(db)
			if err != nil {
				return nil, err
			}
			sets = append(sets, "done = 1", "done_at = datetime('now')", "bucket_id = ?")
			args = append(args, doneBucketID)
		} else if !*req.Done && existing.Done {
			sets = append(sets, "done = 0", "done_at = NULL")
		}
	} else if req.BucketID != nil && !existing.Done {
		var isDone int
		err := db.QueryRow("SELECT is_done_bucket FROM buckets WHERE id = ?", *req.BucketID).Scan(&isDone)
		if err == nil && isDone == 1 {
			sets = append(sets, "done = 1", "done_at = datetime('now')")
		}
	} else if req.BucketID != nil && existing.Done {
		var isDone int
		err := db.QueryRow("SELECT is_done_bucket FROM buckets WHERE id = ?", *req.BucketID).Scan(&isDone)
		if err == nil && isDone == 0 {
			sets = append(sets, "done = 0", "done_at = NULL")
		}
	}

	args = append(args, id)
	query := "UPDATE tasks SET " + strings.Join(sets, ", ") + " WHERE id = ?"
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
		s.Activity.LogActivity(db, LogActivityParams{
			Action:     action,
			EntityType: "task",
			EntityID:   id,
			Details:    details,
		})
	}

	return s.GetTask(db, id)
}

func (s *BoardService) MoveTask(db *sql.DB, taskID, targetBucketID int64) (*models.Task, error) {
	req := models.UpdateTaskRequest{BucketID: &targetBucketID}
	return s.UpdateTask(db, taskID, req)
}

func (s *BoardService) DeleteTask(db *sql.DB, id int64) error {
	result, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrTaskNotFound
	}

	if s.Activity != nil {
		s.Activity.LogActivity(db, LogActivityParams{
			Action:     "task_deleted",
			EntityType: "task",
			EntityID:   id,
		})
	}

	return nil
}

func (s *BoardService) ReorderTasks(db *sql.DB, req models.ReorderTasksRequest) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range req.TaskIDs {
		pos := float64(i + 1)
		if _, err := tx.Exec("UPDATE tasks SET position = ?, updated_at = datetime('now') WHERE id = ?", pos, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *BoardService) listTasksByBucket(db *sql.DB, bucketID int64) ([]models.Task, error) {
	rows, err := db.Query(
		`SELECT id, bucket_id, title, description, position, priority, due_date,
		        done, done_at, created_by, created_at, updated_at
		 FROM tasks WHERE bucket_id = ? ORDER BY position`, bucketID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labelSvc := &LabelService{}
	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var done int
		if err := rows.Scan(&t.ID, &t.BucketID, &t.Title, &t.Description, &t.Position,
			&t.Priority, &t.DueDate, &done, &t.DoneAt, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Done = done == 1

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
	db.Exec("DELETE FROM task_labels WHERE task_id = ?", taskID)
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
			db.Exec("INSERT OR IGNORE INTO task_labels (task_id, label_id) VALUES (?, ?)", taskID, labelID)
		}
	}
}
