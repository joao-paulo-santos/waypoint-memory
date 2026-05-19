package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAuthHandlerTest(t *testing.T) (*AuthHandler, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	authSvc := services.NewAuthService(centralDB)
	return NewAuthHandler(authSvc), func() { centralDB.Close() }
}

func TestAuthHandler_Status_NoPassword(t *testing.T) {
	h, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/auth/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]bool
	json.NewDecoder(w.Body).Decode(&resp)
	assert.False(t, resp["has_password"])
}

func TestAuthHandler_Login_NoPassword(t *testing.T) {
	h, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"password": "anything"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Login_CorrectPassword(t *testing.T) {
	h, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	require.NoError(t, h.AuthSvc.SetPassword("test123"))

	body, _ := json.Marshal(map[string]string{"password": "test123"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var cookies []*http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "waypoint_session" {
			cookies = append(cookies, c)
		}
	}
	require.Len(t, cookies, 1)
	assert.Equal(t, "authenticated", cookies[0].Value)
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	h, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	require.NoError(t, h.AuthSvc.SetPassword("test123"))

	body, _ := json.Marshal(map[string]string{"password": "wrong"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Middleware_NoPassword(t *testing.T) {
	_, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint2.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	defer centralDB.Close()

	authSvc := services.NewAuthService(centralDB)
	called := false

	r := chi.NewRouter()
	r.Use(AuthMiddleware(authSvc))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Middleware_WithPassword_NoCookie(t *testing.T) {
	_, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint2.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	defer centralDB.Close()

	authSvc := services.NewAuthService(centralDB)
	require.NoError(t, authSvc.SetPassword("test123"))

	called := false

	r := chi.NewRouter()
	r.Use(AuthMiddleware(authSvc))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.False(t, called)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_Middleware_WithPassword_WithCookie(t *testing.T) {
	_, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint2.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	defer centralDB.Close()

	authSvc := services.NewAuthService(centralDB)
	require.NoError(t, authSvc.SetPassword("test123"))

	called := false

	r := chi.NewRouter()
	r.Use(AuthMiddleware(authSvc))
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "waypoint_session", Value: "authenticated"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthHandler_Middleware_ExemptEndpoints(t *testing.T) {
	_, cleanup := setupAuthHandlerTest(t)
	defer cleanup()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "waypoint2.db")
	centralDB, err := db.InitCentralDB(dbPath)
	require.NoError(t, err)
	defer centralDB.Close()

	authSvc := services.NewAuthService(centralDB)
	require.NoError(t, authSvc.SetPassword("test123"))

	called := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	r := chi.NewRouter()
	r.Use(AuthMiddleware(authSvc))
	r.Get("/api/v1/auth/login", handler)
	r.Get("/api/v1/auth/status", handler)

	for _, path := range []string{"/api/v1/auth/login", "/api/v1/auth/status"} {
		called = false
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.True(t, called, "expected %s to be exempt", path)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
