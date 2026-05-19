package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupWikiTest(t *testing.T) (string, string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	waypointDir := filepath.Join(tmpDir, ".waypoint")
	wikiDir := filepath.Join(waypointDir, "wiki")
	os.MkdirAll(wikiDir, 0755)

	os.WriteFile(filepath.Join(wikiDir, "index.md"), []byte("# Welcome"), 0644)
	os.WriteFile(filepath.Join(wikiDir, "architecture.md"), []byte("# Architecture\n\nSystem design."), 0644)

	notesDir := filepath.Join(wikiDir, "meeting-notes", "2026")
	os.MkdirAll(notesDir, 0755)
	os.WriteFile(filepath.Join(notesDir, "standup.md"), []byte("## Standup Notes"), 0644)

	return tmpDir, waypointDir, func() {}
}

func TestWikiService_ListWikiPages(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	tree, err := svc.ListWikiPages(waypointDir)
	require.NoError(t, err)
	require.Len(t, tree, 3)

	assert.Equal(t, "meeting-notes", tree[0].Name)
	assert.True(t, tree[0].IsDir)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "2026", tree[0].Children[0].Name)

	assert.Equal(t, "architecture.md", tree[1].Name)
	assert.False(t, tree[1].IsDir)

	assert.Equal(t, "index.md", tree[2].Name)
	assert.False(t, tree[2].IsDir)
}

func TestWikiService_ReadWikiPage(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	page, err := svc.ReadWikiPage(waypointDir, "architecture.md")
	require.NoError(t, err)
	assert.Equal(t, "architecture", page.Title)
	assert.Equal(t, "# Architecture\n\nSystem design.", page.Content)
}

func TestWikiService_ReadNestedPage(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	page, err := svc.ReadWikiPage(waypointDir, "meeting-notes/2026/standup.md")
	require.NoError(t, err)
	assert.Equal(t, "standup", page.Title)
	assert.Equal(t, "## Standup Notes", page.Content)
}

func TestWikiService_PathTraversal(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	_, err := svc.ReadWikiPage(waypointDir, "../../etc/passwd")
	assert.ErrorIs(t, err, ErrPathTraversal)
}

func TestWikiService_AbsolutePath(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	_, err := svc.ReadWikiPage(waypointDir, "/etc/passwd")
	assert.ErrorIs(t, err, ErrPathTraversal)
}

func TestWikiService_MissingPage(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	_, err := svc.ReadWikiPage(waypointDir, "nonexistent.md")
	assert.ErrorIs(t, err, ErrWikiNotFound)
}

func TestWikiService_DocsNotConfigured(t *testing.T) {
	tmpDir, _, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	_, err := svc.ListDocsPages(tmpDir, "")
	assert.ErrorIs(t, err, ErrDocsNotEnabled)
}

func TestWikiService_DocsWithSubfolders(t *testing.T) {
	tmpDir, _, cleanup := setupWikiTest(t)
	defer cleanup()

	docsDir := filepath.Join(tmpDir, "docs", "api")
	os.MkdirAll(docsDir, 0755)
	os.WriteFile(filepath.Join(docsDir, "endpoints.md"), []byte("# API"), 0644)
	os.WriteFile(filepath.Join(filepath.Join(tmpDir, "docs"), "readme.md"), []byte("# Docs"), 0644)

	svc := NewWikiService()
	tree, err := svc.ListDocsPages(tmpDir, "docs")
	require.NoError(t, err)
	require.Len(t, tree, 2)
	assert.Equal(t, "api", tree[0].Name)
	assert.True(t, tree[0].IsDir)
	assert.Equal(t, "readme.md", tree[1].Name)
}

func TestWikiService_HiddenFilesExcluded(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	wikiDir := filepath.Join(waypointDir, "wiki")
	os.WriteFile(filepath.Join(wikiDir, ".hidden.md"), []byte("hidden"), 0644)

	svc := NewWikiService()
	tree, err := svc.ListWikiPages(waypointDir)
	require.NoError(t, err)
	for _, node := range tree {
		assert.False(t, node.Name[0] == '.', "hidden file %s should be excluded", node.Name)
	}
}

func TestWikiService_FoldersBeforeFiles(t *testing.T) {
	_, waypointDir, cleanup := setupWikiTest(t)
	defer cleanup()

	svc := NewWikiService()
	tree, err := svc.ListWikiPages(waypointDir)
	require.NoError(t, err)
	require.True(t, len(tree) >= 2, "need at least 2 entries")
	assert.True(t, tree[0].IsDir, "first entry should be a folder")
}
