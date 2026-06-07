-- 004: Payment Orders table
-- Aspira Pay V2 — Cross-border payment orders with full state machine

CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    payment_id VARCHAR(64) UNIQUE NOT NULL,
    request_id VARCHAR(128) UNIQUE NOT NULL,
    sender_user_id VARCHAR(64) NOT NULL,
    receiver_user_id VARCHAR(64) NOT NULL,
    source_currency VARCHAR(16) NOT NULL,
    target_currency VARCHAR(16) NOT NULL,
    source_amount BIGINT NOT NULL,
    target_amount BIGINT NOT NULL,
    fee_amount BIGINT NOT NULL DEFAULT 0,
    fx_rate NUMERIC(30, 12) NOT NULL,
    status VARCHAR(64) NOT NULL DEFAULT 'CREATED',
    risk_score INT DEFAULT 0,
    risk_reasons TEXT,
    quote_id VARCHAR(64),
    chain_tx_id VARCHAR(128),
    purpose VARCHAR(128),
    country_from VARCHAR(8),
    country_to VARCHAR(8),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_payment_id ON payment_orders(payment_id);
CREATE INDEX idx_payments_request_id ON payment_orders(request_id);
CREATE INDEX idx_payments_sender ON payment_orders(sender_user_id);
CREATE INDEX idx_payments_receiver ON payment_orders(receiver_user_id);
CREATE INDEX idx_payments_status ON payment_orders(status);
CREATE INDEX idx_payments_created ON payment_orders(created_at);
