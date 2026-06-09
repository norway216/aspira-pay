// Package merchant defines V3 Merchant domain (§3.2, §5.1.4).
// API keys for programmatic access, webhook configuration.
package merchant

import "time"

// Merchant represents a business customer with API access.
type Merchant struct {
	ID           int64     `json:"-"`
	MerchantID   string    `json:"merchant_id"`
	UserID       string    `json:"user_id"`
	BusinessName string    `json:"business_name"`
	BusinessType string    `json:"business_type"`
	Country      string    `json:"country"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// APIKey is a merchant's API credential for programmatic access (§5.1.4).
type APIKey struct {
	ID         int64     `json:"-"`
	KeyID      string    `json:"key_id"`
	MerchantID string    `json:"merchant_id"`
	APIKey     string    `json:"api_key"`      // Only returned on creation
	KeyPrefix  string    `json:"key_prefix"`   // First 8 chars for display
	Scopes     string    `json:"scopes"`       // "read,write,admin"
	Status     string    `json:"status"`
	LastUsed   *time.Time `json:"last_used,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Webhook is a merchant's registered callback URL for event notifications.
type Webhook struct {
	ID          int64     `json:"-"`
	WebhookID   string    `json:"webhook_id"`
	MerchantID  string    `json:"merchant_id"`
	URL         string    `json:"url"`
	Events      string    `json:"events"`       // "payment.completed,payment.failed"
	Secret      string    `json:"secret"`       // HMAC signing secret
	Status      string    `json:"status"`
	RetryCount  int       `json:"retry_count"`
	LastSent    *time.Time `json:"last_sent,omitempty"`
	LastError   string    `json:"last_error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WebhookDelivery records each webhook delivery attempt.
type WebhookDelivery struct {
	ID           int64     `json:"-"`
	WebhookID    string    `json:"webhook_id"`
	EventID      string    `json:"event_id"`
	EventType    string    `json:"event_type"`
	Payload      string    `json:"payload"`
	StatusCode   int       `json:"status_code"`
	ResponseBody string    `json:"response_body,omitempty"`
	Success      bool      `json:"success"`
	CreatedAt    time.Time `json:"created_at"`
}
