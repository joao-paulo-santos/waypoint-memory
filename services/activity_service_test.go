package services

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupActivityTest(t *testing.T) (*sql.DB, *BoardService, *ActivityService, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")
	projDB, err := db.InitProjectDB(dbPath)
	require.NoError(t, err)

	activitySvc := NewActivityService()
	boardSvc := &BoardService{Activity: activitySvc}

	return projDB, boardSvc, activitySvc, func() { projDB.Close() }
}

func TestActivityService_TaskCreated(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	_, err := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "My task",
	})
	require.NoError(t, err)

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "task_created", entries[0].Action)
	assert.Equal(t, "task", entries[0].EntityType)

	var details map[string]any
	require.NoError(t, json.Unmarshal([]byte(entries[0].Details), &details))
	assert.Equal(t, "My task", details["title"])
}

func TestActivityService_TaskMoved(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Move me",
	})

	boardSvc.MoveTask(projDB, task.ID, 3)

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "task_moved", entries[0].Action)
}

func TestActivityService_TaskCompleted(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Complete me",
	})

	done := true
	boardSvc.UpdateTask(projDB, task.ID, models.UpdateTaskRequest{Done: &done})

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "task_completed", entries[0].Action)
}

func TestActivityService_TaskDeleted(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Delete me",
	})

	boardSvc.DeleteTask(projDB, task.ID)

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "task_deleted", entries[0].Action)
}

func TestActivityService_BucketCreated(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	boardSvc.CreateBucket(projDB, models.CreateBucketRequest{Title: "Review"})

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "bucket_created", entries[0].Action)
}

func TestActivityService_BucketDeleted(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	boardSvc.DeleteBucket(projDB, 1)

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "bucket_deleted", entries[0].Action)
}

func TestActivityService_LabelCreated(t *testing.T) {
	projDB, _, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	labelSvc := &LabelService{Activity: activitySvc}
	labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "bug"})

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "label_created", entries[0].Action)
}

func TestActivityService_CommentAdded(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Task",
	})

	commentSvc := &CommentService{Activity: activitySvc}
	commentSvc.AddComment(projDB, task.ID, models.CreateCommentRequest{
		Author: "john",
		Body:   "Looks good",
	})

	entries, err := activitySvc.GetProjectActivity(projDB, 10)
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "comment_added", entries[0].Action)
}

func TestActivityService_MultipleOperations(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Sprint task",
	})
	boardSvc.MoveTask(projDB, task.ID, 3)

	sprintSvc := &SprintService{}
	sprintSvc.EndSprint(projDB, models.EndSprintRequest{})

	entries, err := activitySvc.GetProjectActivity(projDB, 20)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	assert.Equal(t, "sprint_archived", entries[0].Action)
}

func TestActivityService_Limit(t *testing.T) {
	projDB, boardSvc, activitySvc, cleanup := setupActivityTest(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		boardSvc.CreateTask(projDB, models.CreateTaskRequest{
			BucketID: 1,
			Title:    "Task",
		})
	}

	entries, err := activitySvc.GetProjectActivity(projDB, 3)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestActivityService_GlobalActivity(t *testing.T) {
	tmpDir := t.TempDir()

	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	defer centralDB.Close()

	projectsDir := filepath.Join(tmpDir, "projects")
	os.MkdirAll(projectsDir, 0755)

	projectSvc := NewProjectService(centralDB, projectsDir)
	p1, _ := projectSvc.Create(models.CreateProjectRequest{Name: "Project A"})
	p2, _ := projectSvc.Create(models.CreateProjectRequest{Name: "Project B"})

	activitySvc := NewActivityService()
	boardSvc := &BoardService{Activity: activitySvc}

	projDB1, _ := projectSvc.GetProjectDB(p1.ID)
	defer projDB1.Close()
	boardSvc.CreateTask(projDB1, models.CreateTaskRequest{BucketID: 1, Title: "A task"})

	projDB2, _ := projectSvc.GetProjectDB(p2.ID)
	defer projDB2.Close()
	boardSvc.CreateTask(projDB2, models.CreateTaskRequest{BucketID: 1, Title: "B task"})

	globalEntries, err := activitySvc.GetGlobalActivity(centralDB, projectSvc, 10)
	require.NoError(t, err)
	assert.Len(t, globalEntries, 2)
}
