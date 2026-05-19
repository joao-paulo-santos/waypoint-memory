package models

type Project struct {
	ID          int64  `json:"id"`
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path"`
	DocsPath    string `json:"docs_path"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	IsArchived  bool   `json:"is_archived"`
	LastOpened  string `json:"last_opened"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

type InitProjectRequest struct {
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

type AddProjectRequest struct {
	Path  string `json:"path"`
	Name  string `json:"name,omitempty"`
	Color string `json:"color,omitempty"`
	Icon  string `json:"icon,omitempty"`
}

type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	DocsPath    *string `json:"docs_path,omitempty"`
	Color       *string `json:"color,omitempty"`
	Icon        *string `json:"icon,omitempty"`
	IsArchived  *bool   `json:"is_archived,omitempty"`
}

type BoardSummary struct {
	TotalActive int             `json:"total_active"`
	TotalDone   int             `json:"total_done"`
	Buckets     []BucketSummary `json:"buckets"`
}

type BucketSummary struct {
	Title     string `json:"title"`
	TaskCount int    `json:"task_count"`
}

type ProjectDetail struct {
	Project      Project      `json:"project"`
	BoardSummary BoardSummary `json:"board_summary"`
}
