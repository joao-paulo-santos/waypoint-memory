package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type NullInt64 struct {
	sql.NullInt64
}

func (n NullInt64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("%d", n.Int64)), nil
}

func (n *NullInt64) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	var v int64
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Int64 = v
	n.Valid = true
	return nil
}

type Contact struct {
	ID                   int64    `json:"id"`
	FirstName            string   `json:"first_name"`
	LastName             string   `json:"last_name"`
	Email                string   `json:"email"`
	Phone                string   `json:"phone"`
	Company              string   `json:"company"`
	Role                 string   `json:"role"`
	Notes                string   `json:"notes"`
	Tags                 string   `json:"tags"`
	BirthdayDate         string   `json:"birthday_date"`
	BirthdayYear         NullInt64 `json:"birthday_year"`
	BirthdayRemindDays   int      `json:"birthday_remind_days"`
	BirthdayNotes        string   `json:"birthday_notes"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
}

type CreateContactRequest struct {
	FirstName          string `json:"first_name"`
	LastName           string `json:"last_name,omitempty"`
	Email              string `json:"email,omitempty"`
	Phone              string `json:"phone,omitempty"`
	Company            string `json:"company,omitempty"`
	Role               string `json:"role,omitempty"`
	Notes              string `json:"notes,omitempty"`
	Tags               string `json:"tags,omitempty"`
	BirthdayDate       string `json:"birthday_date,omitempty"`
	BirthdayYear       *int   `json:"birthday_year,omitempty"`
	BirthdayRemindDays int    `json:"birthday_remind_days,omitempty"`
	BirthdayNotes      string `json:"birthday_notes,omitempty"`
}

type UpdateContactRequest struct {
	FirstName          *string `json:"first_name,omitempty"`
	LastName           *string `json:"last_name,omitempty"`
	Email              *string `json:"email,omitempty"`
	Phone              *string `json:"phone,omitempty"`
	Company            *string `json:"company,omitempty"`
	Role               *string `json:"role,omitempty"`
	Notes              *string `json:"notes,omitempty"`
	Tags               *string `json:"tags,omitempty"`
	BirthdayDate       *string `json:"birthday_date,omitempty"`
	BirthdayYear       *int    `json:"birthday_year,omitempty"`
	BirthdayRemindDays *int    `json:"birthday_remind_days,omitempty"`
	BirthdayNotes      *string `json:"birthday_notes,omitempty"`
}

type UpcomingBirthday struct {
	ContactID    int64  `json:"contact_id"`
	Name         string `json:"name"`
	Date         string `json:"date"`
	OccurrenceDate string `json:"occurrence_date"`
	AgeTurning   *int   `json:"age_turning"`
	DaysUntil    int    `json:"days_until"`
}
