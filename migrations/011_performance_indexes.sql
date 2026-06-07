-- Aspira Pay V2 — Performance Optimization Indexes
-- Added after bottleneck analysis of high-frequency queries.
-- See docs/development-issues.md §16 for details.

-- Composite index for risk engine's daily total check (GetDailyTotal).
-- Query pattern: WHERE sender_user_id = $1 AND created_at >= $2 AND created_at < $3 AND status NOT IN (...)
-- Enables index-only scan for the risk assessment hot path.
CREATE INDEX IF NOT EXISTS idx_payments_sender_created_status
    ON payment_orders(sender_user_id, created_at, status);

-- Composite index for risk engine's recent transaction count (GetRecentTxCount).
-- Query pattern: WHERE sender_user_id = $1 AND created_at > $2
-- Already partially covered by idx_payments_sender_created_status above.
-- This is a narrower index for the COUNT-only query.
CREATE INDEX IF NOT EXISTS idx_payments_sender_created
    ON payment_orders(sender_user_id, created_at);

-- Composite index for open settlement batch lookup (GetOpenSettlementBatch).
-- Query pattern: WHERE currency = $1 AND status = 'OPEN' ORDER BY created_at DESC LIMIT 1
-- Prevents sequential scan over all batches when finding the current open batch.
CREATE INDEX IF NOT EXISTS idx_settlement_currency_status
    ON settlement_batches(currency, status);

-- Composite index for KYC pending review queue (ListKYCPending).
-- Query pattern: WHERE kyc_status IN ('PENDING', 'MANUAL_REVIEW') ORDER BY submitted_at ASC
-- Speeds up the compliance reviewer's queue page.
CREATE INDEX IF NOT EXISTS idx_kyc_status_submitted
    ON kyc_profiles(kyc_status, submitted_at);

-- Index for chain events by payment (GetChainEventsByPayment).
-- Query pattern: WHERE payment_id = $1 ORDER BY created_at ASC
-- Already indexed by idx_chain_events_payment, but adding composite with created_at
-- to avoid a separate sort step.
CREATE INDEX IF NOT EXISTS idx_chain_events_payment_created
    ON chain_events(payment_id, created_at);
