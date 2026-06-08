-- Aspira Pay V2 Architecture: Admin Audit, Wallet, Card Review, External Card Binding, KYC Enhancement

-- §14.2: Admin operation audit log
CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_id VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    target_type VARCHAR(64) NOT NULL,
    target_id VARCHAR(64) NOT NULL,
    details JSONB,
    ip_address VARCHAR(45),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_admin_audit_admin ON admin_audit_logs(admin_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_target ON admin_audit_logs(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_created ON admin_audit_logs(created_at);

-- §5.3: Login attempt tracking (rate limiting)
CREATE TABLE IF NOT EXISTS login_attempts (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(128) NOT NULL,
    ip_address VARCHAR(45),
    success BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_login_attempts_user ON login_attempts(username, created_at);

-- §8: Wallet accounts (explicit wallet concept)
CREATE TABLE IF NOT EXISTS wallet_accounts (
    id BIGSERIAL PRIMARY KEY,
    wallet_id VARCHAR(64) UNIQUE NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    available_balance BIGINT NOT NULL DEFAULT 0,
    frozen_balance BIGINT NOT NULL DEFAULT 0,
    total_balance BIGINT GENERATED ALWAYS AS (available_balance + frozen_balance) STORED,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(user_id, currency)
);
CREATE INDEX IF NOT EXISTS idx_wallet_user ON wallet_accounts(user_id);

-- §7.1: External bank card binding
CREATE TABLE IF NOT EXISTS external_cards (
    id BIGSERIAL PRIMARY KEY,
    card_id VARCHAR(64) UNIQUE NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    card_token VARCHAR(128) UNIQUE NOT NULL,
    cardholder_name VARCHAR(256) NOT NULL,
    last4 VARCHAR(4) NOT NULL,
    card_network VARCHAR(32) NOT NULL,
    card_type VARCHAR(32) NOT NULL DEFAULT 'DEBIT',
    issuing_country VARCHAR(16),
    expiry_month INT NOT NULL,
    expiry_year INT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ext_card_user ON external_cards(user_id);

-- Card application: add review status
ALTER TABLE cards ADD COLUMN IF NOT EXISTS application_status VARCHAR(32) DEFAULT 'APPROVED';
ALTER TABLE cards ADD COLUMN IF NOT EXISTS reviewed_by VARCHAR(64) DEFAULT '';
ALTER TABLE cards ADD COLUMN IF NOT EXISTS review_notes VARCHAR(512) DEFAULT '';

-- §6.2: KYC document type and expiry
ALTER TABLE kyc_profiles ADD COLUMN IF NOT EXISTS document_number VARCHAR(128) DEFAULT '';
ALTER TABLE kyc_profiles ADD COLUMN IF NOT EXISTS document_expiry VARCHAR(16) DEFAULT '';

-- §5.3: Password strength config
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_count INT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP;
