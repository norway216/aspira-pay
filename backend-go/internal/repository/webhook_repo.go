package repository

import (
	"time"
)

type WebhookRecord struct {
	ID         int64      `json:"-"`
	WebhookID  string     `json:"webhook_id"`
	MerchantID string     `json:"merchant_id"`
	URL        string     `json:"url"`
	Events     string     `json:"events"`
	Secret     string     `json:"secret"`
	Status     string     `json:"status"`
	RetryCount int        `json:"retry_count"`
	LastSent   *time.Time `json:"last_sent,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type WebhookDeliveryRecord struct {
	ID           int64  `json:"-"`
	WebhookID    string `json:"webhook_id"`
	EventID      string `json:"event_id"`
	EventType    string `json:"event_type"`
	Payload      string `json:"payload"`
	StatusCode   int    `json:"status_code"`
	ResponseBody string `json:"response_body,omitempty"`
	Success      bool   `json:"success"`
}

func (db *DB) CreateWebhook(w *WebhookRecord) error {
	return db.QueryRow(`INSERT INTO webhooks (webhook_id, merchant_id, url, events, secret, status)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, created_at, updated_at`,
		w.WebhookID, w.MerchantID, w.URL, w.Events, w.Secret, w.Status,
	).Scan(&w.ID, &w.CreatedAt, &w.UpdatedAt)
}

func (db *DB) ListActiveWebhooks(merchantID, eventType string) ([]*WebhookRecord, error) {
	query := `SELECT id, webhook_id, merchant_id, url, events, secret, status, retry_count, last_sent, COALESCE(last_error,''), created_at, updated_at
		FROM webhooks WHERE status='ACTIVE' AND merchant_id=$1 AND events LIKE '%' || $2 || '%'`
	rows, err := db.Query(query, merchantID, eventType)
	if err != nil { return nil, err }
	defer rows.Close()
	var result []*WebhookRecord
	for rows.Next() {
		w := &WebhookRecord{}
		if err := rows.Scan(&w.ID, &w.WebhookID, &w.MerchantID, &w.URL, &w.Events, &w.Secret,
			&w.Status, &w.RetryCount, &w.LastSent, &w.LastError, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	return result, nil
}

func (db *DB) RecordWebhookDelivery(d *WebhookDeliveryRecord) error {
	_, err := db.Exec(`INSERT INTO webhook_deliveries (webhook_id, event_id, event_type, payload, status_code, response_body, success)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, d.WebhookID, d.EventID, d.EventType, d.Payload, d.StatusCode, d.ResponseBody, d.Success)
	return err
}

func (db *DB) MarkWebhookSent(webhookID string) {
	db.Exec(`UPDATE webhooks SET last_sent=now(), retry_count=0, last_error=NULL, updated_at=now() WHERE webhook_id=$1`, webhookID)
}

func (db *DB) IncrementWebhookRetry(webhookID, lastError string) {
	db.Exec(`UPDATE webhooks SET retry_count=retry_count+1, last_error=$1, updated_at=now() WHERE webhook_id=$2`, lastError, webhookID)
}

func (db *DB) RecordNotification(userID, paymentID, status string) {
	db.Exec(`INSERT INTO notifications (user_id, payment_id, status) VALUES ($1,$2,$3)`, userID, paymentID, status)
}

// ── Outbox Dead Letter (§7.2) ──────────────────

func (db *DB) MarkOutboxDead(eventID, lastError string) {
	db.Exec(`UPDATE outbox_events SET dead_letter=true, last_error=$1, updated_at=now() WHERE event_id=$2`, lastError, eventID)
}

func (db *DB) FetchDeadLetterEvents(limit int) ([]OutboxEvent, error) {
	query := `SELECT id, event_id, aggregate_id, event_type, payload::text, published, published_at, created_at
		FROM outbox_events WHERE dead_letter=true AND retry_count < max_retries
		AND (next_retry_at IS NULL OR next_retry_at <= now())
		ORDER BY created_at ASC LIMIT $1`
	rows, err := db.Query(query, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	var events []OutboxEvent
	for rows.Next() {
		var e OutboxEvent
		if err := rows.Scan(&e.ID, &e.EventID, &e.AggregateID, &e.EventType, &e.Payload, &e.Published, &e.PublishedAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (db *DB) IncrementOutboxRetry(eventID, lastError string, nextRetry time.Time) {
	db.Exec(`UPDATE outbox_events SET retry_count=retry_count+1, last_error=$1, next_retry_at=$2 WHERE event_id=$3`, lastError, nextRetry, eventID)
}

func (db *DB) ReviveDeadLetter(eventID string) {
	db.Exec(`UPDATE outbox_events SET dead_letter=false, published=false, retry_count=0, next_retry_at=NULL, last_error=NULL WHERE event_id=$1`, eventID)
}
