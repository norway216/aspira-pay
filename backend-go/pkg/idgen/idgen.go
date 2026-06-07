// Package idgen provides ID generation for all Aspira Pay entities.
// IDs follow the pattern: {prefix}_{YYYYMMDD}_{random}
package idgen

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	rng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	rngMu sync.Mutex
)

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomStr(n int) string {
	rngMu.Lock()
	defer rngMu.Unlock()
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}

// UserID generates a user ID: u_{random12}
func UserID() string {
	return fmt.Sprintf("u_%s", randomStr(12))
}

// AccountID generates an account ID: acc_{random14}
func AccountID() string {
	return fmt.Sprintf("acc_%s", randomStr(14))
}

// PaymentID generates a payment ID: pay_{YYYYMMDD}_{random8}
func PaymentID() string {
	return fmt.Sprintf("pay_%s_%s", time.Now().Format("20060102"), randomStr(8))
}

// RequestID generates an idempotency request ID: req_{YYYYMMDD}_{random12}
func RequestID() string {
	return fmt.Sprintf("req_%s_%s", time.Now().Format("20060102"), randomStr(12))
}

// EventID generates an event ID: evt_{timestamp}_{random8}
func EventID() string {
	return fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), randomStr(8))
}

// QuoteID generates an FX quote ID: q_{YYYYMMDD}_{random6}
func QuoteID() string {
	return fmt.Sprintf("q_%s_%s", time.Now().Format("20060102"), randomStr(6))
}

// EntryID generates a ledger entry ID: lentry_{paymentID}_{seq}
func EntryID(paymentID string, seq int) string {
	return fmt.Sprintf("lentry_%s_%04d", paymentID, seq)
}

// BatchID generates a settlement batch ID: batch_{YYYYMMDD}_{random6}
func BatchID() string {
	return fmt.Sprintf("batch_%s_%s", time.Now().Format("20060102"), randomStr(6))
}

// ChainTxID generates a chain transaction ID: chtx_{random16}
func ChainTxID() string {
	return fmt.Sprintf("chtx_%s", randomStr(16))
}

// SequenceID generates a monotonically increasing sequence ID.
// For C++ engine command ordering.
type Sequencer struct {
	mu  sync.Mutex
	seq uint64
}

func NewSequencer(start uint64) *Sequencer {
	return &Sequencer{seq: start}
}

func (s *Sequencer) Next() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return s.seq
}

func (s *Sequencer) Current() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seq
}
