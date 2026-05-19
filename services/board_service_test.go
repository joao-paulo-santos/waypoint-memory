package services

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBoardTest(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")
	projDB, err := db.InitProjectDB(dbPath)
	require.NoError(t, err)
	return projDB, func() { projDB.Close() }
}

func TestBoardService_GetBoard(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	board, err := svc.GetBoard(projDB)
	require.NoError(t, err)
	require.Len(t, board.Buckets, 3)

	assert.Equal(t, "To-Do", board.Buckets[0].Bucket.Title)
	assert.Equal(t, "In Progress", board.Buckets[1].Bucket.Title)
	assert.Equal(t, "Done", board.Buckets[2].Bucket.Title)
	assert.True(t, board.Buckets[2].Bucket.IsDoneBucket)
}

func TestBoardService_CreateBucket(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	b, err := svc.CreateBucket(projDB, models.CreateBucketRequest{Title: "Review"})
	require.NoError(t, err)
	assert.Equal(t, "Review", b.Title)

	board, _ := svc.GetBoard(projDB)
	assert.Len(t, board.Buckets, 4)
}

func TestBoardService_CreateBucket_AutoPosition(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	b, err := svc.CreateBucket(projDB, models.CreateBucketRequest{Title: "New"})
	require.NoError(t, err)
	assert.Equal(t, float64(400), b.Position)
}

func TestBoardService_CreateBucket_AsDone(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	b, err := svc.CreateBucket(projDB, models.CreateBucketRequest{
		Title:        "New Done",
		IsDoneBucket: true,
	})
	require.NoError(t, err)
	assert.True(t, b.IsDoneBucket)

	board, _ := svc.GetBoard(projDB)
	doneCount := 0
	for _, bw := range board.Buckets {
		if bw.Bucket.IsDoneBucket {
			doneCount++
		}
	}
	assert.Equal(t, 1, doneCount)
}

func TestBoardService_RenameBucket(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	newTitle := "Backlog"
	b, err := svc.UpdateBucket(projDB, 1, models.UpdateBucketRequest{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Backlog", b.Title)
}

func TestBoardService_DeleteBucket(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	err := svc.DeleteBucket(projDB, 1)
	require.NoError(t, err)

	board, _ := svc.GetBoard(projDB)
	assert.Len(t, board.Buckets, 2)
}

func TestBoardService_DeleteBucket_WithTasks(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	_, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Task in To-Do",
	})
	require.NoError(t, err)

	err = svc.DeleteBucket(projDB, 1)
	assert.ErrorIs(t, err, ErrBucketHasTasks)
}

func TestBoardService_DeleteBucket_OnlyDone(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	svc.DeleteBucket(projDB, 1)
	svc.DeleteBucket(projDB, 2)

	err := svc.DeleteBucket(projDB, 3)
	assert.ErrorIs(t, err, ErrBucketOnlyDone)
}

func TestBoardService_ReorderBuckets(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	err := svc.ReorderBuckets(projDB, models.ReorderBucketsRequest{
		BucketIDs: []int64{3, 1, 2},
	})
	require.NoError(t, err)

	board, _ := svc.GetBoard(projDB)
	assert.Equal(t, "Done", board.Buckets[0].Bucket.Title)
	assert.Equal(t, "To-Do", board.Buckets[1].Bucket.Title)
	assert.Equal(t, "In Progress", board.Buckets[2].Bucket.Title)
}

func TestBoardService_CreateTask(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	task, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Fix bug",
	})
	require.NoError(t, err)
	assert.Equal(t, "Fix bug", task.Title)
	assert.Equal(t, int64(1), task.BucketID)
	assert.False(t, task.Done)
}

func TestBoardService_CreateTask_InDoneBucket(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	task, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 3,
		Title:    "Already done",
	})
	require.NoError(t, err)
	assert.True(t, task.Done)
	assert.NotNil(t, task.DoneAt)
}

func TestBoardService_UpdateTask_Merge(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}

	task, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID:    1,
		Title:       "Original",
		Description: "Keep this",
		Priority:    2,
	})
	require.NoError(t, err)

	newTitle := "Updated Title"
	updated, err := svc.UpdateTask(projDB, task.ID, models.UpdateTaskRequest{Title: &newTitle})
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)
	assert.Equal(t, "Keep this", updated.Description)
	assert.Equal(t, 2, updated.Priority)
}

func TestBoardService_MoveTask_ToDone(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	task, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "To complete",
	})
	require.NoError(t, err)
	assert.False(t, task.Done)

	moved, err := svc.MoveTask(projDB, task.ID, 3)
	require.NoError(t, err)
	assert.True(t, moved.Done)
	assert.NotNil(t, moved.DoneAt)
	assert.Equal(t, int64(3), moved.BucketID)
}

func TestBoardService_MoveTask_OutOfDone(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	task, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 3,
		Title:    "Undo this",
	})
	require.NoError(t, err)
	assert.True(t, task.Done)

	moved, err := svc.MoveTask(projDB, task.ID, 1)
	require.NoError(t, err)
	assert.False(t, moved.Done)
	assert.Nil(t, moved.DoneAt)
	assert.Equal(t, int64(1), moved.BucketID)
}

func TestBoardService_MarkTaskDone(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	task, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Mark done",
	})
	require.NoError(t, err)

	done := true
	updated, err := svc.UpdateTask(projDB, task.ID, models.UpdateTaskRequest{Done: &done})
	require.NoError(t, err)
	assert.True(t, updated.Done)
	assert.Equal(t, int64(3), updated.BucketID)
}

func TestBoardService_DeleteTask(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	task, err := svc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Delete me",
	})
	require.NoError(t, err)

	err = svc.DeleteTask(projDB, task.ID)
	require.NoError(t, err)

	_, err = svc.GetTask(projDB, task.ID)
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestBoardService_ReorderTasks(t *testing.T) {
	projDB, cleanup := setupBoardTest(t)
	defer cleanup()

	svc := &BoardService{}
	t1, _ := svc.CreateTask(projDB, models.CreateTaskRequest{BucketID: 1, Title: "A"})
	t2, _ := svc.CreateTask(projDB, models.CreateTaskRequest{BucketID: 1, Title: "B"})
	t3, _ := svc.CreateTask(projDB, models.CreateTaskRequest{BucketID: 1, Title: "C"})

	err := svc.ReorderTasks(projDB, models.ReorderTasksRequest{
		TaskIDs: []int64{t3.ID, t1.ID, t2.ID},
	})
	require.NoError(t, err)

	board, _ := svc.GetBoard(projDB)
	tasks := board.Buckets[0].Tasks
	require.Len(t, tasks, 3)
	assert.Equal(t, "C", tasks[0].Title)
	assert.Equal(t, "A", tasks[1].Title)
	assert.Equal(t, "B", tasks[2].Title)
}
