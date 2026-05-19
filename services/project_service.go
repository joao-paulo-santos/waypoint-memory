package services

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"gopkg.in/yaml.v3"
)

var (
	ErrProjectAlreadyRegistered = errors.New("project already registered at this path")
	ErrProjectPathNotFound      = errors.New("path does not exist")
	ErrProjectPathNotDir        = errors.New("path is not a directory")
	ErrProjectPathNotAbsolute   = errors.New("path must be absolute")
	ErrProjectNested            = errors.New("path is parent/child of existing project")
	ErrProjectNotFound          = errors.New("project not found")
	ErrWaypointNotFound         = errors.New("no .waypoint/ found at path")
)

type ProjectService struct {
	CentralDB   *sql.DB
	ProjectsDir string
}

func NewProjectService(centralDB *sql.DB, projectsDir string) *ProjectService {
	return &ProjectService{CentralDB: centralDB, ProjectsDir: projectsDir}
}

func (s *ProjectService) Create(req models.CreateProjectRequest) (*models.Project, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	projectPath := filepath.Join(s.ProjectsDir, req.Name)

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return nil, fmt.Errorf("create project directory: %w", err)
	}

	uuid, err := db.InitProjectDir(projectPath, req.Name, req.Description)
	if err != nil {
		os.RemoveAll(projectPath)
		return nil, fmt.Errorf("init project dir: %w", err)
	}

	return s.registerProject(uuid, req.Name, req.Description, projectPath, req.Color, req.Icon)
}

func (s *ProjectService) Init(req models.InitProjectRequest) (*models.Project, error) {
	if err := validatePath(req.Path); err != nil {
		return nil, err
	}

	if err := s.checkNesting(req.Path); err != nil {
		return nil, err
	}

	name := req.Name
	if name == "" {
		name = filepath.Base(req.Path)
	}

	uuid, err := db.InitProjectDir(req.Path, name, req.Description)
	if err != nil {
		return nil, fmt.Errorf("init project dir: %w", err)
	}

	return s.registerProject(uuid, name, req.Description, req.Path, req.Color, req.Icon)
}

func (s *ProjectService) Add(req models.AddProjectRequest) (*models.Project, error) {
	if err := validatePath(req.Path); err != nil {
		return nil, err
	}

	if err := s.checkNesting(req.Path); err != nil {
		return nil, err
	}

	waypointDir := filepath.Join(req.Path, ".waypoint")
	if _, err := os.Stat(waypointDir); os.IsNotExist(err) {
		return nil, ErrWaypointNotFound
	}

	cfgData, err := os.ReadFile(filepath.Join(waypointDir, "config.yaml"))
	if err != nil {
		return nil, fmt.Errorf("read config.yaml: %w", err)
	}

	var cfg db.ProjectConfig
	if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.yaml: %w", err)
	}

	name := req.Name
	if name == "" {
		name = cfg.Project.Name
	}
	if name == "" {
		name = filepath.Base(req.Path)
	}

	return s.registerProject(cfg.Project.UUID, name, cfg.Project.Description, req.Path, req.Color, req.Icon)
}

func (s *ProjectService) registerProject(uuid, name, description, path, color, icon string) (*models.Project, error) {
	var existingID int64
	err := s.CentralDB.QueryRow("SELECT id FROM projects WHERE path = ?", path).Scan(&existingID)
	if err == nil {
		return nil, ErrProjectAlreadyRegistered
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("check existing: %w", err)
	}

	err = s.CentralDB.QueryRow("SELECT id FROM projects WHERE uuid = ?", uuid).Scan(&existingID)
	if err == nil {
		_, err := s.CentralDB.Exec(
			"UPDATE projects SET path = ?, name = ?, updated_at = datetime('now') WHERE uuid = ?",
			path, name, uuid,
		)
		if err != nil {
			return nil, fmt.Errorf("update project path: %w", err)
		}
		return s.GetByID(existingID)
	}

	result, err := s.CentralDB.Exec(
		`INSERT INTO projects (uuid, name, description, path, color, icon)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		uuid, name, description, path, color, icon,
	)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}

	id, _ := result.LastInsertId()
	return s.GetByID(id)
}

func (s *ProjectService) GetByID(id int64) (*models.Project, error) {
	p := &models.Project{}
	err := s.CentralDB.QueryRow(
		`SELECT id, uuid, name, description, path, docs_path, color, icon,
		        is_archived, last_opened, created_at, updated_at
		 FROM projects WHERE id = ?`, id,
	).Scan(&p.ID, &p.UUID, &p.Name, &p.Description, &p.Path, &p.DocsPath,
		&p.Color, &p.Icon, &p.IsArchived, &p.LastOpened, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrProjectNotFound
	}
	return p, err
}

func (s *ProjectService) List() ([]models.Project, error) {
	rows, err := s.CentralDB.Query(
		`SELECT id, uuid, name, description, path, docs_path, color, icon,
		        is_archived, last_opened, created_at, updated_at
		 FROM projects ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UUID, &p.Name, &p.Description, &p.Path,
			&p.DocsPath, &p.Color, &p.Icon, &p.IsArchived, &p.LastOpened,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *ProjectService) Update(id int64, req models.UpdateProjectRequest) (*models.Project, error) {
	if _, err := s.GetByID(id); err != nil {
		return nil, err
	}

	sets := []string{}
	args := []any{}

	if req.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *req.Description)
	}
	if req.DocsPath != nil {
		sets = append(sets, "docs_path = ?")
		args = append(args, *req.DocsPath)
	}
	if req.Color != nil {
		sets = append(sets, "color = ?")
		args = append(args, *req.Color)
	}
	if req.Icon != nil {
		sets = append(sets, "icon = ?")
		args = append(args, *req.Icon)
	}
	if req.IsArchived != nil {
		sets = append(sets, "is_archived = ?")
		args = append(args, *req.IsArchived)
	}

	if len(sets) == 0 {
		return s.GetByID(id)
	}

	sets = append(sets, "updated_at = datetime('now')")
	args = append(args, id)

	query := "UPDATE projects SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := s.CentralDB.Exec(query, args...); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *ProjectService) Delete(id int64) error {
	result, err := s.CentralDB.Exec("DELETE FROM projects WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *ProjectService) GetProjectDB(id int64) (*sql.DB, error) {
	p, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(p.Path, ".waypoint", "project.db")
	return sql.Open("sqlite", dbPath)
}

func (s *ProjectService) GetProjectDetail(id int64) (*models.ProjectDetail, error) {
	p, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	projDB, err := s.GetProjectDB(id)
	if err != nil {
		return nil, err
	}
	defer projDB.Close()

	summary := models.BoardSummary{}
	rows, err := projDB.Query(
		`SELECT b.title, count(t.id)
		 FROM buckets b
		 LEFT JOIN tasks t ON t.bucket_id = b.id
		 GROUP BY b.id
		 ORDER BY b.position`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var bs models.BucketSummary
		if err := rows.Scan(&bs.Title, &bs.TaskCount); err != nil {
			return nil, err
		}
		summary.Buckets = append(summary.Buckets, bs)
		if bs.Title == "Done" {
			summary.TotalDone = bs.TaskCount
		} else {
			summary.TotalActive += bs.TaskCount
		}
	}

	return &models.ProjectDetail{Project: *p, BoardSummary: summary}, nil
}

func validatePath(path string) error {
	if !filepath.IsAbs(path) {
		return ErrProjectPathNotAbsolute
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return ErrProjectPathNotFound
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return ErrProjectPathNotDir
	}
	return nil
}

func (s *ProjectService) checkNesting(path string) error {
	rows, err := s.CentralDB.Query("SELECT path FROM projects")
	if err != nil {
		return err
	}
	defer rows.Close()

	cleanPath := filepath.Clean(path)
	for rows.Next() {
		var existing string
		if err := rows.Scan(&existing); err != nil {
			return err
		}
		cleanExisting := filepath.Clean(existing)

		if strings.HasPrefix(cleanPath, cleanExisting+string(os.PathSeparator)) {
			return fmt.Errorf("%w: %q is inside %q", ErrProjectNested, cleanPath, cleanExisting)
		}
		if strings.HasPrefix(cleanExisting, cleanPath+string(os.PathSeparator)) {
			return fmt.Errorf("%w: %q is parent of %q", ErrProjectNested, cleanPath, cleanExisting)
		}
	}
	return nil
}
