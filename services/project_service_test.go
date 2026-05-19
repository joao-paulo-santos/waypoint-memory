package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*ProjectService, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")

	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)

	projectsDir := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	svc := NewProjectService(centralDB, projectsDir)
	cleanup := func() { centralDB.Close() }

	return svc, cleanup
}

func TestProjectService_Create(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	p, err := svc.Create(models.CreateProjectRequest{Name: "Test Project"})
	require.NoError(t, err)

	assert.Equal(t, "Test Project", p.Name)
	assert.NotEmpty(t, p.UUID)
	assert.DirExists(t, filepath.Join(svc.ProjectsDir, "Test Project", ".waypoint"))
	assert.FileExists(t, filepath.Join(svc.ProjectsDir, "Test Project", ".waypoint", "config.yaml"))
	assert.FileExists(t, filepath.Join(svc.ProjectsDir, "Test Project", ".waypoint", "project.db"))
}

func TestProjectService_Create_DefaultBuckets(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	p, err := svc.Create(models.CreateProjectRequest{Name: "BucketTest"})
	require.NoError(t, err)

	projDB, err := svc.GetProjectDB(p.ID)
	require.NoError(t, err)
	defer projDB.Close()

	var count int
	err = projDB.QueryRow("SELECT count(*) FROM buckets").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	var doneCount int
	err = projDB.QueryRow("SELECT count(*) FROM buckets WHERE is_done_bucket = 1").Scan(&doneCount)
	require.NoError(t, err)
	assert.Equal(t, 1, doneCount)
}

func TestProjectService_Init(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	existingDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(existingDir, "README.md"), []byte("# Test"), 0644))

	p, err := svc.Init(models.InitProjectRequest{Path: existingDir})
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(existingDir, ".waypoint"))
	assert.FileExists(t, filepath.Join(existingDir, "README.md"))
	assert.Equal(t, filepath.Base(existingDir), p.Name)
}

func TestProjectService_Init_CustomName(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	existingDir := t.TempDir()

	p, err := svc.Init(models.InitProjectRequest{
		Path: existingDir,
		Name: "My Custom Name",
	})
	require.NoError(t, err)
	assert.Equal(t, "My Custom Name", p.Name)
}

func TestProjectService_Add(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	projectDir := t.TempDir()
	_, err := db.InitProjectDir(projectDir, "Existing Project", "Desc")
	require.NoError(t, err)

	p, err := svc.Add(models.AddProjectRequest{Path: projectDir})
	require.NoError(t, err)

	assert.Equal(t, "Existing Project", p.Name)
	assert.NotEmpty(t, p.UUID)
}

func TestProjectService_Add_NoWaypoint(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	emptyDir := t.TempDir()

	_, err := svc.Add(models.AddProjectRequest{Path: emptyDir})
	assert.ErrorIs(t, err, ErrWaypointNotFound)
}

func TestProjectService_Add_RelativePath(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	_, err := svc.Add(models.AddProjectRequest{Path: "relative/path"})
	assert.ErrorIs(t, err, ErrProjectPathNotAbsolute)
}

func TestProjectService_Create_DuplicatePath(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	_, err := svc.Create(models.CreateProjectRequest{Name: "Same"})
	require.NoError(t, err)

	_, err = svc.Create(models.CreateProjectRequest{Name: "Same"})
	assert.ErrorIs(t, err, ErrProjectAlreadyRegistered)
}

func TestProjectService_Nesting_ParentChild(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	parentDir := t.TempDir()
	childDir := filepath.Join(parentDir, "child")
	require.NoError(t, os.MkdirAll(childDir, 0755))

	_, err := svc.Init(models.InitProjectRequest{Path: parentDir})
	require.NoError(t, err)

	_, err = svc.Init(models.InitProjectRequest{Path: childDir})
	assert.ErrorIs(t, err, ErrProjectNested)

	svc2, cleanup2 := setupTestService(t)
	defer cleanup2()

	_, err = svc2.Init(models.InitProjectRequest{Path: childDir})
	require.NoError(t, err)

	_, err = svc2.Init(models.InitProjectRequest{Path: parentDir})
	assert.ErrorIs(t, err, ErrProjectNested)
}

func TestProjectService_List(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	_, err := svc.Create(models.CreateProjectRequest{Name: "Alpha"})
	require.NoError(t, err)
	_, err = svc.Create(models.CreateProjectRequest{Name: "Beta"})
	require.NoError(t, err)

	projects, err := svc.List()
	require.NoError(t, err)
	require.Len(t, projects, 2)
	assert.Equal(t, "Alpha", projects[0].Name)
	assert.Equal(t, "Beta", projects[1].Name)
}

func TestProjectService_GetByID(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	p, err := svc.Create(models.CreateProjectRequest{Name: "Test"})
	require.NoError(t, err)

	got, err := svc.GetByID(p.ID)
	require.NoError(t, err)
	assert.Equal(t, p.UUID, got.UUID)
	assert.Equal(t, "Test", got.Name)
}

func TestProjectService_GetByID_NotFound(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	_, err := svc.GetByID(999)
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestProjectService_Update(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	p, err := svc.Create(models.CreateProjectRequest{Name: "Original"})
	require.NoError(t, err)

	newName := "Updated"
	newDesc := "New description"
	updated, err := svc.Update(p.ID, models.UpdateProjectRequest{
		Name:        &newName,
		Description: &newDesc,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.Equal(t, "New description", updated.Description)
}

func TestProjectService_Update_PartialMerge(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	p, err := svc.Create(models.CreateProjectRequest{
		Name:        "Original",
		Description: "Keep this",
	})
	require.NoError(t, err)

	newName := "Renamed"
	updated, err := svc.Update(p.ID, models.UpdateProjectRequest{Name: &newName})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Name)
	assert.Equal(t, "Keep this", updated.Description)
}

func TestProjectService_Delete(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	p, err := svc.Create(models.CreateProjectRequest{Name: "ToDelete"})
	require.NoError(t, err)

	projectPath := p.Path

	err = svc.Delete(p.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(p.ID)
	assert.ErrorIs(t, err, ErrProjectNotFound)

	assert.DirExists(t, filepath.Join(projectPath, ".waypoint"))
}

func TestProjectService_GetProjectDetail(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	p, err := svc.Create(models.CreateProjectRequest{Name: "Detail Test"})
	require.NoError(t, err)

	detail, err := svc.GetProjectDetail(p.ID)
	require.NoError(t, err)

	assert.Equal(t, p.UUID, detail.Project.UUID)
	assert.Equal(t, 3, len(detail.BoardSummary.Buckets))
	assert.Equal(t, 0, detail.BoardSummary.TotalActive)
	assert.Equal(t, 0, detail.BoardSummary.TotalDone)
}
