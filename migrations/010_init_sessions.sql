-- Aspira Pay V2 — Login Sessions, User Devices, Account Limits
-- Architecture doc §6.2: user_devices, login_sessions tables
-- Architecture doc §6.4: account_limits for risk-based transaction limits
-- Architecture doc §10.1-10.3: Users, Accounts, KYC tables

-- Login sessions table
CREATE TABLE IF NOT EXISTS login_sessions (
    id BIGSERIAL PRIMARY KEY,
    session_id VARCHAR(128) UNIQUE NOT NULL,
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    refresh_token_hash VARCHAR(255) NOT NULL,
    device_fingerprint VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    expires_at TIMESTAMP NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_login_sessions_user_id ON login_sessions(user_id);
CREATE INDEX idx_login_sessions_expires ON login_sessions(expires_at) WHERE NOT revoked;

-- User devices table
CREATE TABLE IF NOT EXISTS user_devices (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    device_id VARCHAR(128) UNIQUE NOT NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(64),
    ip_address VARCHAR(45),
    last_seen_at TIMESTAMP NOT NULL DEFAULT now(),
    trusted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_devices_user_id ON user_devices(user_id);

-- Account limits (per-user per-currency risk limits)
CREATE TABLE IF NOT EXISTS account_limits (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    currency VARCHAR(16) NOT NULL,
    daily_limit BIGINT NOT NULL DEFAULT 100000000,      -- $1,000,000 in cents (default LOW risk)
    monthly_limit BIGINT NOT NULL DEFAULT 1000000000,    -- $10,000,000 in cents
    single_tx_limit BIGINT NOT NULL DEFAULT 10000000,    -- $100,000 in cents
    max_tx_per_day INT NOT NULL DEFAULT 50,
    risk_level VARCHAR(32) NOT NULL DEFAULT 'LOW',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(user_id, currency)
);
