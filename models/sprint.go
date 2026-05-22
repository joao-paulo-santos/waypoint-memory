package models

type Sprint struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	StartedAt *string `json:"started_at"`
	EndedAt   string  `json:"ended_at"`
	TaskCount int     `json:"task_count"`
	Summary   string  `json:"summary"`
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
	Sprint Sprint `json:"sprint"`
	Tasks  []Task `json:"tasks"`
}
