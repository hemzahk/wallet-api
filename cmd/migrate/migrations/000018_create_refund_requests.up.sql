CREATE TABLE IF NOT EXISTS refund_requests(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_ref TEXT NOT NULL REFERENCES transactions(reference),
    status TEXT,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);