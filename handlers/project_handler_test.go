package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"github.com/joao-paulo-santos/waypoint-memory/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupHandler(t *testing.T) (*ProjectHandler, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)

	projectsDir := filepath.Join(tmpDir, "projects")
	require.NoError(t, os.MkdirAll(projectsDir, 0755))

	svc := services.NewProjectService(centralDB, projectsDir)
	handler := NewProjectHandler(svc)
	cleanup := func() { centralDB.Close() }

	return handler, cleanup
}

func makeRequest(t *testing.T, handler http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(data)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestProjectHandler_Create(t *testing.T) {
	h, cleanup := setupHandler(t)
	defer cleanup()

	rec := makeRequest(t, h.Routes(), http.MethodPost, "/create",
		models.CreateProjectRequest{Name: "Test"})

	assert.Equal(t, http.StatusCreated, rec.Code)

	var p models.Project
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))
	assert.Equal(t, "Test", p.Name)
	assert.NotEmpty(t, p.UUID)
}

func TestProjectHandler_List(t *testing.T) {
	h, cleanup := setupHandler(t)
	defer cleanup()

	makeRequest(t, h.Routes(), http.MethodPost, "/create",
		models.CreateProjectRequest{Name: "A"})
	makeRequest(t, h.Routes(), http.MethodPost, "/create",
		models.CreateProjectRequest{Name: "B"})

	rec := makeRequest(t, h.Routes(), http.MethodGet, "/", nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	projects := result["projects"].([]any)
	assert.Len(t, projects, 2)
}

func TestProjectHandler_Get(t *testing.T) {
	h, cleanup := setupHandler(t)
	defer cleanup()

	rec := makeRequest(t, h.Routes(), http.MethodPost, "/create",
		models.CreateProjectRequest{Name: "Test"})
	var p models.Project
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))

	rec = makeRequest(t, h.Routes(), http.MethodGet, "/"+fmt.Sprintf("%d", p.ID), nil)
	assert.Equal(t, http.StatusOK, rec.Code)

	var detail models.ProjectDetail
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&detail))
	assert.Equal(t, p.ID, detail.Project.ID)
}

func TestProjectHandler_Update(t *testing.T) {
	h, cleanup := setupHandler(t)
	defer cleanup()

	rec := makeRequest(t, h.Routes(), http.MethodPost, "/create",
		models.CreateProjectRequest{Name: "Original"})
	var p models.Project
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))

	newName := "Updated"
	rec = makeRequest(t, h.Routes(), http.MethodPut, "/"+fmt.Sprintf("%d", p.ID),
		models.UpdateProjectRequest{Name: &newName})
	assert.Equal(t, http.StatusOK, rec.Code)

	var updated models.Project
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&updated))
	assert.Equal(t, "Updated", updated.Name)
}

func TestProjectHandler_Delete(t *testing.T) {
	h, cleanup := setupHandler(t)
	defer cleanup()

	rec := makeRequest(t, h.Routes(), http.MethodPost, "/create",
		models.CreateProjectRequest{Name: "DeleteMe"})
	var p models.Project
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&p))

	rec = makeRequest(t, h.Routes(), http.MethodDelete, "/"+fmt.Sprintf("%d", p.ID), nil)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = makeRequest(t, h.Routes(), http.MethodGet, "/"+fmt.Sprintf("%d", p.ID), nil)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestProjectHandler_Init_NonAbsolute(t *testing.T) {
	h, cleanup := setupHandler(t)
	defer cleanup()

	rec := makeRequest(t, h.Routes(), http.MethodPost, "/init",
		models.InitProjectRequest{Path: "relative/path"})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestProjectHandler_Add_NoWaypoint(t *testing.T) {
	h, cleanup := setupHandler(t)
	defer cleanup()

	emptyDir := t.TempDir()

	rec := makeRequest(t, h.Routes(), http.MethodPost, "/add",
		models.AddProjectRequest{Path: emptyDir})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
