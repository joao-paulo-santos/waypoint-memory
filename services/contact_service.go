package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

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

func (s *ContactService) List() ([]models.Contact, error) {
	rows, err := s.DB.Query(
		`SELECT id, first_name, last_name, email, phone, company, role, notes, tags,
		        created_at, updated_at
		 FROM contacts ORDER BY first_name, last_name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var c models.Contact
		if err := rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
			&c.Company, &c.Role, &c.Notes, &c.Tags, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func (s *ContactService) GetByID(id int64) (*models.Contact, error) {
	c := &models.Contact{}
	err := s.DB.QueryRow(
		`SELECT id, first_name, last_name, email, phone, company, role, notes, tags,
		        created_at, updated_at
		 FROM contacts WHERE id = ?`, id,
	).Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.Company, &c.Role, &c.Notes, &c.Tags, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrContactNotFound
	}
	return c, err
}

func (s *ContactService) Create(req models.CreateContactRequest) (*models.Contact, error) {
	if req.FirstName == "" {
		return nil, errors.New("first_name is required")
	}

	tags := "[]"
	if req.Tags != "" {
		parts := strings.Split(req.Tags, ",")
		data, _ := json.Marshal(parts)
		tags = string(data)
	}

	result, err := s.DB.Exec(
		`INSERT INTO contacts (first_name, last_name, email, phone, company, role, notes, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.FirstName, req.LastName, req.Email, req.Phone,
		req.Company, req.Role, req.Notes, tags,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()
	return s.GetByID(id)
}

func (s *ContactService) Update(id int64, req models.UpdateContactRequest) (*models.Contact, error) {
	if _, err := s.GetByID(id); err != nil {
		return nil, err
	}

	sets := []string{"updated_at = datetime('now')"}
	args := []any{}

	if req.FirstName != nil {
		sets = append(sets, "first_name = ?")
		args = append(args, *req.FirstName)
	}
	if req.LastName != nil {
		sets = append(sets, "last_name = ?")
		args = append(args, *req.LastName)
	}
	if req.Email != nil {
		sets = append(sets, "email = ?")
		args = append(args, *req.Email)
	}
	if req.Phone != nil {
		sets = append(sets, "phone = ?")
		args = append(args, *req.Phone)
	}
	if req.Company != nil {
		sets = append(sets, "company = ?")
		args = append(args, *req.Company)
	}
	if req.Role != nil {
		sets = append(sets, "role = ?")
		args = append(args, *req.Role)
	}
	if req.Notes != nil {
		sets = append(sets, "notes = ?")
		args = append(args, *req.Notes)
	}
	if req.Tags != nil {
		parts := strings.Split(*req.Tags, ",")
		data, _ := json.Marshal(parts)
		sets = append(sets, "tags = ?")
		args = append(args, string(data))
	}

	if len(sets) == 1 {
		return s.GetByID(id)
	}

	args = append(args, id)
	query := "UPDATE contacts SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := s.DB.Exec(query, args...); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *ContactService) Delete(id int64) error {
	result, err := s.DB.Exec("DELETE FROM contacts WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrContactNotFound
	}
	return nil
}
