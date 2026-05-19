package services

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrWikiNotFound   = errors.New("wiki page not found")
	ErrDocsNotEnabled = errors.New("docs path not configured for this project")
	ErrPathTraversal  = errors.New("invalid path: traversal not allowed")
)

type WikiService struct{}

func NewWikiService() *WikiService {
	return &WikiService{}
}

func (s *WikiService) ListWikiPages(waypointDir string) ([]*models.FileNode, error) {
	wikiDir := filepath.Join(waypointDir, "wiki")
	return s.buildTree(wikiDir, "")
}

func (s *WikiService) ReadWikiPage(waypointDir, slug string) (*models.WikiPage, error) {
	wikiDir := filepath.Join(waypointDir, "wiki")
	return s.readPage(wikiDir, slug)
}

func (s *WikiService) ListDocsPages(projectPath, docsPath string) ([]*models.FileNode, error) {
	if docsPath == "" {
		return nil, ErrDocsNotEnabled
	}

	fullDocsPath := filepath.Join(projectPath, docsPath)
	if _, err := os.Stat(fullDocsPath); os.IsNotExist(err) {
		return nil, ErrWikiNotFound
	}

	return s.buildTree(fullDocsPath, "")
}

func (s *WikiService) ReadDocsPage(projectPath, docsPath, slug string) (*models.WikiPage, error) {
	if docsPath == "" {
		return nil, ErrDocsNotEnabled
	}

	fullDocsPath := filepath.Join(projectPath, docsPath)
	return s.readPage(fullDocsPath, slug)
}

func (s *WikiService) buildTree(rootDir, relativePath string) ([]*models.FileNode, error) {
	dirPath := filepath.Join(rootDir, relativePath)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var nodes []*models.FileNode
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		childRelPath := entry.Name()
		if relativePath != "" {
			childRelPath = relativePath + "/" + entry.Name()
		}

		node := &models.FileNode{
			Name:  entry.Name(),
			Path:  childRelPath,
			IsDir: entry.IsDir(),
		}

		if entry.IsDir() {
			children, err := s.buildTree(rootDir, childRelPath)
			if err != nil {
				return nil, err
			}
			node.Children = children
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		return nodes[i].Name < nodes[j].Name
	})

	return nodes, nil
}

func (s *WikiService) readPage(rootDir, slug string) (*models.WikiPage, error) {
	if err := validateSlug(slug); err != nil {
		return nil, err
	}

	filePath := filepath.Join(rootDir, slug)

	absRoot, _ := filepath.Abs(rootDir)
	absFile, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFile, absRoot) {
		return nil, ErrPathTraversal
	}

	content, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return nil, ErrWikiNotFound
	}
	if err != nil {
		return nil, err
	}

	title := strings.TrimSuffix(filepath.Base(slug), ".md")
	if title == "" {
		title = slug
	}

	return &models.WikiPage{
		Title:   title,
		Content: string(content),
		Path:    slug,
	}, nil
}

func validateSlug(slug string) error {
	if slug == "" {
		return errors.New("slug is required")
	}
	if strings.Contains(slug, "..") {
		return ErrPathTraversal
	}
	if filepath.IsAbs(slug) {
		return ErrPathTraversal
	}
	return nil
}
