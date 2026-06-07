-- 008: Idempotency Keys table
-- Aspira Pay V2 — Idempotency control for all critical APIs

CREATE TABLE IF NOT EXISTS idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(128) UNIQUE NOT NULL,
    request_hash VARCHAR(255) NOT NULL,
    response_body JSONB,
    response_status INT NOT NULL DEFAULT 200,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_idempotency_request ON idempotency_keys(request_id);
