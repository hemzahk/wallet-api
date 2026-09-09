CREATE UNIQUE INDEX uq_refund_active_per_transaction
    ON refund_requests (transaction_ref)
    WHERE status IN ('pending', 'approved');