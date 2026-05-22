package models

type CalendarEvent struct {
	Type        string `json:"type"`
	Date        string `json:"date"`
	Title       string `json:"title"`
	ProjectID   *int64 `json:"project_id,omitempty"`
	ProjectName string `json:"project,omitempty"`
	TaskID      *int64 `json:"task_id,omitempty"`
	EventID     *int64 `json:"event_id,omitempty"`
	Name        string `json:"name,omitempty"`
	AgeTurning  *int   `json:"age_turning,omitempty"`
	DaysUntil   *int   `json:"days_until,omitempty"`
	Done        bool   `json:"done,omitempty"`
	Category    string `json:"category,omitempty"`
}

type UpcomingItem struct {
	Type        string `json:"type"`
	Date        string `json:"date"`
	Title       string `json:"title"`
	Subtitle    string `json:"subtitle,omitempty"`
	ProjectID   *int64 `json:"project_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	TaskID      *int64 `json:"task_id,omitempty"`
	ContactID   *int64 `json:"contact_id,omitempty"`
	AgeTurning  *int   `json:"age_turning,omitempty"`
	DaysUntil   int    `json:"days_until"`
	Done        bool   `json:"done,omitempty"`
	Category    string `json:"category,omitempty"`
}

type CalendarResponse struct {
	Events []CalendarEvent `json:"events"`
	From   string          `json:"from"`
	To     string          `json:"to"`
}
