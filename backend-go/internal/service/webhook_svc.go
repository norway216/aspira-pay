// Package service implements V3 Webhook & Notification (§5.13.4, §7.1).
// Webhook: merchant-registered callbacks for payment events.
// Notification: email/SMS/in-app messages for payment status updates.
package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// WebhookService manages merchant webhook registration and delivery.
type WebhookService struct {
	db     *repository.DB
	client *http.Client
}

func NewWebhookService(db *repository.DB) *WebhookService {
	return &WebhookService{
		db:     db,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// RegisterWebhook creates a new webhook for a merchant.
func (s *WebhookService) RegisterWebhook(merchantID, url, events string) (*repository.WebhookRecord, error) {
	secret := idgen.ChainTxID() // Random secret for HMAC signing
	w := &repository.WebhookRecord{
		WebhookID:  "wh_" + idgen.CardID(),
		MerchantID: merchantID,
		URL:        url,
		Events:     events,
		Secret:     secret,
		Status:     "ACTIVE",
	}
	if err := s.db.CreateWebhook(w); err != nil {
		return nil, fmt.Errorf("cannot create webhook: %w", err)
	}
	log.Printf("Webhook: registered %s for merchant %s → %s", w.WebhookID, merchantID, url)
	return w, nil
}

// DeliverEvent sends a payment event to all registered webhooks.
// Architecture doc §7.1: webhook.dispatch.requested → webhook.dispatch.completed
func (s *WebhookService) DeliverEvent(eventType, paymentID string, payload map[string]interface{}) {
	merchantID, _ := payload["merchant_id"].(string)
	if merchantID == "" {
		return // Only deliver events with a merchant context
	}

	webhooks, _ := s.db.ListActiveWebhooks(merchantID, eventType)
	for _, wh := range webhooks {
		go s.deliverWebhook(wh, eventType, paymentID, payload)
	}
}

func (s *WebhookService) deliverWebhook(wh *repository.WebhookRecord, eventType, paymentID string, payload map[string]interface{}) {
	body, _ := json.Marshal(map[string]interface{}{
		"event_id":   idgen.EventID(),
		"event_type": eventType,
		"payment_id": paymentID,
		"payload":    payload,
		"timestamp":  time.Now().Unix(),
	})

	// HMAC signature for webhook verification (§9.1)
	sig := computeHMAC(string(body), wh.Secret)

	req, _ := http.NewRequest("POST", wh.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Aspira-Signature", sig)
	req.Header.Set("X-Aspira-Event", eventType)

	resp, err := s.client.Do(req)

	delivery := &repository.WebhookDeliveryRecord{
		WebhookID: wh.WebhookID,
		EventID:   paymentID,
		EventType: eventType,
		Payload:   string(body),
		Success:   err == nil && resp != nil && resp.StatusCode >= 200 && resp.StatusCode < 300,
	}
	if resp != nil {
		delivery.StatusCode = resp.StatusCode
		resp.Body.Close()
	}
	if err != nil {
		delivery.ResponseBody = err.Error()
	}

	s.db.RecordWebhookDelivery(delivery)

	if delivery.Success {
		log.Printf("Webhook: delivered %s to %s (HTTP %d)", eventType, wh.URL, delivery.StatusCode)
		s.db.MarkWebhookSent(wh.WebhookID)
	} else {
		log.Printf("Webhook: FAILED %s to %s: %v", eventType, wh.URL, err)
		s.db.IncrementWebhookRetry(wh.WebhookID, delivery.ResponseBody)
	}
}

func computeHMAC(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// NotificationService sends user-facing notifications.
type NotificationService struct {
	db *repository.DB
}

func NewNotificationService(db *repository.DB) *NotificationService {
	return &NotificationService{db: db}
}

// NotifyPaymentStatus logs a notification for a payment status change.
// Production: sends email/SMS/push via external providers.
func (s *NotificationService) NotifyPaymentStatus(userID, paymentID, status string) {
	log.Printf("[Notification] User %s: payment %s → %s", userID, paymentID, status)
	// In production: look up user email/phone, send via SES/Twilio/FCM
	s.db.RecordNotification(userID, paymentID, status)
}
