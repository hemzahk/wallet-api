package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrKeyNotFound = errors.New("idempotency key not found")
)

type IdempotencyKeys interface {
		Create(ctx context.Context, record IdempotencyKey)  error
		Get(ctx context.Context, key string, userID uuid.UUID) (*IdempotencyKey, error)
}

type IdempotencyKey struct {
	Key          string    `json:"key"`
	UserID       uuid.UUID `json:"user_id"`
	RequestHash  string    `json:"request_hash"`
	ResponseBody []byte    `json:"response_body"`
	StatusCode   int       `json:"status_code"`
	CreatedAt    string    `json:"created_at"`
}

type IdempotencyKeyStore struct {
	db *sql.DB
}

func (s *IdempotencyKeyStore) Create(ctx context.Context, record IdempotencyKey)  error {
	query := `
		INSERT INTO idempotency_keys (key, user_id, request_hash, response_body, status_code) 
		VALUES ($1,$2,$3,$4,$5)
	`
	ctx, cancel := context.WithTimeout(ctx, time.Second * 5)
	defer cancel()

	_, err := s.db.ExecContext(ctx, query, record.Key, record.UserID, record.RequestHash, record.ResponseBody, record.StatusCode)

	if err != nil {
		return err
	}
	return nil
}

func (s *IdempotencyKeyStore) Get(ctx context.Context, key string, userID uuid.UUID) (*IdempotencyKey, error) {
    query := `
        SELECT key, user_id, request_hash, response_body, status_code, created_at
        FROM idempotency_keys
        WHERE key = $1 AND user_id = $2
    `

    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    var ik IdempotencyKey
    err := s.db.QueryRowContext(ctx, query, key, userID).Scan(
        &ik.Key, &ik.UserID, &ik.RequestHash, &ik.ResponseBody, &ik.StatusCode, &ik.CreatedAt,
    )
    switch err{
		case sql.ErrNoRows:
			return nil, ErrKeyNotFound
    }

    return &ik, nil
}