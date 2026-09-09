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
	ClaimPending(ctx context.Context) (*RefundRequest, error)
	MarkCompleted(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, description string) error
}

type RefundRequest struct {
	ID             uuid.UUID `json:"id"`
	TransactionRef string    `json:"transaction_ref"`
	Status         string    `json:"status"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
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

func (s *RefundRequestStore) ClaimPending(ctx context.Context) (*RefundRequest, error) {
	dbtx := dbtx.ExtractTx(ctx, s.db)
	query := `
		UPDATE refund_requests
		SET status = 'processing'
		WHERE id = (
			SELECT id
			FROM refund_requests
			WHERE status = 'pending'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, transaction_ref, status, description, created_at
	`

	refundRequest := &RefundRequest{}
	err := dbtx.QueryRowContext(ctx, query).Scan(
		&refundRequest.ID,
		&refundRequest.TransactionRef,
		&refundRequest.Status,
		&refundRequest.Description,
		&refundRequest.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return refundRequest, nil
}

func (s *RefundRequestStore) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	return s.updateStatus(ctx, id, "completed", "")
}

func (s *RefundRequestStore) MarkFailed(ctx context.Context, id uuid.UUID, description string) error {
	return s.updateStatus(ctx, id, "failed", description)
}

func (s *RefundRequestStore) updateStatus(ctx context.Context, id uuid.UUID, status string, description string) error {
	dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		UPDATE refund_requests
		SET status = $1, description = CASE WHEN $2 = '' THEN description ELSE $2 END
		WHERE id = $3
	`
	_, err := dbtx.ExecContext(ctx, query, status, description, id)
	if err != nil {
		return err
	}
	
	return nil
}
