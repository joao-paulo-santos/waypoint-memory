package services

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrWikiNotFound  = errors.New("wiki page not found")
	ErrPathTraversal = errors.New("invalid path: traversal not allowed")
)

type WikiService struct {
	DB *sql.DB
}

func NewWikiService(db *sql.DB) *WikiService {
	return &WikiService{DB: db}
}

func (s *WikiService) ListWikiPages(projectID int64) ([]*models.FileNode, error) {
	rows, err := s.DB.Query(
		`SELECT path FROM wiki_pages WHERE project_id = $1 ORDER BY path`, projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}

	return buildTreeFromPaths(paths), nil
}

func (s *WikiService) ReadWikiPage(projectID int64, pagePath string) (*models.WikiPage, error) {
	if err := validateWikiPath(pagePath); err != nil {
		return nil, err
	}

	var content string
	var createdAt, updatedAt string
	err := s.DB.QueryRow(
		`SELECT content, created_at, updated_at FROM wiki_pages WHERE project_id = $1 AND path = $2`,
		projectID, pagePath,
	).Scan(&content, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrWikiNotFound
	}
	if err != nil {
		return nil, err
	}

	title := strings.TrimSuffix(pagePath, ".md")
	if title == "" {
		title = pagePath
	}
	if strings.Contains(title, "/") {
		title = title[strings.LastIndex(title, "/")+1:]
	}

	return &models.WikiPage{
		Title:   title,
		Content: content,
		Path:    pagePath,
	}, nil
}

func (s *WikiService) WriteWikiPage(projectID int64, pagePath, content string) error {
	if err := validateWikiPath(pagePath); err != nil {
		return err
	}

	_, err := s.DB.Exec(
		`INSERT INTO wiki_pages (project_id, path, content, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (project_id, path) DO UPDATE SET content = $3, updated_at = NOW()`,
		projectID, pagePath, content,
	)
	return err
}

func (s *WikiService) DeleteWikiPage(projectID int64, pagePath string) error {
	if err := validateWikiPath(pagePath); err != nil {
		return err
	}

	result, err := s.DB.Exec(
		"DELETE FROM wiki_pages WHERE project_id = $1 AND path = $2",
		projectID, pagePath,
	)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrWikiNotFound
	}
	return nil
}

func validateWikiPath(pagePath string) error {
	if pagePath == "" {
		return errors.New("path is required")
	}
	if strings.Contains(pagePath, "..") {
		return ErrPathTraversal
	}
	return nil
}

func buildTreeFromPaths(paths []string) []*models.FileNode {
	root := &models.FileNode{Name: "wiki", IsDir: true}
	for _, p := range paths {
		parts := strings.Split(p, "/")
		current := root
		for i, part := range parts {
			isFile := i == len(parts)-1
			found := false
			for _, child := range current.Children {
				if child.Name == part {
					current = child
					found = true
					break
				}
			}
			if !found {
				node := &models.FileNode{
					Name:  part,
					Path:  strings.Join(parts[:i+1], "/"),
					IsDir: !isFile,
				}
				current.Children = append(current.Children, node)
				current = node
			}
		}
	}
	return root.Children
}
