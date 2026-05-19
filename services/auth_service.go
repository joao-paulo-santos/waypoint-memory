package services

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidPassword = errors.New("invalid password")

type AuthService struct {
	DB *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{DB: db}
}

func (s *AuthService) HasPassword() (bool, error) {
	var count int
	err := s.DB.QueryRow(
		"SELECT count(*) FROM app_config WHERE key = 'password_hash'",
	).Scan(&count)
	return count > 0, err
}

func (s *AuthService) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	_, err = s.DB.Exec(
		`INSERT INTO app_config (key, value) VALUES ('password_hash', ?)
		 ON CONFLICT(key) DO UPDATE SET value = ?`,
		string(hash), string(hash),
	)
	return err
}

func (s *AuthService) VerifyPassword(password string) error {
	var hash string
	err := s.DB.QueryRow(
		"SELECT value FROM app_config WHERE key = 'password_hash'",
	).Scan(&hash)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidPassword
	}
	return nil
}
