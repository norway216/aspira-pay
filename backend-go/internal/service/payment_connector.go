// Package service defines the V3.0 Payment Connector interface (§5.8).
// Each payment channel (Bank, PSP, SWIFT, Card, Local Rail, Stablecoin)
// implements this interface for uniform payment execution.
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aspira/aspira-pay/pkg/idgen"
)

// PaymentRequest is the input for executing a payment through a channel.
type PaymentRequest struct {
	PaymentID      string
	OrderID        string
	SourceCurrency string
	TargetCurrency string
	SourceAmount   int64
	TargetAmount   int64
	FeeAmount      int64
	ReceiverInfo   map[string]string
}

// PaymentResponse is the result of a payment execution.
type PaymentResponse struct {
	ChannelTxID string
	Status      string
	ReceiptHash string
	ExecutedAt  time.Time
}

// RefundRequest is the input for processing a refund.
type RefundRequest struct {
	RefundID       string
	OriginalPaymentID string
	Amount         int64
	Currency       string
	Reason         string
}

// RefundResponse is the result of a refund.
type RefundResponse struct {
	ChannelRefundID string
	Status          string
	ReceiptHash     string
	ExecutedAt      time.Time
}

// PaymentConnector is the unified interface for all payment channels (§5.8.3).
type PaymentConnector interface {
	CreatePayment(ctx context.Context, req PaymentRequest) (PaymentResponse, error)
	QueryPayment(ctx context.Context, paymentID string) (PaymentResponse, error)
	Refund(ctx context.Context, req RefundRequest) (RefundResponse, error)
	QueryRefund(ctx context.Context, refundID string) (RefundResponse, error)
	HealthCheck(ctx context.Context) error
	ChannelName() string
}

// SimulatedChannel implements PaymentConnector for Sandbox testing.
// It simulates a bank/PSP channel without real money movement.
type SimulatedChannel struct {
	name       string
	successRate float64 // 0.0-1.0, probability of successful execution
}

func NewSimulatedChannel(name string, successRate float64) *SimulatedChannel {
	return &SimulatedChannel{name: name, successRate: successRate}
}

func (s *SimulatedChannel) ChannelName() string { return s.name }

func (s *SimulatedChannel) CreatePayment(ctx context.Context, req PaymentRequest) (PaymentResponse, error) {
	log.Printf("[Channel:%s] Executing payment %s: %d %s → %d %s",
		s.name, req.PaymentID, req.SourceAmount, req.SourceCurrency,
		req.TargetAmount, req.TargetCurrency)

	// Simulate processing delay
	select {
	case <-ctx.Done():
		return PaymentResponse{}, ctx.Err()
	case <-time.After(50 * time.Millisecond):
	}

	return PaymentResponse{
		ChannelTxID: fmt.Sprintf("ch_%s_%s", s.name, idgen.PaymentID()),
		Status:      "CONFIRMED",
		ReceiptHash: idgen.RequestID(),
		ExecutedAt:  time.Now(),
	}, nil
}

func (s *SimulatedChannel) QueryPayment(ctx context.Context, paymentID string) (PaymentResponse, error) {
	return PaymentResponse{
		ChannelTxID: fmt.Sprintf("ch_%s_%s", s.name, paymentID),
		Status:      "CONFIRMED",
	}, nil
}

func (s *SimulatedChannel) Refund(ctx context.Context, req RefundRequest) (RefundResponse, error) {
	log.Printf("[Channel:%s] Processing refund %s for payment %s",
		s.name, req.RefundID, req.OriginalPaymentID)

	select {
	case <-ctx.Done():
		return RefundResponse{}, ctx.Err()
	case <-time.After(30 * time.Millisecond):
	}

	return RefundResponse{
		ChannelRefundID: fmt.Sprintf("ref_%s_%s", s.name, idgen.PaymentID()),
		Status:          "REFUNDED",
		ReceiptHash:     idgen.RequestID(),
		ExecutedAt:      time.Now(),
	}, nil
}

func (s *SimulatedChannel) QueryRefund(ctx context.Context, refundID string) (RefundResponse, error) {
	return RefundResponse{Status: "REFUNDED"}, nil
}

func (s *SimulatedChannel) HealthCheck(ctx context.Context) error {
	return nil // Always healthy in sandbox
}

// Ensure SimulatedChannel implements PaymentConnector
var _ PaymentConnector = (*SimulatedChannel)(nil)
