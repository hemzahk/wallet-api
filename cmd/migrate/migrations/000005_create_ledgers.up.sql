CREATE TABLE IF NOT EXISTS ledgers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT,
    ledger_id TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

INSERT INTO ledgers (name, ledger_id, created_at)
VALUES 
    ('General Ledger', 'general_ledger_id', NOW()),
    ('Customer Ledger', 'customer_ledger_id', NOW()),
    ('Merchant Ledger', 'merchant_ledger_id', NOW());