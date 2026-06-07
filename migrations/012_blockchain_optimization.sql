-- Aspira Pay V2 — Blockchain Layer Optimization Migration
-- Architecture doc §7, §14: chain_batches, chain_submit_logs, chain_retry_queue tables.
-- Updates chain_blocks with batch tracking fields per §7.2.

-- Add batch tracking columns to existing chain_blocks (§7.2)
ALTER TABLE chain_blocks
    ADD COLUMN IF NOT EXISTS batch_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS start_sequence_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS end_sequence_id BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS audit_signature VARCHAR(256);

CREATE INDEX IF NOT EXISTS idx_chain_blocks_batch ON chain_blocks(batch_id);

-- Add batch tracking to chain_events (§7.3)
ALTER TABLE chain_events
    ADD COLUMN IF NOT EXISTS batch_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS merkle_proof JSONB;

CREATE INDEX IF NOT EXISTS idx_chain_events_batch ON chain_events(batch_id);

-- §14.1: Chain batches table — one row per Merkle batch
CREATE TABLE IF NOT EXISTS chain_batches (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) UNIQUE NOT NULL,
    merkle_root VARCHAR(128) NOT NULL,
    ledger_root_hash VARCHAR(128),
    event_count INT NOT NULL DEFAULT 0,
    start_sequence_id BIGINT NOT NULL DEFAULT 0,
    end_sequence_id BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    audit_signature VARCHAR(256),
    block_height BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    submitted_at TIMESTAMP,
    confirmed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chain_batches_status ON chain_batches(status);
CREATE INDEX IF NOT EXISTS idx_chain_batches_created ON chain_batches(created_at);

-- §14.2: Chain submission log — records every external blockchain submission attempt
CREATE TABLE IF NOT EXISTS chain_submit_logs (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) NOT NULL,
    chain_type VARCHAR(32) NOT NULL DEFAULT 'hash_chain',
    chain_tx_id VARCHAR(128),
    status VARCHAR(32) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chain_submit_logs_batch ON chain_submit_logs(batch_id);

-- §14.3: Chain retry queue — batches awaiting retry after submission failure
CREATE TABLE IF NOT EXISTS chain_retry_queue (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) UNIQUE NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    max_retries INT NOT NULL DEFAULT 10,
    next_retry_at TIMESTAMP NOT NULL,
    last_error TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chain_retry_next ON chain_retry_queue(next_retry_at, status);
