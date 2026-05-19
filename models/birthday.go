package models

type Birthday struct {
	ID               int64  `json:"id"`
	ContactID        *int64 `json:"contact_id"`
	Name             string `json:"name"`
	Date             string `json:"date"`
	Year             *int   `json:"year"`
	RemindDaysBefore int    `json:"remind_days_before"`
	Notes            string `json:"notes"`
	CreatedAt        string `json:"created_at"`
}

type CreateBirthdayRequest struct {
	ContactID        *int64 `json:"contact_id,omitempty"`
	Name             string `json:"name"`
	Date             string `json:"date"`
	Year             *int   `json:"year,omitempty"`
	RemindDaysBefore int    `json:"remind_days_before,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

type UpdateBirthdayRequest struct {
	Name             *string `json:"name,omitempty"`
	Date             *string `json:"date,omitempty"`
	Year             *int    `json:"year,omitempty"`
	RemindDaysBefore *int    `json:"remind_days_before,omitempty"`
	Notes            *string `json:"notes,omitempty"`
}

type UpcomingBirthday struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Date       string `json:"date"`
	AgeTurning *int   `json:"age_turning"`
	DaysUntil  int    `json:"days_until"`
	ContactID  *int64 `json:"contact_id"`
}
