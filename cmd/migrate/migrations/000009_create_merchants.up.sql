CREATE TABLE IF NOT EXISTS merchants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    business_name TEXT NOT NULL UNIQUE,
    fee_rate TEXT NOT NULL DEFAULT '0',
    cashback_rate TEXT NOT NULL DEFAULT '0',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);