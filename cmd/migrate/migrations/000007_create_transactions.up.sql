CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    precise_amount BIGINT NOT NULL,
    reference TEXT,
    source TEXT,
    destination TEXT,
    status TEXT,
    description TEXT, 
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_source_balance FOREIGN KEY (source) REFERENCES balances (balance_id),
    CONSTRAINT fk_destination_balance FOREIGN KEY (destination) REFERENCES balances (balance_id)
);