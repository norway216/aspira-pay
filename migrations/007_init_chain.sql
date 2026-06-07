-- 007: Chain Blocks & Chain Events tables
-- Aspira Pay V2 — Blockchain audit layer (Hash Chain with Merkle Tree)

CREATE TABLE IF NOT EXISTS chain_blocks (
    id BIGSERIAL PRIMARY KEY,
    block_height BIGINT NOT NULL UNIQUE,
    block_hash VARCHAR(128) NOT NULL,
    prev_hash VARCHAR(128) NOT NULL,
    merkle_root VARCHAR(128) NOT NULL,
    event_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_chain_blocks_height ON chain_blocks(block_height);
CREATE INDEX idx_chain_blocks_hash ON chain_blocks(block_hash);

CREATE TABLE IF NOT EXISTS chain_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    block_height BIGINT NOT NULL REFERENCES chain_blocks(block_height),
    payment_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload_hash VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_chain_events_event_id ON chain_events(event_id);
CREATE INDEX idx_chain_events_block ON chain_events(block_height);
CREATE INDEX idx_chain_events_payment ON chain_events(payment_id);

-- Genesis block
INSERT INTO chain_blocks (block_height, block_hash, prev_hash, merkle_root, event_count)
VALUES (0, '0000000000000000000000000000000000000000000000000000000000000000',
        '0000000000000000000000000000000000000000000000000000000000000000',
        '0000000000000000000000000000000000000000000000000000000000000000', 0)
ON CONFLICT (block_height) DO NOTHING;
