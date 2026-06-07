package repository

import (
	"database/sql"
	"fmt"

	"github.com/aspira/aspira-pay/internal/domain/chain"
)

// InsertChainBlock inserts a new block in the hash chain.
func (db *DB) InsertChainBlock(b *chain.ChainBlock) error {
	query := `
		INSERT INTO chain_blocks (block_height, block_hash, prev_hash, merkle_root, event_count)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	return db.QueryRow(query,
		b.BlockHeight, b.BlockHash, b.PrevHash, b.MerkleRoot, b.EventCount,
	).Scan(&b.ID, &b.CreatedAt)
}

// InsertChainEvent inserts a chain event linked to a block.
func (db *DB) InsertChainEvent(e *chain.ChainEvent) error {
	query := `
		INSERT INTO chain_events (event_id, block_height, payment_id, event_type, payload_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	return db.QueryRow(query,
		e.EventID, e.BlockHeight, e.PaymentID, e.EventType, e.PayloadHash,
	).Scan(&e.ID, &e.CreatedAt)
}

// GetLatestBlock returns the most recent chain block.
func (db *DB) GetLatestBlock() (*chain.ChainBlock, error) {
	b := &chain.ChainBlock{}
	query := `SELECT id, block_height, block_hash, prev_hash, merkle_root, event_count, created_at
		FROM chain_blocks ORDER BY block_height DESC LIMIT 1`
	err := db.QueryRow(query).Scan(
		&b.ID, &b.BlockHeight, &b.BlockHash, &b.PrevHash,
		&b.MerkleRoot, &b.EventCount, &b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no blocks in chain")
	}
	return b, err
}

// GetChainBlock retrieves a block by height.
func (db *DB) GetChainBlock(height int64) (*chain.ChainBlock, error) {
	b := &chain.ChainBlock{}
	query := `SELECT id, block_height, block_hash, prev_hash, merkle_root, event_count, created_at
		FROM chain_blocks WHERE block_height = $1`
	err := db.QueryRow(query, height).Scan(
		&b.ID, &b.BlockHeight, &b.BlockHash, &b.PrevHash,
		&b.MerkleRoot, &b.EventCount, &b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chain block not found: height=%d", height)
	}
	return b, err
}

// GetChainBlocks returns a paginated list of blocks.
func (db *DB) GetChainBlocks(page, pageSize int) ([]chain.ChainBlock, int64, error) {
	var total int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM chain_blocks`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	query := `SELECT id, block_height, block_hash, prev_hash, merkle_root, event_count, created_at
		FROM chain_blocks ORDER BY block_height DESC LIMIT $1 OFFSET $2`
	rows, err := db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var blocks []chain.ChainBlock
	for rows.Next() {
		var b chain.ChainBlock
		if err := rows.Scan(
			&b.ID, &b.BlockHeight, &b.BlockHash, &b.PrevHash,
			&b.MerkleRoot, &b.EventCount, &b.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		blocks = append(blocks, b)
	}
	return blocks, total, nil
}

// GetChainEventsByPayment retrieves all chain events for a payment.
func (db *DB) GetChainEventsByPayment(paymentID string) ([]chain.ChainEvent, error) {
	query := `SELECT id, event_id, block_height, payment_id, event_type, payload_hash, created_at
		FROM chain_events WHERE payment_id = $1 ORDER BY created_at ASC`
	rows, err := db.Query(query, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []chain.ChainEvent
	for rows.Next() {
		var e chain.ChainEvent
		if err := rows.Scan(
			&e.ID, &e.EventID, &e.BlockHeight, &e.PaymentID,
			&e.EventType, &e.PayloadHash, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// GetChainEventsByBlock retrieves all events in a block.
func (db *DB) GetChainEventsByBlock(blockHeight int64) ([]chain.ChainEvent, error) {
	query := `SELECT id, event_id, block_height, payment_id, event_type, payload_hash, created_at
		FROM chain_events WHERE block_height = $1 ORDER BY created_at ASC`
	rows, err := db.Query(query, blockHeight)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []chain.ChainEvent
	for rows.Next() {
		var e chain.ChainEvent
		if err := rows.Scan(
			&e.ID, &e.EventID, &e.BlockHeight, &e.PaymentID,
			&e.EventType, &e.PayloadHash, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}
