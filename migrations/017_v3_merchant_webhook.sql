-- Aspira Pay V3: Merchant API Keys, Webhooks, Notification Log, Outbox Dead Letter

-- Merchant API Keys (§5.1.4)
CREATE TABLE IF NOT EXISTS merchant_api_keys (
    id BIGSERIAL PRIMARY KEY,
    key_id VARCHAR(64) UNIQUE NOT NULL,
    merchant_id VARCHAR(64) NOT NULL,
    api_key VARCHAR(256) NOT NULL,
    key_prefix VARCHAR(8) NOT NULL,
    scopes VARCHAR(128) NOT NULL DEFAULT 'read,write',
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    last_used TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_apikey_merchant ON merchant_api_keys(merchant_id);
CREATE INDEX IF NOT EXISTS idx_apikey_key ON merchant_api_keys(api_key);

-- Webhooks (§7.1)
CREATE TABLE IF NOT EXISTS webhooks (
    id BIGSERIAL PRIMARY KEY,
    webhook_id VARCHAR(64) UNIQUE NOT NULL,
    merchant_id VARCHAR(64) NOT NULL,
    url VARCHAR(512) NOT NULL,
    events VARCHAR(256) NOT NULL DEFAULT 'payment.completed',
    secret VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    retry_count INT NOT NULL DEFAULT 0,
    last_sent TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Webhook delivery log
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id BIGSERIAL PRIMARY KEY,
    webhook_id VARCHAR(64) NOT NULL,
    event_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(128) NOT NULL,
    payload TEXT NOT NULL,
    status_code INT,
    response_body TEXT,
    success BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_wh_deliveries_webhook ON webhook_deliveries(webhook_id);

-- Notification log
CREATE TABLE IF NOT EXISTS notifications (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    status VARCHAR(64) NOT NULL,
    channel VARCHAR(32) NOT NULL DEFAULT 'log',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Outbox dead letter queue
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0;
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS max_retries INT NOT NULL DEFAULT 10;
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMP;
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS last_error TEXT;
ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS dead_letter BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_outbox_dead ON outbox_events(dead_letter, next_retry_at);
