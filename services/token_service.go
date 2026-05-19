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

func (s *TokenService) Create(name string, permissions string) (*CreateTokenResult, error) {
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

	result, err := s.DB.Exec(
		`INSERT INTO api_tokens (name, token_hash, prefix, permissions)
		 VALUES (?, ?, ?, ?)`,
		name, hashHex, prefix, permissions,
	)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	return &CreateTokenResult{
		Token: token,
		TokenInfo: TokenInfo{
			ID:          id,
			Name:        name,
			Prefix:      prefix,
			Permissions: permissions,
		},
	}, nil
}

func (s *TokenService) List() ([]TokenInfo, error) {
	rows, err := s.DB.Query(
		`SELECT id, name, prefix, permissions, COALESCE(last_used, ''), created_at
		 FROM api_tokens ORDER BY created_at DESC`,
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

func (s *TokenService) Delete(id int64) error {
	result, err := s.DB.Exec("DELETE FROM api_tokens WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrTokenNotFound
	}
	return nil
}
