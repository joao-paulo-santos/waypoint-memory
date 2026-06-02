package models

type Task struct {
	ID          int64    `json:"id"`
	ProjectID   int64    `json:"project_id"`
	BucketID    int64    `json:"bucket_id"`
	SprintID    *int64   `json:"sprint_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Position    float64  `json:"position"`
	Priority    int      `json:"priority"`
	DueDate     *string  `json:"due_date"`
	Done        bool     `json:"done"`
	DoneAt      *string  `json:"done_at"`
	CreatedBy   string   `json:"created_by"`
	Labels      []Label  `json:"labels,omitempty"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
	DescriptionPreview string  `json:"description_preview,omitempty"`
	LabelNames         string  `json:"label_names,omitempty"`
	BucketTitle        string  `json:"bucket_title,omitempty"`
}

type CreateTaskRequest struct {
	BucketID    int64   `json:"bucket_id"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Priority    int     `json:"priority,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	CreatedBy   string  `json:"created_by,omitempty"`
	Labels      string  `json:"labels,omitempty"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	BucketID    *int64  `json:"bucket_id,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Done        *bool   `json:"done,omitempty"`
	Labels      *string `json:"labels,omitempty"`
}

type MoveTaskRequest struct {
	BucketID int64 `json:"bucket_id"`
}

type ReorderTasksRequest struct {
	TaskIDs []int64 `json:"task_ids"`
}
