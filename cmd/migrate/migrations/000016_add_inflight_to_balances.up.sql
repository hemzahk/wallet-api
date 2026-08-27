ALTER TABLE balances ADD COLUMN inflight_credit_balance BIGINT NOT NULL DEFAULT 0;
ALTER TABLE balances ADD COLUMN inflight_debit_balance BIGINT NOT NULL DEFAULT 0;