package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
)

type Merchants interface {
	Create(ctx context.Context, merchant *Merchant) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*Merchant, error)
	Delete(ctx context.Context, merchantID uuid.UUID) error
}

type Merchant struct {
	ID uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
	BusinessName string `json:"business_name"`
	FeeRate string `json:"fee_rate"`
	CashbackRate string `json:"cashback_rate"`
	CreatedAt time.Time `json:"created_at"`
}

type MerchantStore struct {
	db *sql.DB
}

func (s *MerchantStore) Create(ctx context.Context, merchant *Merchant) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		INSERT INTO merchants (id, user_id, business_name, fee_rate, cashback_rate, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(
		ctx,
		query,
		merchant.ID, 
		merchant.UserID, 
		merchant.BusinessName, 
		merchant.FeeRate,
		merchant.CashbackRate,
		merchant.CreatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *MerchantStore) GetByUserID(ctx context.Context, userID uuid.UUID) (*Merchant, error) {
	query := `
		SELECT id, user_id, business_name, fee_rate, cashback_rate, created_at
		FROM merchants
		WHERE user_id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	merchant := &Merchant{}
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&merchant.ID,
		&merchant.UserID,
		&merchant.BusinessName,
		&merchant.FeeRate,
		&merchant.CashbackRate,
		&merchant.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return merchant, nil
}

func (s *MerchantStore) Delete(ctx context.Context, merchantID uuid.UUID) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		DELETE FROM merchants WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx, query, merchantID)
	if err != nil {
		return err
	}

	return nil
}