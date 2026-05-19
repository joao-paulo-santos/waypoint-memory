package models

type Sprint struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	StartedAt *string `json:"started_at"`
	EndedAt   string  `json:"ended_at"`
	TaskCount int     `json:"task_count"`
	Summary   string  `json:"summary"`
}

type ArchivedTask struct {
	ID              int64   `json:"id"`
	SprintID        int64   `json:"sprint_id"`
	OriginalTaskID  int64   `json:"original_task_id"`
	BucketTitle     string  `json:"bucket_title"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Priority        int     `json:"priority"`
	CreatedBy       string  `json:"created_by"`
	DueDate         *string `json:"due_date"`
	Labels          string  `json:"labels"`
	Comments        string  `json:"comments"`
	DoneAt          *string `json:"done_at"`
	OriginalCreated string  `json:"original_created"`
	ArchivedAt      string  `json:"archived_at"`
}

type EndSprintRequest struct {
	SprintName string `json:"sprint_name,omitempty"`
	Summary    string `json:"summary,omitempty"`
}

type EndSprintResponse struct {
	Sprint        Sprint `json:"sprint"`
	TasksArchived int    `json:"tasks_archived"`
}

type SprintDetail struct {
	Sprint Sprint        `json:"sprint"`
	Tasks  []ArchivedTask `json:"tasks"`
}
