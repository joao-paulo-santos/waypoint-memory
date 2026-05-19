package models

type RecurringEvent struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date"`
	Date             string  `json:"date"`
	Year             *int    `json:"year"`
	Recurrence       string  `json:"recurrence"`
	Category         string  `json:"category"`
	RemindDaysBefore int     `json:"remind_days_before"`
	CreatedAt        string  `json:"created_at"`
}

type CreateRecurringEventRequest struct {
	Title            string  `json:"title"`
	Description      string  `json:"description,omitempty"`
	StartDate        string  `json:"start_date"`
	EndDate          *string `json:"end_date,omitempty"`
	Date             string  `json:"date"`
	Year             *int    `json:"year,omitempty"`
	Recurrence       string  `json:"recurrence,omitempty"`
	Category         string  `json:"category,omitempty"`
	RemindDaysBefore int     `json:"remind_days_before,omitempty"`
}

type UpdateRecurringEventRequest struct {
	Title            *string `json:"title,omitempty"`
	Description      *string `json:"description,omitempty"`
	StartDate        *string `json:"start_date,omitempty"`
	EndDate          *string `json:"end_date,omitempty"`
	Date             *string `json:"date,omitempty"`
	Year             *int    `json:"year,omitempty"`
	Recurrence       *string `json:"recurrence,omitempty"`
	Category         *string `json:"category,omitempty"`
	RemindDaysBefore *int    `json:"remind_days_before,omitempty"`
}

type UpcomingEvent struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Date       string `json:"date"`
	Recurrence string `json:"recurrence"`
	Category   string `json:"category"`
	DaysUntil  int    `json:"days_until"`
}
