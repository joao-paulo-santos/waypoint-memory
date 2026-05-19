package services

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSprintTest(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")
	projDB, err := db.InitProjectDB(dbPath)
	require.NoError(t, err)
	return projDB, func() { projDB.Close() }
}

func populateDoneBucket(t *testing.T, projDB *sql.DB) {
	t.Helper()
	boardSvc := &BoardService{}
	labelSvc := &LabelService{}
	commentSvc := &CommentService{}

	labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "bug", HexColor: "#FF0000"})

	task, err := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Task to archive",
		Labels:   "1",
	})
	require.NoError(t, err)

	commentSvc.AddComment(projDB, task.ID, models.CreateCommentRequest{
		Author: "john",
		Body:   "Fixed in commit abc123",
	})

	moved, err := boardSvc.MoveTask(projDB, task.ID, 3)
	require.NoError(t, err)
	require.True(t, moved.Done)
}

func TestSprintService_EndSprint(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	result, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	require.NoError(t, err)

	assert.Equal(t, 1, result.TasksArchived)
	assert.NotEmpty(t, result.Sprint.Name)
	assert.Contains(t, result.Sprint.Name, "Sprint")
}

func TestSprintService_EndSprint_LabelSnapshot(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	result, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	require.NoError(t, err)

	detail, err := svc.GetSprintDetail(projDB, result.Sprint.ID)
	require.NoError(t, err)
	require.Len(t, detail.Tasks, 1)

	var labels []map[string]any
	require.NoError(t, json.Unmarshal([]byte(detail.Tasks[0].Labels), &labels))
	require.Len(t, labels, 1)
	assert.Equal(t, "bug", labels[0]["title"])
	assert.Equal(t, "#FF0000", labels[0]["hex_color"])
}

func TestSprintService_EndSprint_CommentSnapshot(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	result, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	require.NoError(t, err)

	detail, err := svc.GetSprintDetail(projDB, result.Sprint.ID)
	require.NoError(t, err)
	require.Len(t, detail.Tasks, 1)

	var comments []map[string]any
	require.NoError(t, json.Unmarshal([]byte(detail.Tasks[0].Comments), &comments))
	require.Len(t, comments, 1)
	assert.Equal(t, "john", comments[0]["author"])
	assert.Equal(t, "Fixed in commit abc123", comments[0]["body"])
}

func TestSprintService_EndSprint_EmptyDoneBucket(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	svc := &SprintService{}
	_, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	assert.ErrorIs(t, err, ErrEmptyDoneBucket)
}

func TestSprintService_EndSprint_OtherBucketsUntouched(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	boardSvc := &BoardService{}
	boardSvc.CreateTask(projDB, models.CreateTaskRequest{BucketID: 1, Title: "Active task"})
	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	_, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	require.NoError(t, err)

	board, err := boardSvc.GetBoard(projDB)
	require.NoError(t, err)

	require.Len(t, board.Buckets[0].Tasks, 1)
	assert.Equal(t, "Active task", board.Buckets[0].Tasks[0].Title)
	assert.Len(t, board.Buckets[2].Tasks, 0)
}

func TestSprintService_EndSprint_NameCollision(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	svc := &SprintService{}

	populateDoneBucket(t, projDB)
	r1, err := svc.EndSprint(projDB, models.EndSprintRequest{SprintName: "Sprint Test"})
	require.NoError(t, err)
	assert.Equal(t, "Sprint Test", r1.Sprint.Name)

	populateDoneBucket(t, projDB)
	r2, err := svc.EndSprint(projDB, models.EndSprintRequest{SprintName: "Sprint Test"})
	require.NoError(t, err)
	assert.Equal(t, "Sprint Test-2", r2.Sprint.Name)
}

func TestSprintService_EndSprint_DoneBucketCleared(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	result, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.TasksArchived)

	boardSvc := &BoardService{}
	board, err := boardSvc.GetBoard(projDB)
	require.NoError(t, err)

	assert.Equal(t, "Done", board.Buckets[2].Bucket.Title)
	assert.Len(t, board.Buckets[2].Tasks, 0)
}

func TestSprintService_EndSprint_ActivityLogged(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	result, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	require.NoError(t, err)

	var action, details string
	err = projDB.QueryRow(
		`SELECT action, details FROM activity_log WHERE entity_type = 'sprint' AND entity_id = ?`,
		result.Sprint.ID,
	).Scan(&action, &details)
	require.NoError(t, err)
	assert.Equal(t, "sprint_archived", action)

	var d map[string]int
	require.NoError(t, json.Unmarshal([]byte(details), &d))
	assert.Equal(t, 1, d["task_count"])
}

func TestSprintService_ListSprints(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	svc := &SprintService{}

	populateDoneBucket(t, projDB)
	svc.EndSprint(projDB, models.EndSprintRequest{SprintName: "First"})

	populateDoneBucket(t, projDB)
	svc.EndSprint(projDB, models.EndSprintRequest{SprintName: "Second"})

	sprints, err := svc.ListSprints(projDB)
	require.NoError(t, err)
	require.Len(t, sprints, 2)
	names := []string{sprints[0].Name, sprints[1].Name}
	assert.Contains(t, names, "First")
	assert.Contains(t, names, "Second")
}

func TestSprintService_GetSprintDetail(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	result, err := svc.EndSprint(projDB, models.EndSprintRequest{
		SprintName: "Detail Test",
		Summary:    "A productive sprint",
	})
	require.NoError(t, err)

	detail, err := svc.GetSprintDetail(projDB, result.Sprint.ID)
	require.NoError(t, err)
	assert.Equal(t, "Detail Test", detail.Sprint.Name)
	assert.Equal(t, "A productive sprint", detail.Sprint.Summary)
	assert.Equal(t, 1, detail.Sprint.TaskCount)
	require.Len(t, detail.Tasks, 1)
	assert.Equal(t, "Task to archive", detail.Tasks[0].Title)
	assert.Equal(t, "Done", detail.Tasks[0].BucketTitle)
}

func TestSprintService_ConcurrentEndSprint(t *testing.T) {
	projDB, cleanup := setupSprintTest(t)
	defer cleanup()

	populateDoneBucket(t, projDB)

	svc := &SprintService{}
	_, err := svc.EndSprint(projDB, models.EndSprintRequest{})
	require.NoError(t, err)

	_, err = svc.EndSprint(projDB, models.EndSprintRequest{})
	assert.ErrorIs(t, err, ErrEmptyDoneBucket)
}
