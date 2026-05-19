package models

type CalendarEvent struct {
	Type        string `json:"type"`
	Date        string `json:"date"`
	Title       string `json:"title"`
	ProjectID   *int64 `json:"project_id,omitempty"`
	ProjectName string `json:"project,omitempty"`
	TaskID      *int64 `json:"task_id,omitempty"`
	Name        string `json:"name,omitempty"`
	AgeTurning  *int   `json:"age_turning,omitempty"`
	DaysUntil   *int   `json:"days_until,omitempty"`
	HadDueDate  *bool  `json:"had_due_date,omitempty"`
	Recurrence  string `json:"recurrence,omitempty"`
	Category    string `json:"category,omitempty"`
}

type CalendarResponse struct {
	Events []CalendarEvent `json:"events"`
	From   string          `json:"from"`
	To     string          `json:"to"`
}
