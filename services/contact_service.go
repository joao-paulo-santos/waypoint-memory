package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/models"
)

var (
	ErrContactNotFound = errors.New("contact not found")
)

type ContactService struct {
	DB *sql.DB
}

func NewContactService(db *sql.DB) *ContactService {
	return &ContactService{DB: db}
}

const contactColumns = `id, first_name, last_name, email, phone, company, role, notes, tags,
	birthday_date, birthday_year, birthday_remind_days, birthday_notes,
	created_at, updated_at`

func scanContact(scanner interface{ Scan(...interface{}) error }) (*models.Contact, error) {
	c := &models.Contact{}
	err := scanner.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.Company, &c.Role, &c.Notes, &c.Tags,
		&c.BirthdayDate, &c.BirthdayYear, &c.BirthdayRemindDays, &c.BirthdayNotes,
		&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ContactService) List(ownerID int64) ([]models.Contact, error) {
	rows, err := s.DB.Query(
		`SELECT `+contactColumns+` FROM contacts WHERE owner_id = $1 ORDER BY first_name, last_name`, ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, err
		}
		contacts = append(contacts, *c)
	}
	return contacts, nil
}

func (s *ContactService) GetByID(id int64, ownerID int64) (*models.Contact, error) {
	c, err := scanContact(s.DB.QueryRow(
		`SELECT `+contactColumns+` FROM contacts WHERE id = $1 AND owner_id = $2`, id, ownerID,
	))
	if err == sql.ErrNoRows {
		return nil, ErrContactNotFound
	}
	return c, err
}

func (s *ContactService) Create(req models.CreateContactRequest, ownerID int64) (*models.Contact, error) {
	if req.FirstName == "" {
		return nil, errors.New("first_name is required")
	}

	tags := "[]"
	if req.Tags != "" {
		parts := strings.Split(req.Tags, ",")
		data, _ := json.Marshal(parts)
		tags = string(data)
	}

	var birthdayYear any
	if req.BirthdayYear != nil {
		birthdayYear = *req.BirthdayYear
	}

	remind := req.BirthdayRemindDays
	if remind == 0 {
		remind = 3
	}

	var id int64
	err := s.DB.QueryRow(
		`INSERT INTO contacts (owner_id, first_name, last_name, email, phone, company, role, notes, tags,
			birthday_date, birthday_year, birthday_remind_days, birthday_notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING id`,
		ownerID, req.FirstName, req.LastName, req.Email, req.Phone,
		req.Company, req.Role, req.Notes, tags,
		req.BirthdayDate, birthdayYear, remind, req.BirthdayNotes,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return s.GetByID(id, ownerID)
}

func (s *ContactService) Update(id int64, ownerID int64, req models.UpdateContactRequest) (*models.Contact, error) {
	if _, err := s.GetByID(id, ownerID); err != nil {
		return nil, err
	}

	sets := []string{"updated_at = NOW()"}
	args := []any{}
	paramIdx := 1

	if req.FirstName != nil {
		sets = append(sets, fmt.Sprintf("first_name = $%d", paramIdx))
		args = append(args, *req.FirstName)
		paramIdx++
	}
	if req.LastName != nil {
		sets = append(sets, fmt.Sprintf("last_name = $%d", paramIdx))
		args = append(args, *req.LastName)
		paramIdx++
	}
	if req.Email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", paramIdx))
		args = append(args, *req.Email)
		paramIdx++
	}
	if req.Phone != nil {
		sets = append(sets, fmt.Sprintf("phone = $%d", paramIdx))
		args = append(args, *req.Phone)
		paramIdx++
	}
	if req.Company != nil {
		sets = append(sets, fmt.Sprintf("company = $%d", paramIdx))
		args = append(args, *req.Company)
		paramIdx++
	}
	if req.Role != nil {
		sets = append(sets, fmt.Sprintf("role = $%d", paramIdx))
		args = append(args, *req.Role)
		paramIdx++
	}
	if req.Notes != nil {
		sets = append(sets, fmt.Sprintf("notes = $%d", paramIdx))
		args = append(args, *req.Notes)
		paramIdx++
	}
	if req.Tags != nil {
		parts := strings.Split(*req.Tags, ",")
		data, _ := json.Marshal(parts)
		sets = append(sets, fmt.Sprintf("tags = $%d", paramIdx))
		args = append(args, string(data))
		paramIdx++
	}
	if req.BirthdayDate != nil {
		sets = append(sets, fmt.Sprintf("birthday_date = $%d", paramIdx))
		args = append(args, *req.BirthdayDate)
		paramIdx++
	}
	if req.BirthdayYear != nil {
		sets = append(sets, fmt.Sprintf("birthday_year = $%d", paramIdx))
		args = append(args, *req.BirthdayYear)
		paramIdx++
	}
	if req.BirthdayRemindDays != nil {
		sets = append(sets, fmt.Sprintf("birthday_remind_days = $%d", paramIdx))
		args = append(args, *req.BirthdayRemindDays)
		paramIdx++
	}
	if req.BirthdayNotes != nil {
		sets = append(sets, fmt.Sprintf("birthday_notes = $%d", paramIdx))
		args = append(args, *req.BirthdayNotes)
		paramIdx++
	}

	if len(sets) == 1 {
		return s.GetByID(id, ownerID)
	}

	args = append(args, id, ownerID)
	query := "UPDATE contacts SET " + strings.Join(sets, ", ") + fmt.Sprintf(" WHERE id = $%d AND owner_id = $%d", paramIdx, paramIdx+1)
	if _, err := s.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return s.GetByID(id, ownerID)
}

func (s *ContactService) Delete(id int64, ownerID int64) error {
	result, err := s.DB.Exec("DELETE FROM contacts WHERE id = $1 AND owner_id = $2", id, ownerID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrContactNotFound
	}
	return nil
}

func (s *ContactService) GetUpcomingBirthdays(ownerID int64) ([]models.UpcomingBirthday, error) {
	contacts, err := s.List(ownerID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var results []models.UpcomingBirthday
	for _, c := range contacts {
		if c.BirthdayDate == "" {
			continue
		}
		parts := strings.SplitN(c.BirthdayDate, "-", 2)
		if len(parts) != 2 {
			continue
		}
		month, err1 := strconv.Atoi(parts[0])
		day, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			continue
		}

		occ := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if occ.Before(today) {
			occ = time.Date(now.Year()+1, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		}

		days := int(occ.Sub(today).Hours() / 24)
		name := strings.TrimSpace(c.FirstName + " " + c.LastName)
		var ageTurning *int
		if c.BirthdayYear.Valid {
			age := occ.Year() - int(c.BirthdayYear.Int64)
			ageTurning = &age
		}
		results = append(results, models.UpcomingBirthday{
			ContactID:     c.ID,
			Name:          name,
			Date:          c.BirthdayDate,
			OccurrenceDate: occ.Format("2006-01-02"),
			AgeTurning:    ageTurning,
			DaysUntil:     days,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DaysUntil < results[j].DaysUntil
	})

	if len(results) > 5 {
		results = results[:5]
	}

	return results, nil
}
