-- Aspira Pay V3.0 — Ledger Voucher + Account Balance + Settlement Details
-- Architecture doc §5.5 (Ledger), §5.6 (Settlement), §5.7 (Reconciliation)

-- Voucher table (§5.5.4)
CREATE TABLE IF NOT EXISTS ledger_vouchers (
    id BIGSERIAL PRIMARY KEY,
    voucher_no VARCHAR(64) UNIQUE NOT NULL,
    business_type VARCHAR(64) NOT NULL,
    business_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_vouchers_business ON ledger_vouchers(business_id);

-- Account table with typed accounts (§5.5.2, §5.5.4)
CREATE TABLE IF NOT EXISTS accounts_v3 (
    id BIGSERIAL PRIMARY KEY,
    account_no VARCHAR(64) UNIQUE NOT NULL,
    owner_type VARCHAR(32) NOT NULL DEFAULT 'USER',
    owner_id VARCHAR(64) NOT NULL,
    account_type VARCHAR(64) NOT NULL DEFAULT 'USER_AVAILABLE',
    currency VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_accounts_v3_owner ON accounts_v3(owner_id, account_type);

-- Account balance with optimistic locking (§5.5.4, §5.5.5)
CREATE TABLE IF NOT EXISTS account_balances (
    account_no VARCHAR(64) PRIMARY KEY,
    available_balance BIGINT NOT NULL DEFAULT 0,
    frozen_balance BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(16) NOT NULL,
    version BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- Settlement details (§5.6)
CREATE TABLE IF NOT EXISTS settlement_details (
    id BIGSERIAL PRIMARY KEY,
    detail_id VARCHAR(64) UNIQUE NOT NULL,
    batch_id VARCHAR(64) NOT NULL,
    order_id VARCHAR(64) NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(16) NOT NULL,
    fee_amount BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_settlement_detail_batch ON settlement_details(batch_id);

-- Reconciliation records (§5.7)
CREATE TABLE IF NOT EXISTS reconciliation_records (
    id BIGSERIAL PRIMARY KEY,
    record_id VARCHAR(64) UNIQUE NOT NULL,
    batch_id VARCHAR(64),
    order_id VARCHAR(64),
    internal_count BIGINT NOT NULL DEFAULT 0,
    channel_count BIGINT NOT NULL DEFAULT 0,
    chain_count BIGINT NOT NULL DEFAULT 0,
    matched_count BIGINT NOT NULL DEFAULT 0,
    mismatched_count BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reconciliation_batch ON reconciliation_records(batch_id);

-- Channel receipts (§5.8)
CREATE TABLE IF NOT EXISTS channel_receipts (
    id BIGSERIAL PRIMARY KEY,
    receipt_id VARCHAR(64) UNIQUE NOT NULL,
    channel_name VARCHAR(64) NOT NULL,
    channel_tx_id VARCHAR(128),
    payment_id VARCHAR(64) NOT NULL,
    order_id VARCHAR(64) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(16) NOT NULL,
    receipt_hash VARCHAR(128),
    status VARCHAR(32) NOT NULL DEFAULT 'CONFIRMED',
    executed_at TIMESTAMP NOT NULL DEFAULT now(),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_channel_receipts_payment ON channel_receipts(payment_id);
