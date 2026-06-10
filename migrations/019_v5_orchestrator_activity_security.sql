-- Aspira Pay V5: Transaction Orchestrator, Activity Feed, Balance Snapshot, Security

-- §14.5: Activity Feed table
CREATE TABLE IF NOT EXISTS activity_feed (
    activity_id   VARCHAR(64) PRIMARY KEY,
    user_id       VARCHAR(64) NOT NULL,
    activity_type VARCHAR(64) NOT NULL,
    ref_type      VARCHAR(64) NOT NULL,
    ref_id        VARCHAR(64) NOT NULL,
    title         VARCHAR(128) NOT NULL,
    subtitle      VARCHAR(256),
    amount        BIGINT,
    currency      VARCHAR(8),
    status        VARCHAR(32),
    created_at    TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_activity_user ON activity_feed(user_id, created_at DESC);

-- §13.4: Balance snapshot for recovery & audit
CREATE TABLE IF NOT EXISTS account_balance_snapshots (
    snapshot_id       VARCHAR(64) PRIMARY KEY,
    account_id        VARCHAR(64) NOT NULL,
    currency          VARCHAR(8) NOT NULL,
    available_balance BIGINT NOT NULL,
    frozen_balance    BIGINT NOT NULL,
    ledger_balance    BIGINT NOT NULL,
    last_ledger_seq   BIGINT NOT NULL DEFAULT 0,
    created_at        TIMESTAMP NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_bal_snapshot_acct ON account_balance_snapshots(account_id, created_at DESC);

-- §13.2: Ledger Batch for transaction-level debit/credit verification
CREATE TABLE IF NOT EXISTS ledger_batches (
    batch_id         VARCHAR(64) PRIMARY KEY,
    transaction_id   VARCHAR(64) NOT NULL,
    transaction_type VARCHAR(64) NOT NULL,
    status           VARCHAR(32) NOT NULL DEFAULT 'pending',
    debit_total      BIGINT NOT NULL,
    credit_total     BIGINT NOT NULL,
    currency         VARCHAR(8) NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT now(),
    committed_at     TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ledger_batch_tx ON ledger_batches(transaction_id);

-- §12.3: Funds reservation tracking
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS reserved_balance BIGINT NOT NULL DEFAULT 0;

-- §12.4: Idempotency enhancement — add ledger-level idempotency
CREATE TABLE IF NOT EXISTS ledger_idempotency (
    transaction_id   VARCHAR(64) NOT NULL,
    operation_type   VARCHAR(64) NOT NULL,
    ledger_batch_id  VARCHAR(64),
    created_at       TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (transaction_id, operation_type)
);

-- §21.2: Unified error code registry
CREATE TABLE IF NOT EXISTS error_codes (
    code        VARCHAR(64) PRIMARY KEY,
    category    VARCHAR(32) NOT NULL,
    message     VARCHAR(256) NOT NULL,
    http_status INT NOT NULL DEFAULT 400,
    retryable   BOOLEAN NOT NULL DEFAULT false
);

INSERT INTO error_codes (code, category, message, http_status, retryable) VALUES
    ('AUTH_INVALID_TOKEN', 'AUTH', 'Invalid or expired token', 401, false),
    ('AUTH_INSUFFICIENT_PERMISSIONS', 'AUTH', 'Insufficient permissions', 403, false),
    ('ACCOUNT_NOT_FOUND', 'ACCOUNT', 'Account not found', 404, false),
    ('ACCOUNT_FROZEN', 'ACCOUNT', 'Account is frozen', 403, false),
    ('ACCOUNT_INSUFFICIENT_BALANCE', 'ACCOUNT', 'Insufficient available balance', 400, false),
    ('TRANSFER_INVALID_STATE', 'TRANSFER', 'Invalid transfer state transition', 400, false),
    ('TRANSFER_RECIPIENT_NOT_FOUND', 'TRANSFER', 'Recipient not found', 404, false),
    ('TRANSFER_LIMIT_EXCEEDED', 'TRANSFER', 'Transfer exceeds limit', 400, false),
    ('PAYMENT_LINK_EXPIRED', 'PAYMENT_LINK', 'Payment link has expired', 400, false),
    ('PAYMENT_LINK_ALREADY_PAID', 'PAYMENT_LINK', 'Payment link already paid', 400, false),
    ('RISK_REJECTED', 'RISK', 'Transaction rejected by risk engine', 400, false),
    ('RISK_MANUAL_REVIEW', 'RISK', 'Transaction requires manual review', 202, true),
    ('LEDGER_COMMIT_FAILED', 'LEDGER', 'Ledger commit failed', 500, true),
    ('LEDGER_IMBALANCE', 'LEDGER', 'Debit/credit imbalance detected', 500, false),
    ('FX_RATE_UNAVAILABLE', 'FX', 'Exchange rate unavailable', 503, true),
    ('SYSTEM_INTERNAL_ERROR', 'SYSTEM', 'Internal server error', 500, true),
    ('SYSTEM_SERVICE_UNAVAILABLE', 'SYSTEM', 'Service temporarily unavailable', 503, true)
ON CONFLICT (code) DO NOTHING;
