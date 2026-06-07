package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// OutboxEvent represents an event to be published via the outbox pattern.
type OutboxEvent struct {
	ID          int64     `json:"-"`
	EventID     string    `json:"event_id"`
	AggregateID string    `json:"aggregate_id"`
	EventType   string    `json:"event_type"`
	Payload     string    `json:"payload"`
	Published   bool      `json:"published"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// InsertOutboxEvent inserts an outbox event (must be in same tx as the business operation).
func (db *DB) InsertOutboxEvent(eventID, aggregateID, eventType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cannot marshal outbox payload: %w", err)
	}

	query := `INSERT INTO outbox_events (event_id, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4)`
	_, err = db.Exec(query, eventID, aggregateID, eventType, string(payloadJSON))
	return err
}

// InsertOutboxEventTx inserts an outbox event within a transaction.
func (db *DB) InsertOutboxEventTx(tx *sql.Tx, eventID, aggregateID, eventType string, payload interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cannot marshal outbox payload: %w", err)
	}

	query := `INSERT INTO outbox_events (event_id, aggregate_id, event_type, payload)
		VALUES ($1, $2, $3, $4)`
	_, err = tx.Exec(query, eventID, aggregateID, eventType, string(payloadJSON))
	return err
}

// FetchUnpublishedEvents retrieves unpublished events ordered by creation time.
func (db *DB) FetchUnpublishedEvents(limit int) ([]OutboxEvent, error) {
	query := `SELECT id, event_id, aggregate_id, event_type, payload::text, published, published_at, created_at
		FROM outbox_events WHERE published = false ORDER BY created_at ASC LIMIT $1`
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(
			&e.ID, &e.EventID, &e.AggregateID, &e.EventType,
			&e.Payload, &e.Published, &e.PublishedAt, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// MarkOutboxPublished marks an outbox event as published.
func (db *DB) MarkOutboxPublished(eventID string) error {
	query := `UPDATE outbox_events SET published = true, published_at = now() WHERE event_id = $1`
	_, err := db.Exec(query, eventID)
	return err
}
