package services

import (
	"database/sql"
	"errors"
	"time"

	"github.com/joao-paulo-santos/waypoint-memory/db"
	"github.com/joao-paulo-santos/waypoint-memory/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
)

type AuthService struct {
	DB *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{DB: db}
}

func (s *AuthService) Register(username, password string) (*models.User, error) {
	if username == "" || password == "" {
		return nil, errors.New("username and password are required")
	}

	var existing int
	err := s.DB.QueryRow("SELECT count(*) FROM users WHERE username = $1", username).Scan(&existing)
	if err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, err
	}

	var id int64
	var createdAt string
	err = s.DB.QueryRow(
		`INSERT INTO users (username, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, created_at`,
		username, string(hash),
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:        id,
		Username:  username,
		CreatedAt: createdAt,
	}, nil
}

func (s *AuthService) Login(username, password string) (string, *models.User, error) {
	var id int64
	var hash string
	var createdAt string
	err := s.DB.QueryRow(
		"SELECT id, password_hash, created_at FROM users WHERE username = $1",
		username,
	).Scan(&id, &hash, &createdAt)
	if err == sql.ErrNoRows {
		return "", nil, ErrInvalidCredentials
	}
	if err != nil {
		return "", nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token := db.GenerateToken(32)
	expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

	_, err = s.DB.Exec(
		"INSERT INTO sessions (user_id, token, expires_at) VALUES ($1, $2, $3)",
		id, token, expiresAt,
	)
	if err != nil {
		return "", nil, err
	}

	user := &models.User{
		ID:        id,
		Username:  username,
		CreatedAt: createdAt,
	}

	return token, user, nil
}

func (s *AuthService) Logout(token string) error {
	result, err := s.DB.Exec("DELETE FROM sessions WHERE token = $1", token)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *AuthService) GetSessionUser(token string) (*models.User, error) {
	var u models.User
	var expiresAt time.Time
	err := s.DB.QueryRow(
		`SELECT u.id, u.username, u.created_at, s.expires_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token = $1`,
		token,
	).Scan(&u.ID, &u.Username, &u.CreatedAt, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}

	if time.Now().UTC().After(expiresAt) {
		s.DB.Exec("DELETE FROM sessions WHERE token = $1", token)
		return nil, ErrSessionExpired
	}

	return &u, nil
}

func (s *AuthService) HasUsers() (bool, error) {
	var count int
	err := s.DB.QueryRow("SELECT count(*) FROM users").Scan(&count)
	return count > 0, err
}
