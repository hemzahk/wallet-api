CREATE TABLE webhook_event_ids (
    event_id     TEXT        NOT NULL,
    source_id    UUID        NOT NULL,
    received_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source_id, event_id)
);