-- 009: Outbox Events table
-- Aspira Pay V2 — Transactional Outbox Pattern for reliable event publishing

CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    published BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_outbox_event_id ON outbox_events(event_id);
CREATE INDEX idx_outbox_published ON outbox_events(published, created_at);
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_id);
