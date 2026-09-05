package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hemzahk/wallet-api/internal/dbtx"
)

type RefundRequests interface {
	Create(ctx context.Context, refundRequest *RefundRequest) error
}

type RefundRequest struct {
	ID uuid.UUID `json:"id"`
	TransactionRef string `json:"transaction_ref"`
	Status string `json:"status"`
	Description string `json:"description"`
	CreatedAt time.Time `json:"created_at"`
}

type RefundRequestStore struct {
	db *sql.DB
}

func (s *RefundRequestStore) Create(ctx context.Context, refundRequest *RefundRequest) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		INSERT INTO refund_requests (id, transaction_ref, status, description, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err := dbtx.ExecContext(ctx,
							   query,
							   refundRequest.ID,
							   refundRequest.TransactionRef,
							   refundRequest.Status,
							   refundRequest.Description,
							   refundRequest.CreatedAt,
							)
	if err != nil {
		return err
	}
	
	return nil
}