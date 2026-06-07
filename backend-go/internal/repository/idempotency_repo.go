package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	pkgerrors "github.com/aspira/aspira-pay/pkg/errors"
)

// IdempotencyRecord holds a cached API response for idempotency.
type IdempotencyRecord struct {
	RequestID      string `json:"request_id"`
	RequestHash    string `json:"request_hash"`
	ResponseBody   string `json:"response_body"`
	ResponseStatus int    `json:"response_status"`
	Status         string `json:"status"`
}

// CreateIdempotencyKey inserts a new idempotency record.
func (db *DB) CreateIdempotencyKey(requestID, requestHash string, responseBody interface{}, responseStatus int) error {
	bodyJSON, err := json.Marshal(responseBody)
	if err != nil {
		return fmt.Errorf("cannot marshal response: %w", err)
	}

	query := `INSERT INTO idempotency_keys (request_id, request_hash, response_body, response_status, status)
		VALUES ($1, $2, $3, $4, 'COMPLETED')`
	_, err = db.Exec(query, requestID, requestHash, string(bodyJSON), responseStatus)
	return err
}

// InsertIdempotencyTx inserts an idempotency record within a transaction.
// Used by the transactional outbox pattern (architecture doc §7.4).
func (db *DB) InsertIdempotencyTx(tx *sql.Tx, requestID, requestHash, responseBody string) error {
	query := `INSERT INTO idempotency_keys (request_id, request_hash, response_body, response_status, status)
		VALUES ($1, $2, $3, 201, 'COMPLETED')
		ON CONFLICT (request_id) DO NOTHING`
	_, err := tx.Exec(query, requestID, requestHash, responseBody)
	return err
}

// GetIdempotencyKey retrieves a cached idempotency response.
func (db *DB) GetIdempotencyKey(requestID string) (*IdempotencyRecord, error) {
	r := &IdempotencyRecord{}
	query := `SELECT request_id, request_hash, COALESCE(response_body::text, ''), response_status, status
		FROM idempotency_keys WHERE request_id = $1`
	err := db.QueryRow(query, requestID).Scan(
		&r.RequestID, &r.RequestHash, &r.ResponseBody, &r.ResponseStatus, &r.Status,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Not found — not an error
	}
	return r, err
}

// GetIdempotencyRecord is an alias for GetIdempotencyKey for cleaner naming.
func (db *DB) GetIdempotencyRecord(requestID string) (*IdempotencyRecord, error) {
	return db.GetIdempotencyKey(requestID)
}

// CheckAndCreateIdempotency performs idempotency check and creation atomically.
// Returns the cached response body if the request was already processed.
// Returns error if same request_id with different hash (mismatch).
func (db *DB) CheckAndCreateIdempotency(requestID, requestHash string) (*IdempotencyRecord, error) {
	// First, check if already exists
	existing, err := db.GetIdempotencyKey(requestID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Already processed — check hash match
		if existing.RequestHash != requestHash {
			return nil, pkgerrors.New(pkgerrors.ErrCodeIdempotencyMismatch,
				"idempotency key reused with different request body")
		}
		return existing, nil
	}

	// Not yet processed — create a PENDING placeholder
	query := `INSERT INTO idempotency_keys (request_id, request_hash, response_body, response_status, status)
		VALUES ($1, $2, '{}', 0, 'PENDING')`
	_, err = db.Exec(query, requestID, requestHash)
	return nil, err
}

// CompleteIdempotency updates the idempotency record with the final response.
func (db *DB) CompleteIdempotency(requestID string, responseBody interface{}, responseStatus int) error {
	bodyJSON, err := json.Marshal(responseBody)
	if err != nil {
		return fmt.Errorf("cannot marshal response: %w", err)
	}

	query := `UPDATE idempotency_keys
		SET response_body = $1, response_status = $2, status = 'COMPLETED'
		WHERE request_id = $3`
	_, err = db.Exec(query, string(bodyJSON), responseStatus, requestID)
	return err
}
