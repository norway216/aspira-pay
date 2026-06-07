-- 002: Accounts table
-- Aspira Pay V2 — User accounts with multi-currency balances

CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(64) UNIQUE NOT NULL,
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    currency VARCHAR(16) NOT NULL,
    available_balance BIGINT NOT NULL DEFAULT 0,
    frozen_balance BIGINT NOT NULL DEFAULT 0,
    settled_balance BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'NORMAL',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(user_id, currency)
);

CREATE INDEX idx_accounts_account_id ON accounts(account_id);
CREATE INDEX idx_accounts_user_id ON accounts(user_id);

-- System accounts for double-entry settlement
INSERT INTO accounts (account_id, user_id, currency, available_balance, status)
VALUES
    ('sys_settlement_usd', 'system', 'USD', 0, 'SYSTEM'),
    ('sys_settlement_jpy', 'system', 'JPY', 0, 'SYSTEM'),
    ('sys_settlement_eur', 'system', 'EUR', 0, 'SYSTEM'),
    ('sys_settlement_cny', 'system', 'CNY', 0, 'SYSTEM'),
    ('sys_fee_income_usd', 'system', 'USD', 0, 'SYSTEM'),
    ('sys_fee_income_jpy', 'system', 'JPY', 0, 'SYSTEM'),
    ('sys_fee_income_eur', 'system', 'EUR', 0, 'SYSTEM'),
    ('sys_fee_income_cny', 'system', 'CNY', 0, 'SYSTEM')
ON CONFLICT (account_id) DO NOTHING;
