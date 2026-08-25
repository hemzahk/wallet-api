package store

import (
	"context"
	"database/sql"
	"time"
)

type WebhookEventID struct {
	EventID  string `json:"event_id"`
	SourceID string `json:"source_id"` // for our PSP 645b44fb-314f-4e34-8106-17b09cc9660a
	ReceivedAt *time.Time `json:"received_at"`
}

type WebhookEventIDStore struct {
	db *sql.DB
}

func (s *WebhookEventIDStore) IsDuplicate(ctx context.Context, sourceID, eventID string) (bool, error) {
    rows, err := s.db.ExecContext(ctx, `
        INSERT INTO webhook_event_ids (event_id, source_id)
        VALUES ($1, $2)
        ON CONFLICT (source_id, event_id) DO NOTHING
    `, eventID, sourceID)
    if err != nil {
        return false, err
    }

    // rowsAffected == 0 means the INSERT was a no-op (duplicate)
    // rowsAffected == 1 means it's a new event
    // (check rows affected in your driver)

	rowsAffected, err := rows.RowsAffected()
	switch rowsAffected {
		case 0:
			return true, nil
		default:
			return false, nil
	}

}