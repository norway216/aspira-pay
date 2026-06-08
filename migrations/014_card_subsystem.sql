-- Aspira Pay Card Payment Subsystem (§15)
-- Architecture doc: Card, Authorization, Transaction, FX Quote, Fee Rule tables.

CREATE TABLE IF NOT EXISTS cards (
    id BIGSERIAL PRIMARY KEY,
    card_id VARCHAR(64) UNIQUE NOT NULL,
    owner_type VARCHAR(32) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    card_token VARCHAR(128) UNIQUE NOT NULL,
    pan_last4 VARCHAR(4) NOT NULL,
    card_network VARCHAR(32) NOT NULL,
    card_type VARCHAR(32) NOT NULL DEFAULT 'DEBIT',
    card_form VARCHAR(32) NOT NULL DEFAULT 'VIRTUAL',
    expiry_month INT NOT NULL,
    expiry_year INT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ISSUING',
    default_currency VARCHAR(16) DEFAULT 'USD',
    daily_limit BIGINT NOT NULL DEFAULT 500000,
    monthly_limit BIGINT NOT NULL DEFAULT 5000000,
    single_tx_limit BIGINT NOT NULL DEFAULT 500000,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cards_owner ON cards(owner_id);
CREATE INDEX IF NOT EXISTS idx_cards_token ON cards(card_token);
CREATE INDEX IF NOT EXISTS idx_cards_status ON cards(status);

CREATE TABLE IF NOT EXISTS card_authorizations (
    id BIGSERIAL PRIMARY KEY,
    auth_id VARCHAR(64) UNIQUE NOT NULL,
    card_id VARCHAR(64) NOT NULL,
    merchant_name VARCHAR(256),
    merchant_country VARCHAR(16),
    merchant_category_code VARCHAR(16),
    transaction_amount BIGINT NOT NULL,
    transaction_currency VARCHAR(16) NOT NULL,
    debit_amount BIGINT NOT NULL DEFAULT 0,
    debit_currency VARCHAR(16) NOT NULL DEFAULT '',
    fx_rate VARCHAR(32),
    fee_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(64) NOT NULL,
    decline_reason VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_card_auth_card ON card_authorizations(card_id);
CREATE INDEX IF NOT EXISTS idx_card_auth_status ON card_authorizations(status);

CREATE TABLE IF NOT EXISTS card_transactions (
    id BIGSERIAL PRIMARY KEY,
    tx_id VARCHAR(64) UNIQUE NOT NULL,
    auth_id VARCHAR(64),
    card_id VARCHAR(64) NOT NULL,
    transaction_amount BIGINT NOT NULL,
    transaction_currency VARCHAR(16) NOT NULL,
    debit_amount BIGINT NOT NULL DEFAULT 0,
    debit_currency VARCHAR(16) NOT NULL DEFAULT '',
    fx_rate VARCHAR(32),
    fee_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(64) NOT NULL DEFAULT 'PENDING',
    settlement_date DATE,
    receipt_hash VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_card_tx_card ON card_transactions(card_id);
CREATE INDEX IF NOT EXISTS idx_card_tx_auth ON card_transactions(auth_id);

CREATE TABLE IF NOT EXISTS fx_quotes (
    id BIGSERIAL PRIMARY KEY,
    quote_id VARCHAR(64) UNIQUE NOT NULL,
    source_currency VARCHAR(16) NOT NULL,
    target_currency VARCHAR(16) NOT NULL,
    source_amount BIGINT,
    target_amount BIGINT,
    mid_rate VARCHAR(32) NOT NULL,
    applied_rate VARCHAR(32) NOT NULL,
    fee_rate VARCHAR(32) NOT NULL DEFAULT '0',
    fee_amount BIGINT NOT NULL DEFAULT 0,
    valid_until TIMESTAMP NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fee_rules (
    id BIGSERIAL PRIMARY KEY,
    rule_id VARCHAR(64) UNIQUE NOT NULL,
    scenario VARCHAR(64) NOT NULL,
    source_currency VARCHAR(16),
    target_currency VARCHAR(16),
    country VARCHAR(16),
    card_network VARCHAR(32),
    percentage_fee VARCHAR(32) NOT NULL DEFAULT '0',
    fixed_fee BIGINT NOT NULL DEFAULT 0,
    min_fee BIGINT,
    max_fee BIGINT,
    risk_level VARCHAR(32),
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE'
);

-- Default fee rules (§9.3)
INSERT INTO fee_rules (rule_id, scenario, percentage_fee, fixed_fee, status) VALUES
    ('FR_SAME_CURRENCY', 'CARD_SAME_CURRENCY_SPEND', '0', 0, 'ACTIVE'),
    ('FR_CROSS_CURRENCY', 'CARD_CROSS_CURRENCY_SPEND', '0.0045', 20, 'ACTIVE'),
    ('FR_ATM', 'CARD_ATM_WITHDRAWAL', '0.01', 150, 'ACTIVE'),
    ('FR_REFUND', 'CARD_REFUND', '0', 0, 'ACTIVE'),
    ('FR_CHARGEBACK', 'CARD_CHARGEBACK', '0', 1500, 'ACTIVE')
ON CONFLICT (rule_id) DO NOTHING;
