-- 005: Ledger Entries table
-- Aspira Pay V2 — Double-entry accounting ledger (append-only)

CREATE TABLE IF NOT EXISTS ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    entry_id VARCHAR(64) UNIQUE NOT NULL,
    event_id VARCHAR(64) NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    account_id VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    direction VARCHAR(16) NOT NULL CHECK (direction IN ('DEBIT', 'CREDIT')),
    amount BIGINT NOT NULL CHECK (amount > 0),
    balance_after BIGINT NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_entry_id ON ledger_entries(entry_id);
CREATE INDEX idx_ledger_payment_id ON ledger_entries(payment_id);
CREATE INDEX idx_ledger_account_id ON ledger_entries(account_id);
CREATE INDEX idx_ledger_event_id ON ledger_entries(event_id);
CREATE INDEX idx_ledger_created ON ledger_entries(created_at);

-- Prevent physical deletion (append-only enforcement)
CREATE OR REPLACE RULE ledger_no_delete AS ON DELETE TO ledger_entries
    DO INSTEAD NOTHING;
