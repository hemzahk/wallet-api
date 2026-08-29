package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hemzahk/wallet-api/internal/dbtx"
)

type WebhookEventID struct {
	EventID  string `json:"event_id"`
	SourceID string `json:"source_id"` // for our PSP 645b44fb-314f-4e34-8106-17b09cc9660a
	ReceivedAt *time.Time `json:"received_at"`
}

type WebhookEventIDStore struct {
	db *sql.DB
}

func (s *WebhookEventIDStore) MarkProcessed(ctx context.Context, sourceID, eventID string) (bool, error) {
    dbtx := dbtx.ExtractTx(ctx, s.db)

	query := `
		INSERT INTO webhook_event_ids (event_id, source_id)
        VALUES ($1, $2)
        ON CONFLICT (source_id, event_id) DO NOTHING
	`

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	result, err := dbtx.ExecContext(ctx, query, eventID, sourceID)
    if err != nil {
        return false, fmt.Errorf("recording webhook event %s/%s: %w", sourceID, eventID, err)
    }

    // rowsAffected == 0 means the INSERT was a no-op (duplicate)
    // rowsAffected == 1 means it's a new event

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("checking webhook insert result: %w", err)
	}

	return rowsAffected == 1, nil
}