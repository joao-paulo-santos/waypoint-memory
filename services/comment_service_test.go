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

func setupCommentDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "project.db")
	projDB, err := db.InitProjectDB(dbPath)
	require.NoError(t, err)
	return projDB, func() { projDB.Close() }
}

func TestCommentService_AddComment(t *testing.T) {
	projDB, cleanup := setupCommentDB(t)
	defer cleanup()

	boardSvc := &BoardService{}
	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Commented task",
	})

	commentSvc := &CommentService{}
	comment, err := commentSvc.AddComment(projDB, task.ID, models.CreateCommentRequest{
		Author: "john",
		Body:   "This is a comment",
	})
	require.NoError(t, err)
	assert.Equal(t, "john", comment.Author)
	assert.Equal(t, "This is a comment", comment.Body)
	assert.Equal(t, task.ID, comment.TaskID)
}

func TestCommentService_ListComments(t *testing.T) {
	projDB, cleanup := setupCommentDB(t)
	defer cleanup()

	boardSvc := &BoardService{}
	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Task",
	})

	commentSvc := &CommentService{}
	commentSvc.AddComment(projDB, task.ID, models.CreateCommentRequest{Author: "a", Body: "First"})
	commentSvc.AddComment(projDB, task.ID, models.CreateCommentRequest{Author: "b", Body: "Second"})

	comments, err := commentSvc.ListComments(projDB, task.ID)
	require.NoError(t, err)
	require.Len(t, comments, 2)
	assert.Equal(t, "First", comments[0].Body)
	assert.Equal(t, "Second", comments[1].Body)
}

func TestCommentService_AddComment_EmptyBody(t *testing.T) {
	projDB, cleanup := setupCommentDB(t)
	defer cleanup()

	commentSvc := &CommentService{}
	_, err := commentSvc.AddComment(projDB, 1, models.CreateCommentRequest{Body: ""})
	assert.ErrorIs(t, err, ErrCommentBodyEmpty)
}

func TestCommentService_AddComment_TaskNotFound(t *testing.T) {
	projDB, cleanup := setupCommentDB(t)
	defer cleanup()

	commentSvc := &CommentService{}
	_, err := commentSvc.AddComment(projDB, 999, models.CreateCommentRequest{Body: "comment"})
	assert.ErrorIs(t, err, ErrTaskNotFound)
}

func TestCommentService_DeleteTask_CascadeComments(t *testing.T) {
	projDB, cleanup := setupCommentDB(t)
	defer cleanup()

	boardSvc := &BoardService{}
	task, _ := boardSvc.CreateTask(projDB, models.CreateTaskRequest{
		BucketID: 1,
		Title:    "Task",
	})

	commentSvc := &CommentService{}
	commentSvc.AddComment(projDB, task.ID, models.CreateCommentRequest{Body: "Will be deleted"})

	boardSvc.DeleteTask(projDB, task.ID)

	comments, err := commentSvc.ListComments(projDB, task.ID)
	require.NoError(t, err)
	assert.Len(t, comments, 0)
}
