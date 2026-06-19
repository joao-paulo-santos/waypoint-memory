package models

type Project struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
	OwnerID     int64  `json:"owner_id"`
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

type UpdateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
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
