package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/chain"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/crypto"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// ChainService manages the blockchain audit layer (Hash Chain implementation).
// It records payment events on an append-only hash chain for tamper-proof auditing.
// Architecture doc §11.1: Internal hash chain with batch Merkle proofs (11.2).
type ChainService struct {
	db *repository.DB

	// Batching state (architecture doc §10.4: batch_size=100, batch_interval_sec=30)
	mu            sync.Mutex
	pendingEvents []chain.ChainEvent
	batchSize     int
	batchInterval time.Duration
	stopCh        chan struct{}
}

// NewChainService creates a new ChainService with batching support.
func NewChainService(db *repository.DB) *ChainService {
	return &ChainService{
		db:            db,
		pendingEvents: make([]chain.ChainEvent, 0),
		batchSize:     100,  // Architecture doc §10.4
		batchInterval: 30 * time.Second,
		stopCh:        make(chan struct{}),
	}
}

// StartBatching begins the batch processing loop.
// Architecture doc §11.2: Batch events and submit Merkle root.
func (s *ChainService) StartBatching() {
	go s.batchLoop()
	log.Printf("ChainService: batch processing started (size=%d, interval=%v)", s.batchSize, s.batchInterval)
}

// StopBatching stops the batch processing loop.
func (s *ChainService) StopBatching() {
	close(s.stopCh)
	// Flush remaining events
	s.flushBatch()
}

func (s *ChainService) batchLoop() {
	ticker := time.NewTicker(s.batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.flushBatch()
		}
	}
}

// flushBatch creates a new block from all pending events.
func (s *ChainService) flushBatch() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.pendingEvents) == 0 {
		return
	}

	events := s.pendingEvents
	s.pendingEvents = make([]chain.ChainEvent, 0)

	if err := s.createBlock(events); err != nil {
		log.Printf("ChainService: batch flush failed: %v", err)
		// Re-add events to pending for retry
		s.mu.Lock()
		s.pendingEvents = append(events, s.pendingEvents...)
		s.mu.Unlock()
		return
	}

	log.Printf("ChainService: batch flushed — %d events confirmed in block", len(events))
}

// createBlock creates a new hash chain block from a batch of events.
// Architecture doc §11.1:
//   event_hash = sha256(event_payload)
//   merkle_root = merkle(event_hash_1, event_hash_2, ...)
//   block_hash = sha256(prev_hash + merkle_root + timestamp)
func (s *ChainService) createBlock(events []chain.ChainEvent) error {
	latestBlock, err := s.db.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("no chain blocks: %w", err)
	}

	// Collect event hashes for Merkle tree
	hashes := make([]string, len(events))
	for i, e := range events {
		hashes[i] = e.PayloadHash
	}

	// Compute Merkle root of all events in batch (architecture doc §11.2)
	merkleRoot := crypto.MerkleRoot(hashes)

	// Compute block hash
	newHeight := latestBlock.BlockHeight + 1
	blockHash := chainHashBlockBatch(newHeight, latestBlock.BlockHash, merkleRoot, len(events))

	// Create block
	block := &chain.ChainBlock{
		BlockHeight: newHeight,
		BlockHash:   blockHash,
		PrevHash:    latestBlock.BlockHash,
		MerkleRoot:  merkleRoot,
		EventCount:  len(events),
	}

	if err := s.db.InsertChainBlock(block); err != nil {
		return fmt.Errorf("cannot insert chain block: %w", err)
	}

	// Update all events with block hash and insert
	for i := range events {
		events[i].BlockHeight = newHeight
		if err := s.db.InsertChainEvent(&events[i]); err != nil {
			return fmt.Errorf("cannot insert chain event: %w", err)
		}
	}

	return nil
}

// RecordPaymentOnChain records a payment event on the hash chain.
// Events are batched — not written immediately (architecture doc §11.2).
func (s *ChainService) RecordPaymentOnChain(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return fmt.Errorf("payment not found: %w", err)
	}

	// Generate payload hash (only hashes go on chain, never plaintext)
	payloadHash := s.computePaymentPayloadHash(
		order.PaymentID, order.SenderUserID, order.ReceiverUserID,
		order.SourceAmount, order.SourceCurrency, order.TargetCurrency,
		string(order.Status))

	// Enqueue event for batching (non-blocking)
	s.enqueueEvent(paymentID, chain.EventSettlementCompleted, payloadHash)

	// Also enqueue completion event
	completionHash := crypto.SHA256(
		fmt.Sprintf("completed:%s:%d", paymentID, time.Now().UnixNano()))
	s.enqueueEvent(paymentID, chain.EventPaymentCompleted, completionHash)

	log.Printf("Chain: payment %s enqueued for batch processing", paymentID)
	return nil
}

// enqueueEvent adds an event to the pending batch.
// If the batch reaches capacity, it triggers an immediate flush.
func (s *ChainService) enqueueEvent(paymentID, eventType, payloadHash string) {
	event := chain.ChainEvent{
		EventID:     idgen.EventID(),
		PaymentID:   paymentID,
		EventType:   eventType,
		PayloadHash: payloadHash,
	}

	s.mu.Lock()
	s.pendingEvents = append(s.pendingEvents, event)
	shouldFlush := len(s.pendingEvents) >= s.batchSize
	s.mu.Unlock()

	if shouldFlush {
		s.flushBatch()
	}
}

// GetAuditTrail retrieves the complete chain audit trail for a payment.
func (s *ChainService) GetAuditTrail(paymentID string) (*chain.AuditTrail, error) {
	events, err := s.db.GetChainEventsByPayment(paymentID)
	if err != nil {
		return nil, err
	}

	var blocks []chain.ChainBlock
	seenBlocks := make(map[int64]bool)

	for _, e := range events {
		if !seenBlocks[e.BlockHeight] {
			block, err := s.db.GetChainBlock(e.BlockHeight)
			if err == nil {
				blocks = append(blocks, *block)
				seenBlocks[e.BlockHeight] = true
			}
		}
	}

	trail := &chain.AuditTrail{
		PaymentID: paymentID,
		Blocks:    blocks,
		Events:    events,
		Verified:  s.verifyAuditTrail(blocks),
	}

	return trail, nil
}

// verifyAuditTrail verifies the hash chain integrity.
func (s *ChainService) verifyAuditTrail(blocks []chain.ChainBlock) bool {
	for i := 1; i < len(blocks); i++ {
		if blocks[i].PrevHash != blocks[i-1].BlockHash {
			log.Printf("Chain verification failed: block %d prev_hash mismatch", blocks[i].BlockHeight)
			return false
		}
	}
	return true
}

// GetLatestBlock returns the most recent chain block.
func (s *ChainService) GetLatestBlock() (*chain.ChainBlock, error) {
	return s.db.GetLatestBlock()
}

// GetBlock returns a block by height.
func (s *ChainService) GetBlock(height int64) (*chain.ChainBlock, error) {
	return s.db.GetChainBlock(height)
}

// ListBlocks returns paginated chain blocks.
func (s *ChainService) ListBlocks(page, pageSize int) ([]chain.ChainBlock, int64, error) {
	return s.db.GetChainBlocks(page, pageSize)
}

// computePaymentPayloadHash generates a hash of payment data for on-chain recording.
// Only hashes are stored on chain, never sensitive plaintext data.
// Architecture doc §11.2.1: What Goes On-chain.
func (s *ChainService) computePaymentPayloadHash(paymentID, senderID, receiverID string, amount int64, sourceCurrency, targetCurrency, status string) string {
	data := fmt.Sprintf("%s:%s:%s:%d:%s:%s:%s:%d",
		paymentID,
		crypto.HashUserID(senderID),
		crypto.HashUserID(receiverID),
		amount,
		sourceCurrency,
		targetCurrency,
		status,
		time.Now().Unix(),
	)
	return crypto.SHA256(data)
}

// chainHashBlockBatch computes a hash chain block hash from batch data.
// Architecture doc §11.1:
//   block_hash = sha256(prev_hash + merkle_root + timestamp + event_count)
func chainHashBlockBatch(height int64, prevHash string, merkleRoot string, eventCount int) string {
	data := fmt.Sprintf("%d:%s:%s:%d:%d", height, prevHash, merkleRoot, eventCount, time.Now().Unix())
	return crypto.SHA256(data)
}
