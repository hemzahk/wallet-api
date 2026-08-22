CREATE TABLE IF NOT EXISTS checkout_sessions (
    id UUID PRIMARY KEY,
    token BYTEA NOT NULL UNIQUE,
    merchant_id UUID NOT NULL REFERENCES merchants(id),
    amount BIGINT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);