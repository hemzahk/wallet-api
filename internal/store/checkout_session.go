package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/google/uuid"
)

type CheckoutSession struct {
	ID uuid.UUID `json:"id"`
	Token string `json:"token"`
	MerchantID uuid.UUID `json:"merchant_id"`
	Amount *big.Int `json:"amount"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type CheckoutSessionStore struct {
	db *sql.DB
}

func (s *CheckoutSessionStore) Create(ctx context.Context, session *CheckoutSession) error {
	query := `
		INSERT INTO checkout_sessions (
			id,
			token,
			merchant_id,
			amount,
			expires_at,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := s.db.ExecContext(
		ctx,
		query,
		session.ID,
		session.Token,
		session.MerchantID,
		session.Amount.Int64(),
		session.ExpiresAt,
		session.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *CheckoutSessionStore) GetByToken(ctx context.Context, token string) (*CheckoutSession, error) {
	query := `
		SELECT id, token, merchant_id, amount, expires_at, created_at
		FROM checkout_sessions
		WHERE token = $1 AND expires_at > $2
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	hash := sha256.Sum256([]byte(token))
	hashToken := hex.EncodeToString(hash[:])

	session := &CheckoutSession{}
	var rawAmount int64
	err := s.db.QueryRowContext(ctx, query, hashToken, time.Now()).Scan(
		&session.ID,
		&session.Token,
		&session.MerchantID,
		&rawAmount,
		&session.ExpiresAt,
		&session.CreatedAt, 
	)
	if err != nil {
		return nil, err
	}

	amountAsBigInt := big.NewInt(rawAmount)
	session.Amount = amountAsBigInt

	return session, nil
}