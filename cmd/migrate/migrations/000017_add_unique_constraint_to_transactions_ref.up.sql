-- Migration: add unique constraint on transactions.reference
ALTER TABLE transactions
    ADD CONSTRAINT uq_transactions_reference UNIQUE (reference);