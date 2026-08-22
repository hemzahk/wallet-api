CREATE TABLE IF NOT EXISTS balances (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    balance_id TEXT NOT NULL UNIQUE,
    balance BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'DZD',
    ledger_id TEXT NOT NULL REFERENCES ledgers(ledger_id),
    identity_id UUID REFERENCES identities(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO balances (balance_id, ledger_id, created_at)
VALUES 
    ('@World', 'general_ledger_id', NOW()),
    ('@Revenue', 'general_ledger_id', NOW());