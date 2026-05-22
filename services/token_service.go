package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
)

var ErrTokenNotFound = errors.New("token not found")

type TokenService struct {
	DB *sql.DB
}

type TokenInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Prefix      string `json:"prefix"`
	Permissions string `json:"permissions"`
	LastUsed    string `json:"last_used"`
	CreatedAt   string `json:"created_at"`
}

type CreateTokenResult struct {
	Token     string    `json:"token"`
	TokenInfo TokenInfo `json:"info"`
}

func NewTokenService(db *sql.DB) *TokenService {
	return &TokenService{DB: db}
}

func (s *TokenService) Create(name string, userID int64, permissions string) (*CreateTokenResult, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	tokenBytes := make([]byte, 32)
	rand.Read(tokenBytes)
	token := "wpt_" + hex.EncodeToString(tokenBytes)

	hash := sha256.Sum256([]byte(token))
	hashHex := hex.EncodeToString(hash[:])
	prefix := token[:8]

	if permissions == "" {
		permissions = `["read","write"]`
	}

	var id int64
	var createdAt string
	err := s.DB.QueryRow(
		`INSERT INTO api_tokens (name, token_hash, prefix, permissions, user_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at`,
		name, hashHex, prefix, permissions, userID,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, err
	}

	return &CreateTokenResult{
		Token: token,
		TokenInfo: TokenInfo{
			ID:          id,
			Name:        name,
			Prefix:      prefix,
			Permissions: permissions,
			CreatedAt:   createdAt,
		},
	}, nil
}

func (s *TokenService) List(userID int64) ([]TokenInfo, error) {
	rows, err := s.DB.Query(
		`SELECT id, name, prefix, permissions, COALESCE(last_used::text, ''), created_at
		 FROM api_tokens WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []TokenInfo
	for rows.Next() {
		var t TokenInfo
		if err := rows.Scan(&t.ID, &t.Name, &t.Prefix, &t.Permissions, &t.LastUsed, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

func (s *TokenService) Delete(id int64, userID int64) error {
	result, err := s.DB.Exec("DELETE FROM api_tokens WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrTokenNotFound
	}
	return nil
}
