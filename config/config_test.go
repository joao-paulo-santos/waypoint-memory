package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultDataDir(t *testing.T) {
	dir := defaultDataDir()
	expected, _ := os.UserConfigDir()
	assert.Equal(t, filepath.Join(expected, "Waypoint"), dir)
}

func TestCentralDBPath(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/test-waypoint"}
	assert.Equal(t, "/tmp/test-waypoint/waypoint.db", cfg.CentralDBPath())
}

func TestProjectsDir(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/test-waypoint"}
	assert.Equal(t, "/tmp/test-waypoint/projects", cfg.ProjectsDir())
}

func TestEnsureDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{DataDir: filepath.Join(tmpDir, "waypoint-test")}
	require.NoError(t, cfg.EnsureDataDir())
	_, err := os.Stat(cfg.DataDir)
	assert.NoError(t, err)
}

func TestEnsureProjectsDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{DataDir: filepath.Join(tmpDir, "waypoint-test")}
	require.NoError(t, cfg.EnsureProjectsDir())
	_, err := os.Stat(cfg.ProjectsDir())
	assert.NoError(t, err)
}

func TestHandleVersion(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"waypoint", "-version"}
	assert.True(t, HandleVersion())

	os.Args = []string{"waypoint"}
	assert.False(t, HandleVersion())

	os.Args = []string{"waypoint", "-webAddr", ":8080"}
	assert.False(t, HandleVersion())
}
