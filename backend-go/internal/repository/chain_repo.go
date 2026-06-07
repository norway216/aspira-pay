package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/chain"
	"github.com/lib/pq"
)

// ──────────────────────────────────────────────
// Chain Blocks (§7.2)
// ──────────────────────────────────────────────

// InsertChainBlock inserts a new block in the hash chain.
func (db *DB) InsertChainBlock(b *chain.ChainBlock) error {
	query := `
		INSERT INTO chain_blocks (block_height, block_hash, prev_hash, merkle_root, event_count,
			batch_id, start_sequence_id, end_sequence_id, audit_signature)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return db.QueryRow(query,
		b.BlockHeight, b.BlockHash, b.PrevHash, b.MerkleRoot, b.EventCount,
		b.BatchID, b.StartSequenceID, b.EndSequenceID, nullString(b.AuditSignature),
	).Scan(&b.ID, &b.CreatedAt)
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetLatestBlock returns the most recent chain block.
func (db *DB) GetLatestBlock() (*chain.ChainBlock, error) {
	b := &chain.ChainBlock{}
	query := `SELECT id, block_height, block_hash, prev_hash, merkle_root, event_count,
		COALESCE(batch_id, ''), start_sequence_id, end_sequence_id,
		COALESCE(audit_signature, ''), created_at
		FROM chain_blocks ORDER BY block_height DESC LIMIT 1`
	err := db.QueryRow(query).Scan(
		&b.ID, &b.BlockHeight, &b.BlockHash, &b.PrevHash,
		&b.MerkleRoot, &b.EventCount,
		&b.BatchID, &b.StartSequenceID, &b.EndSequenceID,
		&b.AuditSignature, &b.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no blocks in chain")
	}
	return b, err
}

// GetChainBlock retrieves a block by height.
func (db *DB) GetChainBlock(height int64) (*chain.ChainBlock, error) {
	b := &chain.ChainBlock{}
	query := `SELECT id, block_height, block_hash, prev_hash, merkle_root, event_count,
		COALESCE(batch_id, ''), start_sequence_id, end_sequence_id,
		COALESCE(audit_signature, ''), created_at
		FROM chain_blocks WHERE block_height = $1`
	err := db.QueryRow(query, height).Scan(
		&b.ID, &b.BlockHeight, &b.BlockHash, &b.PrevHash,
		&b.MerkleRoot, &b.EventCount,
		&b.BatchID, &b.StartSequenceID, &b.EndSequenceID,
		&b.AuditSignature, &b.CreatedAt,
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

	query := `SELECT id, block_height, block_hash, prev_hash, merkle_root, event_count,
		COALESCE(batch_id, ''), start_sequence_id, end_sequence_id,
		COALESCE(audit_signature, ''), created_at
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
			&b.MerkleRoot, &b.EventCount,
			&b.BatchID, &b.StartSequenceID, &b.EndSequenceID,
			&b.AuditSignature, &b.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		blocks = append(blocks, b)
	}
	return blocks, total, nil
}

// ──────────────────────────────────────────────
// Chain Events (§7.3)
// ──────────────────────────────────────────────

// InsertChainEvent inserts a chain event.
func (db *DB) InsertChainEvent(e *chain.ChainEvent) error {
	query := `
		INSERT INTO chain_events (event_id, block_height, payment_id, event_type, payload_hash, batch_id, merkle_proof)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	var merkleProofJSON interface{}
	if len(e.MerkleProof) > 0 {
		b, _ := json.Marshal(e.MerkleProof)
		merkleProofJSON = string(b)
	}
	return db.QueryRow(query,
		e.EventID, e.BlockHeight, e.PaymentID, e.EventType, e.PayloadHash,
		nullString(e.BatchID), merkleProofJSON,
	).Scan(&e.ID, &e.CreatedAt)
}

// InsertChainEventsBatch inserts multiple chain events in a single query.
func (db *DB) InsertChainEventsBatch(events []chain.ChainEvent) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := db.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO chain_events (event_id, block_height, payment_id, event_type, payload_hash, batch_id, merkle_proof)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := range events {
		var mp interface{}
		if len(events[i].MerkleProof) > 0 {
			b, _ := json.Marshal(events[i].MerkleProof)
			mp = string(b)
		}
		if err := stmt.QueryRow(
			events[i].EventID, events[i].BlockHeight, events[i].PaymentID,
			events[i].EventType, events[i].PayloadHash,
			nullString(events[i].BatchID), mp,
		).Scan(&events[i].ID, &events[i].CreatedAt); err != nil {
			return fmt.Errorf("chain event insert at index %d: %w", i, err)
		}
	}
	return tx.Commit()
}

// GetChainEventsByPayment retrieves all chain events for a payment.
func (db *DB) GetChainEventsByPayment(paymentID string) ([]chain.ChainEvent, error) {
	query := `SELECT id, event_id, block_height, payment_id, event_type, payload_hash,
		COALESCE(batch_id, ''), merkle_proof, created_at
		FROM chain_events WHERE payment_id = $1 ORDER BY created_at ASC`
	rows, err := db.Query(query, paymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []chain.ChainEvent
	for rows.Next() {
		var e chain.ChainEvent
		var mpJSON sql.NullString
		if err := rows.Scan(
			&e.ID, &e.EventID, &e.BlockHeight, &e.PaymentID,
			&e.EventType, &e.PayloadHash, &e.BatchID, &mpJSON, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if mpJSON.Valid && mpJSON.String != "" {
			json.Unmarshal([]byte(mpJSON.String), &e.MerkleProof)
		}
		events = append(events, e)
	}
	return events, nil
}

// GetChainEventsByBatch retrieves all events in a batch.
func (db *DB) GetChainEventsByBatch(batchID string) ([]chain.ChainEvent, error) {
	query := `SELECT id, event_id, block_height, payment_id, event_type, payload_hash,
		COALESCE(batch_id, ''), merkle_proof, created_at
		FROM chain_events WHERE batch_id = $1 ORDER BY created_at ASC`
	rows, err := db.Query(query, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []chain.ChainEvent
	for rows.Next() {
		var e chain.ChainEvent
		var mpJSON sql.NullString
		if err := rows.Scan(
			&e.ID, &e.EventID, &e.BlockHeight, &e.PaymentID,
			&e.EventType, &e.PayloadHash, &e.BatchID, &mpJSON, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if mpJSON.Valid && mpJSON.String != "" {
			json.Unmarshal([]byte(mpJSON.String), &e.MerkleProof)
		}
		events = append(events, e)
	}
	return events, nil
}

// ──────────────────────────────────────────────
// Chain Batches (§14.1)
// ──────────────────────────────────────────────

// CreateChainBatch inserts a new chain batch.
func (db *DB) CreateChainBatch(b *chain.ChainBatch) error {
	query := `
		INSERT INTO chain_batches (batch_id, merkle_root, ledger_root_hash, event_count,
			start_sequence_id, end_sequence_id, status, audit_signature, block_height)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at`
	return db.QueryRow(query,
		b.BatchID, b.MerkleRoot, nullString(b.LedgerRootHash), b.EventCount,
		b.StartSequenceID, b.EndSequenceID, b.Status, nullString(b.AuditSignature),
		b.BlockHeight,
	).Scan(&b.ID, &b.CreatedAt)
}

// UpdateChainBatchStatus updates the batch status and optional block link.
func (db *DB) UpdateChainBatchStatus(batchID string, status chain.BatchStatus, blockHeight *int64) error {
	var bh interface{}
	if blockHeight != nil {
		bh = *blockHeight
	}
	query := `UPDATE chain_batches SET status = $1, block_height = $2,
		confirmed_at = CASE WHEN $1 = 'CONFIRMED' THEN now() ELSE confirmed_at END,
		submitted_at = CASE WHEN $1 = 'SUBMITTING' THEN now() ELSE submitted_at END
		WHERE batch_id = $3`
	_, err := db.Exec(query, status, bh, batchID)
	return err
}

// UpdateChainBatchSignature sets the audit signature on a confirmed batch.
func (db *DB) UpdateChainBatchSignature(batchID, signature string) error {
	query := `UPDATE chain_batches SET audit_signature = $1 WHERE batch_id = $2`
	_, err := db.Exec(query, signature, batchID)
	return err
}

// GetChainBatch retrieves a batch by ID.
func (db *DB) GetChainBatch(batchID string) (*chain.ChainBatch, error) {
	b := &chain.ChainBatch{}
	query := `SELECT id, batch_id, merkle_root, COALESCE(ledger_root_hash, ''), event_count,
		start_sequence_id, end_sequence_id, status, COALESCE(audit_signature, ''),
		block_height, created_at, submitted_at, confirmed_at
		FROM chain_batches WHERE batch_id = $1`
	err := db.QueryRow(query, batchID).Scan(
		&b.ID, &b.BatchID, &b.MerkleRoot, &b.LedgerRootHash, &b.EventCount,
		&b.StartSequenceID, &b.EndSequenceID, &b.Status, &b.AuditSignature,
		&b.BlockHeight, &b.CreatedAt, &b.SubmittedAt, &b.ConfirmedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("chain batch not found: %s", batchID)
	}
	return b, err
}

// ListPendingBatches returns batches that are not yet confirmed.
func (db *DB) ListPendingBatches(limit int) ([]chain.ChainBatch, error) {
	query := `SELECT id, batch_id, merkle_root, COALESCE(ledger_root_hash, ''), event_count,
		start_sequence_id, end_sequence_id, status, COALESCE(audit_signature, ''),
		block_height, created_at, submitted_at, confirmed_at
		FROM chain_batches WHERE status IN ('PENDING', 'BUILDING', 'SUBMITTING', 'FAILED_RETRYABLE')
		ORDER BY created_at ASC LIMIT $1`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var batches []chain.ChainBatch
	for rows.Next() {
		var b chain.ChainBatch
		if err := rows.Scan(
			&b.ID, &b.BatchID, &b.MerkleRoot, &b.LedgerRootHash, &b.EventCount,
			&b.StartSequenceID, &b.EndSequenceID, &b.Status, &b.AuditSignature,
			&b.BlockHeight, &b.CreatedAt, &b.SubmittedAt, &b.ConfirmedAt,
		); err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, nil
}

// ──────────────────────────────────────────────
// Submit Logs (§14.2)
// ──────────────────────────────────────────────

// InsertSubmitLog records a blockchain submission attempt.
func (db *DB) InsertSubmitLog(log *chain.ChainSubmitLog) error {
	query := `INSERT INTO chain_submit_logs (batch_id, chain_type, chain_tx_id, status, error_message)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return db.QueryRow(query,
		log.BatchID, log.ChainType, nullString(log.ChainTxID),
		log.Status, nullString(log.ErrorMessage),
	).Scan(&log.ID, &log.CreatedAt)
}

// GetSubmitLogsByBatch returns all submission logs for a batch.
func (db *DB) GetSubmitLogsByBatch(batchID string) ([]chain.ChainSubmitLog, error) {
	query := `SELECT id, batch_id, chain_type, COALESCE(chain_tx_id, ''), status,
		COALESCE(error_message, ''), created_at
		FROM chain_submit_logs WHERE batch_id = $1 ORDER BY created_at DESC`
	rows, err := db.Query(query, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []chain.ChainSubmitLog
	for rows.Next() {
		var l chain.ChainSubmitLog
		if err := rows.Scan(&l.ID, &l.BatchID, &l.ChainType, &l.ChainTxID,
			&l.Status, &l.ErrorMessage, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// ──────────────────────────────────────────────
// Retry Queue (§14.3)
// ──────────────────────────────────────────────

// EnqueueRetry adds a batch to the retry queue.
func (db *DB) EnqueueRetry(batchID string, maxRetries int, nextRetry time.Time, lastError string) error {
	query := `INSERT INTO chain_retry_queue (batch_id, retry_count, max_retries, next_retry_at, last_error, status)
		VALUES ($1, 0, $2, $3, $4, 'PENDING')
		ON CONFLICT (batch_id) DO UPDATE SET
			retry_count = chain_retry_queue.retry_count + 1,
			next_retry_at = $3,
			last_error = $4,
			updated_at = now()`
	_, err := db.Exec(query, batchID, maxRetries, nextRetry, nullString(lastError))
	return err
}

// GetRetryableBatches returns batches due for retry.
func (db *DB) GetRetryableBatches(limit int) ([]chain.ChainRetryEntry, error) {
	query := `SELECT id, batch_id, retry_count, max_retries, next_retry_at,
		COALESCE(last_error, ''), status, created_at, updated_at
		FROM chain_retry_queue
		WHERE status = 'PENDING' AND next_retry_at <= now() AND retry_count < max_retries
		ORDER BY next_retry_at ASC LIMIT $1`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []chain.ChainRetryEntry
	for rows.Next() {
		var e chain.ChainRetryEntry
		if err := rows.Scan(&e.ID, &e.BatchID, &e.RetryCount, &e.MaxRetries,
			&e.NextRetryAt, &e.LastError, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// MarkRetryComplete marks a retry entry as complete.
func (db *DB) MarkRetryComplete(batchID string) error {
	query := `UPDATE chain_retry_queue SET status = 'COMPLETED', updated_at = now() WHERE batch_id = $1`
	_, err := db.Exec(query, batchID)
	return err
}

// MarkRetryFailed updates retry entry with error and next retry time (exponential backoff).
func (db *DB) MarkRetryFailed(batchID, lastError string, nextRetry time.Time) error {
	query := `UPDATE chain_retry_queue SET
		last_error = $1, next_retry_at = $2, updated_at = now()
		WHERE batch_id = $3`
	_, err := db.Exec(query, lastError, nextRetry, batchID)
	return err
}

// GetRetryQueueDepth returns the number of pending retries.
func (db *DB) GetRetryQueueDepth() (int64, error) {
	var count int64
	err := db.QueryRow(`SELECT COUNT(*) FROM chain_retry_queue WHERE status = 'PENDING'`).Scan(&count)
	return count, err
}

// Ensure pq is imported for nullable types
var _ = pq.Array
