package services

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrProjectAlreadyRegistered = errors.New("project slug already exists")
	ErrProjectNotFound          = errors.New("project not found")
)

type ProjectService struct {
	DB *sql.DB
}

func NewProjectService(db *sql.DB) *ProjectService {
	return &ProjectService{DB: db}
}

func (s *ProjectService) Create(req models.CreateProjectRequest, ownerID int64) (*models.Project, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	slug := s.generateSlug(req.Name)
	slug = s.ensureUniqueSlug(slug)

	var id int64
	var isArchived bool
	var lastOpened sql.NullString
	var createdAt, updatedAt string
	err := s.DB.QueryRow(
		`INSERT INTO projects (slug, name, description, color, icon, owner_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, is_archived, last_opened, created_at, updated_at`,
		slug, req.Name, req.Description, req.Color, req.Icon, ownerID,
	).Scan(&id, &isArchived, &lastOpened, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}

	if err := db.EnsureDefaultBuckets(s.DB, id); err != nil {
		return nil, fmt.Errorf("create default buckets: %w", err)
	}

	if err := db.EnsureWikiIndex(s.DB, id); err != nil {
		return nil, fmt.Errorf("create wiki index: %w", err)
	}

	return &models.Project{
		ID:          id,
		Slug:        slug,
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		OwnerID:     ownerID,
		IsArchived:  isArchived,
		LastOpened:  lastOpened.String,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}

func (s *ProjectService) GetByID(id int64) (*models.Project, error) {
	p := &models.Project{}
	err := s.DB.QueryRow(
		`SELECT id, slug, name, description, color, icon, owner_id,
		        is_archived, COALESCE(last_opened::text, ''), created_at, updated_at
		 FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.Color, &p.Icon, &p.OwnerID,
		&p.IsArchived, &p.LastOpened, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrProjectNotFound
	}
	return p, err
}

func (s *ProjectService) GetBySlug(slug string) (*models.Project, error) {
	p := &models.Project{}
	err := s.DB.QueryRow(
		`SELECT id, slug, name, description, color, icon, owner_id,
		        is_archived, COALESCE(last_opened::text, ''), created_at, updated_at
		 FROM projects WHERE slug = $1`, slug,
	).Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.Color, &p.Icon, &p.OwnerID,
		&p.IsArchived, &p.LastOpened, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrProjectNotFound
	}
	return p, err
}

func (s *ProjectService) List(ownerID int64) ([]models.Project, error) {
	rows, err := s.DB.Query(
		`SELECT id, slug, name, description, color, icon, owner_id,
		        is_archived, COALESCE(last_opened::text, ''), created_at, updated_at
		 FROM projects WHERE owner_id = $1 ORDER BY name`, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanProjects(rows)
}

func (s *ProjectService) Update(id int64, req models.UpdateProjectRequest) (*models.Project, error) {
	if _, err := s.GetByID(id); err != nil {
		return nil, err
	}

	sets := []string{}
	args := []any{}
	paramIdx := 1

	if req.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", paramIdx))
		args = append(args, *req.Name)
		paramIdx++
	}
	if req.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", paramIdx))
		args = append(args, *req.Description)
		paramIdx++
	}
	if req.Color != nil {
		sets = append(sets, fmt.Sprintf("color = $%d", paramIdx))
		args = append(args, *req.Color)
		paramIdx++
	}
	if req.Icon != nil {
		sets = append(sets, fmt.Sprintf("icon = $%d", paramIdx))
		args = append(args, *req.Icon)
		paramIdx++
	}
	if req.IsArchived != nil {
		sets = append(sets, fmt.Sprintf("is_archived = $%d", paramIdx))
		args = append(args, *req.IsArchived)
		paramIdx++
	}

	if len(sets) == 0 {
		return s.GetByID(id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := "UPDATE projects SET " + strings.Join(sets, ", ") + fmt.Sprintf(" WHERE id = $%d", paramIdx)
	if _, err := s.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *ProjectService) Delete(id int64) error {
	if _, err := s.GetByID(id); err != nil {
		return err
	}

	result, err := s.DB.Exec("DELETE FROM projects WHERE id = $1", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrProjectNotFound
	}
	return nil
}

func (s *ProjectService) GetProjectDetail(db *sql.DB, id int64) (*models.ProjectDetail, error) {
	p, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}

	summary := models.BoardSummary{}
	rows, err := db.Query(
		`SELECT b.title, count(t.id)
		 FROM buckets b
		 LEFT JOIN tasks t ON t.bucket_id = b.id AND t.project_id = $1 AND t.sprint_id IS NULL
		 WHERE b.project_id = $1
		 GROUP BY b.id
		 ORDER BY b.position`, id,
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

func (s *ProjectService) generateSlug(name string) string {
	slug := strings.ToLower(name)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug = re.ReplaceAllString(slug, "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func (s *ProjectService) ensureUniqueSlug(slug string) string {
	if s.isSlugAvailable(slug) {
		return slug
	}

	suffix := 2
	for {
		candidate := fmt.Sprintf("%s-%d", slug, suffix)
		if s.isSlugAvailable(candidate) {
			return candidate
		}
		suffix++
	}
}

func (s *ProjectService) isSlugAvailable(slug string) bool {
	var count int
	err := s.DB.QueryRow("SELECT count(*) FROM projects WHERE slug = $1", slug).Scan(&count)
	if err != nil {
		return true
	}
	return count == 0
}

func scanProjects(rows *sql.Rows) ([]models.Project, error) {
	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.Color, &p.Icon,
			&p.OwnerID, &p.IsArchived, &p.LastOpened, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}
