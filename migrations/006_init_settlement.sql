-- 006: Settlement Batches table
-- Aspira Pay V2 — Settlement batch management for batching ledger entries

CREATE TABLE IF NOT EXISTS settlement_batches (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) UNIQUE NOT NULL,
    currency VARCHAR(16) NOT NULL,
    total_debit BIGINT NOT NULL DEFAULT 0,
    total_credit BIGINT NOT NULL DEFAULT 0,
    entry_count INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'OPEN',
    ledger_root_hash VARCHAR(128),
    chain_tx_id VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_settlement_batch_id ON settlement_batches(batch_id);
CREATE INDEX idx_settlement_status ON settlement_batches(status);
CREATE INDEX idx_settlement_currency ON settlement_batches(currency);
