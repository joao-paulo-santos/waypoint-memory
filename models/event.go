package models

type Event struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Date             string  `json:"date"`
	Time             *string `json:"time,omitempty"`
	Recurrence       *string `json:"recurrence,omitempty"`
	Category         string  `json:"category"`
	RemindDaysBefore int     `json:"remind_days_before"`
	CreatedAt        string  `json:"created_at"`
}

type CreateEventRequest struct {
	Title            string  `json:"title"`
	Description      string  `json:"description,omitempty"`
	Date             string  `json:"date"`
	Time             *string `json:"time,omitempty"`
	Recurrence       *string `json:"recurrence,omitempty"`
	Category         string  `json:"category,omitempty"`
	RemindDaysBefore int     `json:"remind_days_before,omitempty"`
}

type UpdateEventRequest struct {
	Title            *string `json:"title,omitempty"`
	Description      *string `json:"description,omitempty"`
	Date             *string `json:"date,omitempty"`
	Time             **string `json:"time,omitempty"`
	Recurrence       **string `json:"recurrence,omitempty"`
	Category         *string `json:"category,omitempty"`
	RemindDaysBefore *int    `json:"remind_days_before,omitempty"`
}
