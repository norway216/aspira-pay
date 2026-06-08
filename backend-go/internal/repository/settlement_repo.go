package repository

import (
	"database/sql"
	"fmt"

	"github.com/aspira/aspira-pay/internal/domain/settlement"
)

// CreateSettlementBatch inserts a new settlement batch.
func (db *DB) CreateSettlementBatch(b *settlement.Batch) error {
	query := `
		INSERT INTO settlement_batches (batch_id, currency, total_debit, total_credit, entry_count, status, ledger_root_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`
	return db.QueryRow(query,
		b.BatchID, b.Currency, b.TotalDebit, b.TotalCredit,
		b.EntryCount, b.Status, b.LedgerRootHash,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

// GetSettlementBatch retrieves a settlement batch by batch_id.
func (db *DB) GetSettlementBatch(batchID string) (*settlement.Batch, error) {
	b := &settlement.Batch{}
	query := `SELECT id, batch_id, currency, total_debit, total_credit, entry_count, status,
		COALESCE(ledger_root_hash, ''), COALESCE(chain_tx_id, ''), created_at, updated_at
		FROM settlement_batches WHERE batch_id = $1`
	err := db.QueryRow(query, batchID).Scan(
		&b.ID, &b.BatchID, &b.Currency, &b.TotalDebit, &b.TotalCredit,
		&b.EntryCount, &b.Status, &b.LedgerRootHash, &b.ChainTxID,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settlement batch not found: %s", batchID)
	}
	return b, err
}

// UpdateSettlementBatch updates a settlement batch.
func (db *DB) UpdateSettlementBatch(batchID string, totalDebit, totalCredit int64, entryCount int) error {
	query := `UPDATE settlement_batches
		SET total_debit = $1, total_credit = $2, entry_count = $3, updated_at = now()
		WHERE batch_id = $4`
	_, err := db.Exec(query, totalDebit, totalCredit, entryCount, batchID)
	return err
}

// UpdateSettlementBatchStatus updates the batch status.
func (db *DB) UpdateSettlementBatchStatus(batchID, status string) error {
	query := `UPDATE settlement_batches SET status = $1, updated_at = now() WHERE batch_id = $2`
	_, err := db.Exec(query, status, batchID)
	return err
}

// UpdateSettlementBatchChainTx updates the chain tx for a batch.
func (db *DB) UpdateSettlementBatchChainTx(batchID, chainTxID string) error {
	query := `UPDATE settlement_batches SET chain_tx_id = $1, updated_at = now() WHERE batch_id = $2`
	_, err := db.Exec(query, chainTxID, batchID)
	return err
}

// ListSettlementBatches returns paginated settlement batches.
func (db *DB) ListSettlementBatches(page, pageSize int) ([]settlement.Batch, int64, error) {
	var total int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM settlement_batches`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	query := `SELECT id, batch_id, currency, total_debit, total_credit, entry_count, status,
		COALESCE(ledger_root_hash, ''), COALESCE(chain_tx_id, ''), created_at, updated_at
		FROM settlement_batches ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var batches []settlement.Batch
	for rows.Next() {
		var b settlement.Batch
		if err := rows.Scan(
			&b.ID, &b.BatchID, &b.Currency, &b.TotalDebit, &b.TotalCredit,
			&b.EntryCount, &b.Status, &b.LedgerRootHash, &b.ChainTxID,
			&b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		batches = append(batches, b)
	}
	return batches, total, nil
}

// CountLedgerEntriesByBatch returns the number of ledger entries for a batch.
func (db *DB) CountLedgerEntriesByBatch(batchID string) (int64, error) {
	var count int64
	err := db.QueryRow(`SELECT COUNT(*) FROM ledger_entries WHERE payment_id IN
		(SELECT payment_id FROM settlement_details WHERE batch_id = $1)`, batchID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountChannelReceiptsByBatch returns the number of channel receipts for a batch.
func (db *DB) CountChannelReceiptsByBatch(batchID string) (int64, error) {
	var count int64
	err := db.QueryRow(`SELECT COUNT(*) FROM channel_receipts WHERE order_id IN
		(SELECT order_id FROM settlement_details WHERE batch_id = $1)`, batchID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// CountChainProofsByBatch returns the number of chain proofs for a batch.
func (db *DB) CountChainProofsByBatch(batchID string) (int64, error) {
	var count int64
	err := db.QueryRow(`SELECT COUNT(*) FROM chain_events WHERE payment_id IN
		(SELECT payment_id FROM settlement_details WHERE batch_id = $1)`, batchID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetOpenSettlementBatch gets or creates an open batch for a currency.
func (db *DB) GetOpenSettlementBatch(currency string) (*settlement.Batch, error) {
	b := &settlement.Batch{}
	query := `SELECT id, batch_id, currency, total_debit, total_credit, entry_count, status,
		COALESCE(ledger_root_hash, ''), COALESCE(chain_tx_id, ''), created_at, updated_at
		FROM settlement_batches WHERE currency = $1 AND status = 'OPEN' ORDER BY created_at DESC LIMIT 1`
	err := db.QueryRow(query, currency).Scan(
		&b.ID, &b.BatchID, &b.Currency, &b.TotalDebit, &b.TotalCredit,
		&b.EntryCount, &b.Status, &b.LedgerRootHash, &b.ChainTxID,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No open batch — caller should create one
	}
	return b, err
}
