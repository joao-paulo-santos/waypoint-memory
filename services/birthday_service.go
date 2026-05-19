package services

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrBirthdayNotFound = errors.New("birthday not found")
	ErrBirthdayBadDate  = errors.New("date must be in MM-DD format")
)

type BirthdayService struct {
	DB *sql.DB
}

func NewBirthdayService(db *sql.DB) *BirthdayService {
	return &BirthdayService{DB: db}
}

func (s *BirthdayService) List() ([]models.Birthday, error) {
	rows, err := s.DB.Query(
		`SELECT id, contact_id, name, date, year, remind_days_before, notes, created_at
		 FROM birthdays ORDER BY date`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanBirthdays(rows)
}

func (s *BirthdayService) GetByID(id int64) (*models.Birthday, error) {
	b := &models.Birthday{}
	err := s.DB.QueryRow(
		`SELECT id, contact_id, name, date, year, remind_days_before, notes, created_at
		 FROM birthdays WHERE id = ?`, id,
	).Scan(&b.ID, &b.ContactID, &b.Name, &b.Date, &b.Year, &b.RemindDaysBefore, &b.Notes, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrBirthdayNotFound
	}
	return b, err
}

func (s *BirthdayService) Create(req models.CreateBirthdayRequest) (*models.Birthday, error) {
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if len(req.Date) != 5 || req.Date[2] != '-' {
		return nil, ErrBirthdayBadDate
	}

	remind := req.RemindDaysBefore
	if remind == 0 {
		remind = 3
	}

	var contactID any
	if req.ContactID != nil {
		contactID = *req.ContactID
	}

	result, err := s.DB.Exec(
		`INSERT INTO birthdays (contact_id, name, date, year, remind_days_before, notes)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		contactID, req.Name, req.Date, req.Year, remind, req.Notes,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return s.GetByID(id)
}

func (s *BirthdayService) Update(id int64, req models.UpdateBirthdayRequest) (*models.Birthday, error) {
	if _, err := s.GetByID(id); err != nil {
		return nil, err
	}

	sets := []string{}
	args := []any{}

	if req.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Date != nil {
		sets = append(sets, "date = ?")
		args = append(args, *req.Date)
	}
	if req.Year != nil {
		sets = append(sets, "year = ?")
		args = append(args, *req.Year)
	}
	if req.RemindDaysBefore != nil {
		sets = append(sets, "remind_days_before = ?")
		args = append(args, *req.RemindDaysBefore)
	}
	if req.Notes != nil {
		sets = append(sets, "notes = ?")
		args = append(args, *req.Notes)
	}

	if len(sets) == 0 {
		return s.GetByID(id)
	}

	args = append(args, id)
	query := "UPDATE birthdays SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := s.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *BirthdayService) Delete(id int64) error {
	result, err := s.DB.Exec("DELETE FROM birthdays WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrBirthdayNotFound
	}
	return nil
}

func (s *BirthdayService) GetUpcoming(daysAhead int) ([]models.UpcomingBirthday, error) {
	if daysAhead <= 0 {
		daysAhead = 30
	}

	now := time.Now()

	rows, err := s.DB.Query(
		`SELECT id, contact_id, name, date, year, remind_days_before, notes, created_at FROM birthdays`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.UpcomingBirthday
	for rows.Next() {
		var b models.Birthday
		if err := rows.Scan(&b.ID, &b.ContactID, &b.Name, &b.Date, &b.Year,
			&b.RemindDaysBefore, &b.Notes, &b.CreatedAt); err != nil {
			return nil, err
		}

		month, err := strconv.Atoi(b.Date[0:2])
		if err != nil {
			continue
		}
		day, err := strconv.Atoi(b.Date[3:5])
		if err != nil {
			continue
		}

		nextOccurrence := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if nextOccurrence.Before(now) {
			nextOccurrence = time.Date(now.Year()+1, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		}

		daysUntil := int(nextOccurrence.Sub(now).Hours() / 24)
		if daysUntil > daysAhead {
			continue
		}

		var ageTurning *int
		if b.Year != nil {
			age := nextOccurrence.Year() - *b.Year
			ageTurning = &age
		}

		results = append(results, models.UpcomingBirthday{
			ID:         b.ID,
			Name:       b.Name,
			Date:       b.Date,
			AgeTurning: ageTurning,
			DaysUntil:  daysUntil,
			ContactID:  b.ContactID,
		})
	}

	return results, nil
}

func scanBirthdays(rows *sql.Rows) ([]models.Birthday, error) {
	var birthdays []models.Birthday
	for rows.Next() {
		var b models.Birthday
		if err := rows.Scan(&b.ID, &b.ContactID, &b.Name, &b.Date, &b.Year,
			&b.RemindDaysBefore, &b.Notes, &b.CreatedAt); err != nil {
			return nil, err
		}
		birthdays = append(birthdays, b)
	}
	return birthdays, nil
}
