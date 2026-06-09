-- Aspira Pay V4: Account Transfer + Payment Link + Transfer Contact (§9)

CREATE TABLE IF NOT EXISTS transfer_orders (
    id BIGSERIAL PRIMARY KEY,
    transfer_id VARCHAR(64) UNIQUE NOT NULL,
    payer_user_id VARCHAR(64) NOT NULL,
    payer_account_id VARCHAR(64) NOT NULL,
    receiver_user_id VARCHAR(64) NOT NULL,
    receiver_account_id VARCHAR(64) NOT NULL,
    source_currency VARCHAR(8) NOT NULL,
    target_currency VARCHAR(8) NOT NULL,
    source_amount BIGINT NOT NULL,
    target_amount BIGINT NOT NULL,
    fee_amount BIGINT NOT NULL DEFAULT 0,
    fx_rate VARCHAR(32),
    quote_id VARCHAR(64),
    payment_link_id VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    remark TEXT,
    idempotency_key VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    completed_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_transfer_payer ON transfer_orders(payer_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfer_receiver ON transfer_orders(receiver_user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_transfer_idempotency ON transfer_orders(payer_user_id, idempotency_key);

CREATE TABLE IF NOT EXISTS transfer_contacts (
    id BIGSERIAL PRIMARY KEY,
    contact_id VARCHAR(64) UNIQUE NOT NULL,
    owner_user_id VARCHAR(64) NOT NULL,
    target_user_id VARCHAR(64) NOT NULL,
    target_account_id VARCHAR(64) NOT NULL,
    target_display_name VARCHAR(128),
    target_aspira_id VARCHAR(64),
    target_account_no_masked VARCHAR(32),
    target_currency VARCHAR(8),
    last_transfer_at TIMESTAMP,
    transfer_count INT NOT NULL DEFAULT 0,
    total_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_unique ON transfer_contacts(owner_user_id, target_user_id, target_account_id);

CREATE TABLE IF NOT EXISTS payment_links (
    id BIGSERIAL PRIMARY KEY,
    payment_link_id VARCHAR(64) UNIQUE NOT NULL,
    link_token_hash VARCHAR(128) NOT NULL UNIQUE,
    link_token_prefix VARCHAR(16) NOT NULL,
    creator_user_id VARCHAR(64) NOT NULL,
    receiver_account_id VARCHAR(64) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(8) NOT NULL,
    title VARCHAR(128),
    description TEXT,
    expire_at TIMESTAMP NOT NULL,
    max_pay_count INT NOT NULL DEFAULT 1,
    paid_count INT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    paid_at TIMESTAMP,
    cancelled_at TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_plink_creator ON payment_links(creator_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_plink_status_expire ON payment_links(status, expire_at);
