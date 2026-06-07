-- 003: KYC Profiles table
-- Aspira Pay V2 — Know Your Customer identity verification

CREATE TABLE IF NOT EXISTS kyc_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(user_id),
    full_name VARCHAR(255) NOT NULL,
    nationality VARCHAR(64),
    date_of_birth DATE,
    document_type VARCHAR(64),
    document_number_hash VARCHAR(255),
    document_hash VARCHAR(255),
    address_hash VARCHAR(255),
    kyc_status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    risk_level VARCHAR(32) NOT NULL DEFAULT 'LOW',
    rejection_reason TEXT,
    reviewed_by VARCHAR(64),
    reviewed_at TIMESTAMP,
    submitted_at TIMESTAMP NOT NULL DEFAULT now(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_kyc_user_id ON kyc_profiles(user_id);
CREATE INDEX idx_kyc_status ON kyc_profiles(kyc_status);
