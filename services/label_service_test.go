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

func setupLabelDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")
	projDB, err := db.InitProjectDB(dbPath)
	require.NoError(t, err)
	return projDB, func() { projDB.Close() }
}

func TestLabelService_Create(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	svc := &LabelService{}
	label, err := svc.CreateLabel(projDB, models.CreateLabelRequest{
		Title:    "bug",
		HexColor: "#FF0000",
	})
	require.NoError(t, err)
	assert.Equal(t, "bug", label.Title)
	assert.Equal(t, "#FF0000", label.HexColor)
	assert.Equal(t, int64(1), label.ID)
}

func TestLabelService_Create_DuplicateTitle(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	svc := &LabelService{}
	_, err := svc.CreateLabel(projDB, models.CreateLabelRequest{Title: "bug"})
	require.NoError(t, err)

	_, err = svc.CreateLabel(projDB, models.CreateLabelRequest{Title: "bug"})
	assert.ErrorIs(t, err, ErrLabelDuplicate)
}

func TestLabelService_List(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	svc := &LabelService{}
	svc.CreateLabel(projDB, models.CreateLabelRequest{Title: "alpha", HexColor: "#FF0000"})
	svc.CreateLabel(projDB, models.CreateLabelRequest{Title: "beta", HexColor: "#00FF00"})

	labels, err := svc.ListLabels(projDB)
	require.NoError(t, err)
	require.Len(t, labels, 2)
	assert.Equal(t, "alpha", labels[0].Title)
	assert.Equal(t, "beta", labels[1].Title)
}

func TestLabelService_Delete(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	svc := &LabelService{}
	label, _ := svc.CreateLabel(projDB, models.CreateLabelRequest{Title: "remove-me"})

	err := svc.DeleteLabel(projDB, label.ID)
	require.NoError(t, err)

	labels, _ := svc.ListLabels(projDB)
	assert.Len(t, labels, 0)
}

func TestLabelService_Delete_NotFound(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	svc := &LabelService{}
	err := svc.DeleteLabel(projDB, 999)
	assert.ErrorIs(t, err, ErrLabelNotFound)
}

func TestLabelService_TaskLabels(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	labelSvc := &LabelService{}
	boardSvc := &BoardService{}

	labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "bug", HexColor: "#FF0000"})
	labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "feature", HexColor: "#00FF00"})

	task, err := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Labeled task",
		Labels:   "1,2",
	})
	require.NoError(t, err)

	labels, err := labelSvc.GetTaskLabels(projDB, task.ID)
	require.NoError(t, err)
	require.Len(t, labels, 2)
	assert.Equal(t, "bug", labels[0].Title)
	assert.Equal(t, "feature", labels[1].Title)
}

func TestLabelService_UpdateTaskLabels(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	labelSvc := &LabelService{}
	boardSvc := &BoardService{}

	labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "bug"})
	labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "feature"})
	labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "docs"})

	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Task",
		Labels:   "1,2",
	})

	newLabels := "2,3"
	_, err := boardSvc.UpdateTask(projDB, task.ID, models.UpdateTaskRequest{Labels: &newLabels})
	require.NoError(t, err)

	labels, _ := labelSvc.GetTaskLabels(projDB, task.ID)
	require.Len(t, labels, 2)
	assert.Equal(t, "docs", labels[0].Title)
	assert.Equal(t, "feature", labels[1].Title)
}

func TestLabelService_DeleteLabel_RemovesFromTasks(t *testing.T) {
	projDB, cleanup := setupLabelDB(t)
	defer cleanup()

	labelSvc := &LabelService{}
	boardSvc := &BoardService{}

	label, _ := labelSvc.CreateLabel(projDB, models.CreateLabelRequest{Title: "bug"})
	boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Task",
		Labels:   "1",
	})

	labelSvc.DeleteLabel(projDB, label.ID)

	task, _ := boardSvc.GetTask(projDB, 1)
	assert.Len(t, task.Labels, 0)
}
