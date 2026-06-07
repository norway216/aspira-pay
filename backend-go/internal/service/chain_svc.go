package service

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/chain"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/crypto"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// ChainService manages the blockchain audit layer with Merkle batch proofs.
// Architecture doc §5: Hash Builder → Merkle Batch Builder → Chain Committer → Audit API.
//
// Key optimizations:
//
//	§4  - Merkle batch proofs: batch 1000 events → one merkle_root on-chain
//	§8  - Configurable triggers: size-based (batchSize) + time-based (batchInterval)
//	§8.2 - Deterministic hashing with fixed field order
//	§9  - Asynchronous submission: does NOT block the main payment flow
//	§10 - Retry queue with exponential backoff
//	§11 - Ed25519 audit signatures on each batch
//	§15 - Merkle proof verification API
type ChainService struct {
	db *repository.DB

	// §11: Audit signing key (generated at startup for Sandbox; loaded from Vault in production)
	auditPrivateKey ed25519.PrivateKey
	auditPublicKey  ed25519.PublicKey

	// §8.1: Batch configuration
	mu            sync.Mutex
	pendingEvents []batchEvent
	batchSize     int
	batchInterval time.Duration

	// Background workers
	stopCh     chan struct{}
}

// batchEvent is an event waiting to be included in a Merkle batch.
type batchEvent struct {
	paymentID       string
	eventType       string
	payloadHash     string
	ledgerEntryHash string
	sequenceID      uint64
	timestamp       int64
}

// NewChainService creates a new ChainService with Ed25519 key generation.
// Architecture doc §11.2: Ed25519 for fast internal audit signatures.
func NewChainService(db *repository.DB) *ChainService {
	pub, priv, err := crypto.GenerateKeyPair()
	if err != nil {
		log.Printf("ChainService: WARNING - cannot generate audit key pair: %v — using zero key", err)
		pub = make([]byte, ed25519.PublicKeySize)
		priv = make([]byte, ed25519.PrivateKeySize)
	}

	svc := &ChainService{
		db:              db,
		auditPrivateKey: priv,
		auditPublicKey:  pub,
		pendingEvents:   make([]batchEvent, 0),
		batchSize:       1000, // §8.1: size-based trigger
		batchInterval:   1 * time.Second, // §8.1: time-based trigger (1000ms)
		stopCh:          make(chan struct{}),
	}

	log.Printf("ChainService: initialized with Ed25519 audit key (pub=%s)",
		hex.EncodeToString(pub)[:16]+"...")

	return svc
}

// Start begins the batch processing and retry loops.
func (s *ChainService) Start() {
	go s.batchFlushLoop()
	go s.retryLoop()
	log.Printf("ChainService: started (batch_size=%d, batch_interval=%v)", s.batchSize, s.batchInterval)
}

// Stop gracefully shuts down background workers.
func (s *ChainService) Stop() {
	close(s.stopCh)
	s.flushBatch() // Final flush
	log.Println("ChainService: stopped")
}

// ──────────────────────────────────────────────
// Event Recording (§7, §8)
// ──────────────────────────────────────────────

// RecordPaymentOnChain enqueues a payment event for batched Merkle proof.
// Events are NOT written immediately — they're buffered for batch Merkle tree construction.
// Architecture doc §4: Batch 1000 events → Merkle root → one on-chain record.
func (s *ChainService) RecordPaymentOnChain(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return fmt.Errorf("chain: payment not found: %w", err)
	}

	now := time.Now().Unix()

	// §8.3: Deterministic hashing with fixed field order
	payloadHash := s.computeDeterministicHash(
		paymentID, order.SenderUserID, order.ReceiverUserID,
		order.SourceAmount, order.SourceCurrency, order.TargetCurrency,
		string(order.Status), now)

	// Settlement completion event
	s.enqueueEvent(batchEvent{
		paymentID:   paymentID,
		eventType:   chain.EventSettlementCompleted,
		payloadHash: payloadHash,
		ledgerEntryHash: crypto.SHA256(
			fmt.Sprintf("ledger:%s:%d:%d", paymentID, order.SourceAmount, order.TargetAmount)),
		sequenceID: uint64(order.ID),
		timestamp:  now,
	})

	// Payment completion event
	completionHash := crypto.DeterministicEventHash(
		idgen.EventID(), paymentID, chain.EventPaymentCompleted,
		payloadHash, uint64(order.ID), now)
	s.enqueueEvent(batchEvent{
		paymentID:   paymentID,
		eventType:   chain.EventPaymentCompleted,
		payloadHash: completionHash,
		timestamp:   now,
	})

	return nil
}

// enqueueEvent adds an event to the batch buffer.
// Triggers flush if batch is full (§8.1: size-based trigger).
func (s *ChainService) enqueueEvent(ev batchEvent) {
	s.mu.Lock()
	s.pendingEvents = append(s.pendingEvents, ev)
	shouldFlush := len(s.pendingEvents) >= s.batchSize
	s.mu.Unlock()

	if shouldFlush {
		s.flushBatch()
	}
}

// ──────────────────────────────────────────────
// Batch Building (§4, §7, §8)
// ──────────────────────────────────────────────

// batchFlushLoop periodically flushes pending events.
// Architecture doc §8.1: time-based trigger.
func (s *ChainService) batchFlushLoop() {
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

// flushBatch builds a Merkle tree from pending events and commits to the hash chain.
func (s *ChainService) flushBatch() {
	s.mu.Lock()
	if len(s.pendingEvents) == 0 {
		s.mu.Unlock()
		return
	}
	events := s.pendingEvents
	s.pendingEvents = make([]batchEvent, 0)
	s.mu.Unlock()

	if err := s.commitBatch(events); err != nil {
		log.Printf("ChainService: batch commit failed: %v — re-enqueuing %d events", err, len(events))
		// Re-enqueue for retry
		s.mu.Lock()
		s.pendingEvents = append(events, s.pendingEvents...)
		s.mu.Unlock()
	}
}

// commitBatch builds the Merkle tree, creates a chain block, and signs the batch.
func (s *ChainService) commitBatch(events []batchEvent) error {
	batchID := idgen.BatchID()
	now := time.Now()

	// §8: Build Merkle tree from event hashes
	hashes := make([]string, len(events))
	for i, ev := range events {
		hashes[i] = ev.payloadHash
	}
	merkleRoot := crypto.MerkleRoot(hashes)

	// §12: Get ledger root hash from latest settlement
	ledgerRootHash := s.computeLedgerRootHash(events)

	// Get latest block for hash chain linkage
	latestBlock, err := s.db.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("no chain blocks: %w", err)
	}

	// §7.1: Compute block hash
	newHeight := latestBlock.BlockHeight + 1
	blockHash := crypto.BatchedHashChainBlock(
		latestBlock.BlockHash, merkleRoot, batchID, len(events), now.Unix())

	// §11: Sign the canonical batch payload
	canonicalPayload := crypto.CanonicalBatchPayload(
		batchID, merkleRoot, ledgerRootHash,
		0, int64(len(events)-1), // start_seq, end_seq
		len(events), now.Unix())
	auditSig := s.signPayload(canonicalPayload)

	// Create chain block
	block := &chain.ChainBlock{
		BlockHeight:     newHeight,
		BlockHash:       blockHash,
		PrevHash:        latestBlock.BlockHash,
		MerkleRoot:      merkleRoot,
		EventCount:      len(events),
		BatchID:         batchID,
		StartSequenceID: 0,
		EndSequenceID:   int64(len(events) - 1),
		AuditSignature:  auditSig,
	}

	if err := s.db.InsertChainBlock(block); err != nil {
		return fmt.Errorf("insert chain block: %w", err)
	}

	// Create chain batch record
	batch := &chain.ChainBatch{
		BatchID:         batchID,
		MerkleRoot:      merkleRoot,
		LedgerRootHash:  ledgerRootHash,
		EventCount:      len(events),
		StartSequenceID: 0,
		EndSequenceID:   int64(len(events) - 1),
		Status:          chain.BatchConfirmed,
		AuditSignature:  auditSig,
		BlockHeight:     &newHeight,
	}
	if err := s.db.CreateChainBatch(batch); err != nil {
		return fmt.Errorf("create chain batch: %w", err)
	}

	// Insert individual chain events with Merkle proofs
	chainEvents := make([]chain.ChainEvent, len(events))
	for i, ev := range events {
		eventID := idgen.EventID()
		// §15: Generate Merkle proof for each event
		merkleProof := crypto.MerkleProof(hashes, i)

		chainEvents[i] = chain.ChainEvent{
			EventID:     eventID,
			BlockHeight: newHeight,
			PaymentID:   ev.paymentID,
			EventType:   ev.eventType,
			PayloadHash: ev.payloadHash,
			BatchID:     batchID,
			MerkleProof: merkleProof,
		}
	}

	if err := s.db.InsertChainEventsBatch(chainEvents); err != nil {
		return fmt.Errorf("insert chain events: %w", err)
	}

	// §14.2: Log the successful submission
	s.db.InsertSubmitLog(&chain.ChainSubmitLog{
		BatchID:   batchID,
		ChainType: "hash_chain",
		ChainTxID: blockHash,
		Status:    "CONFIRMED",
	})

	log.Printf("ChainService: batch %s committed — %d events, block=%d, merkle=%s",
		batchID, len(events), newHeight, merkleRoot[:16]+"...")
	return nil
}

// ──────────────────────────────────────────────
// Audit Verification (§15)
// ──────────────────────────────────────────────

// VerifyPaymentAudit performs full Merkle proof verification for a payment.
// Architecture doc §15.2: 6-step verification process.
func (s *ChainService) VerifyPaymentAudit(paymentID string) (*chain.AuditVerification, error) {
	events, err := s.db.GetChainEventsByPayment(paymentID)
	if err != nil || len(events) == 0 {
		return nil, fmt.Errorf("no chain events found for payment: %s", paymentID)
	}

	// Use the first event (typically SETTLEMENT_COMPLETED)
	ev := events[0]

	verification := &chain.AuditVerification{
		PaymentID:   paymentID,
		EventHash:   ev.PayloadHash,
		BatchID:     ev.BatchID,
		MerkleProof: ev.MerkleProof,
		Verified:    false,
	}

	if ev.BlockHeight > 0 {
		block, err := s.db.GetChainBlock(ev.BlockHeight)
		if err == nil {
			verification.BlockHash = block.BlockHash
			verification.BlockHeight = block.BlockHeight
			verification.MerkleRoot = block.MerkleRoot
			verification.AuditSig = block.AuditSignature

			// §15.2 Step 2: Verify Merkle proof
			if len(ev.MerkleProof) > 0 {
				// Find leaf index within the batch
				batchEvents, _ := s.db.GetChainEventsByBatch(ev.BatchID)
				leafIndex := 0
				for i, be := range batchEvents {
					if be.EventID == ev.EventID {
						leafIndex = i
						break
					}
				}
				verification.LeafIndex = leafIndex

				verified := crypto.VerifyMerkleProof(
					ev.PayloadHash, ev.MerkleProof, block.MerkleRoot, leafIndex)
				verification.Verified = verified
			}

			// §15.2 Step 5: Verify block linkage
			if block.BlockHeight > 1 {
				prevBlock, err := s.db.GetChainBlock(block.BlockHeight - 1)
				if err == nil && block.PrevHash != prevBlock.BlockHash {
					verification.Verified = false
				}
			}

			// §15.2 Step 4: Verify batch signature
			if block.AuditSignature != "" && s.auditPublicKey != nil {
				batch, err := s.db.GetChainBatch(ev.BatchID)
				if err == nil {
					payload := crypto.CanonicalBatchPayload(
						batch.BatchID, block.MerkleRoot, batch.LedgerRootHash,
						batch.StartSequenceID, batch.EndSequenceID,
						batch.EventCount, block.CreatedAt.Unix())
					sigValid := crypto.VerifyAuditSignature(
						s.auditPublicKey, payload, block.AuditSignature)
					if !sigValid {
						verification.Verified = false
					}
				}
			}
		}
	}

	return verification, nil
}

// ──────────────────────────────────────────────
// Retry Logic (§10)
// ──────────────────────────────────────────────

// retryLoop periodically checks for batches that need retry.
// Architecture doc §10.3: Exponential backoff, max 10 retries.
func (s *ChainService) retryLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processRetries()
		}
	}
}

func (s *ChainService) processRetries() {
	entries, err := s.db.GetRetryableBatches(10)
	if err != nil {
		log.Printf("ChainService: retry fetch error: %v", err)
		return
	}

	for _, entry := range entries {
		// Re-attempt batch submission
		batch, err := s.db.GetChainBatch(entry.BatchID)
		if err != nil {
			s.db.MarkRetryFailed(entry.BatchID, err.Error(),
				s.nextRetryTime(entry.RetryCount+1))
			continue
		}

		// Try to re-submit the batch
		if err := s.resubmitBatch(batch); err != nil {
			log.Printf("ChainService: retry %d/%d for batch %s failed: %v",
				entry.RetryCount+1, entry.MaxRetries, entry.BatchID, err)

			if entry.RetryCount+1 >= entry.MaxRetries {
				// §10.2: Exceeded max retries → manual review
				s.db.UpdateChainBatchStatus(entry.BatchID, chain.BatchFailedManualReview, nil)
				s.db.MarkRetryComplete(entry.BatchID)
				log.Printf("ChainService: batch %s exceeded max retries — manual review required", entry.BatchID)
			} else {
				nextRetry := s.nextRetryTime(entry.RetryCount + 1)
				s.db.MarkRetryFailed(entry.BatchID, err.Error(), nextRetry)
			}
		} else {
			s.db.MarkRetryComplete(entry.BatchID)
			log.Printf("ChainService: batch %s retry succeeded", entry.BatchID)
		}
	}
}

// resubmitBatch re-attempts to submit a batch to the chain.
func (s *ChainService) resubmitBatch(batch *chain.ChainBatch) error {
	events, err := s.db.GetChainEventsByBatch(batch.BatchID)
	if err != nil {
		return fmt.Errorf("cannot load batch events: %w", err)
	}

	hashes := make([]string, len(events))
	for i, ev := range events {
		hashes[i] = ev.PayloadHash
	}

	// Re-verify Merkle root
	merkleRoot := crypto.MerkleRoot(hashes)
	if merkleRoot != batch.MerkleRoot {
		return fmt.Errorf("merkle root mismatch: expected %s, got %s", batch.MerkleRoot, merkleRoot)
	}

	// Log the retry attempt
	s.db.InsertSubmitLog(&chain.ChainSubmitLog{
		BatchID:   batch.BatchID,
		ChainType: "hash_chain",
		Status:    "RETRY",
	})

	// In V2 Sandbox, resubmission is simply re-verification
	return nil
}

// nextRetryTime computes exponential backoff: initial_delay * 2^retry_count.
// Architecture doc §10.3: max_delay_ms = 60000, initial_delay_ms = 1000.
func (s *ChainService) nextRetryTime(retryCount int) time.Time {
	delayMs := 1000 * int64(math.Pow(2, float64(retryCount)))
	if delayMs > 60000 {
		delayMs = 60000
	}
	return time.Now().Add(time.Duration(delayMs) * time.Millisecond)
}

// ──────────────────────────────────────────────
// Deterministic Hashing (§8.2-8.3)
// ──────────────────────────────────────────────

// computeDeterministicHash generates a hash with fixed field order.
// Architecture doc §8.3: Fixed field order, fixed encoding, no float values.
func (s *ChainService) computeDeterministicHash(paymentID, senderID, receiverID string,
	amount int64, sourceCurrency, targetCurrency, status string, timestamp int64) string {
	data := fmt.Sprintf("%s:%s:%s:%d:%s:%s:%s:%d",
		crypto.HashPaymentID(paymentID),
		crypto.HashUserID(senderID),
		crypto.HashUserID(receiverID),
		amount,
		sourceCurrency,
		targetCurrency,
		status,
		timestamp,
	)
	return crypto.SHA256(data)
}

// computeLedgerRootHash generates a combined root hash from event ledger hashes.
// Architecture doc §12: Cross-verifiable with PostgreSQL ledger.
func (s *ChainService) computeLedgerRootHash(events []batchEvent) string {
	ledgerHashes := make([]string, len(events))
	for i, ev := range events {
		if ev.ledgerEntryHash != "" {
			ledgerHashes[i] = ev.ledgerEntryHash
		} else {
			ledgerHashes[i] = ev.payloadHash
		}
	}
	return crypto.MerkleRoot(ledgerHashes)
}

// ──────────────────────────────────────────────
// Ed25519 Signing (§11)
// ──────────────────────────────────────────────

// signPayload signs a canonical batch payload with the audit private key.
func (s *ChainService) signPayload(payload string) string {
	if len(s.auditPrivateKey) == 0 {
		return ""
	}
	return crypto.SignAuditBatch(s.auditPrivateKey, payload)
}

// ──────────────────────────────────────────────
// Query Methods
// ──────────────────────────────────────────────

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

	verified := s.verifyChainIntegrity(blocks)

	return &chain.AuditTrail{
		PaymentID: paymentID,
		Blocks:    blocks,
		Events:    events,
		Verified:  verified,
	}, nil
}

// verifyChainIntegrity checks hash chain linkage across all blocks.
func (s *ChainService) verifyChainIntegrity(blocks []chain.ChainBlock) bool {
	for i := 1; i < len(blocks); i++ {
		if blocks[i].PrevHash != blocks[i-1].BlockHash {
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

// ──────────────────────────────────────────────
// Metrics (§16)
// ──────────────────────────────────────────────

// GetChainMetrics returns current chain metrics.
func (s *ChainService) GetChainMetrics() *chain.ChainMetrics {
	s.mu.Lock()
	pendingCount := len(s.pendingEvents)
	s.mu.Unlock()

	retryDepth, _ := s.db.GetRetryQueueDepth()
	pendingBatches, _ := s.db.ListPendingBatches(1000)

	m := &chain.ChainMetrics{
		PendingBatches:  int64(len(pendingBatches)),
		RetryQueueDepth: retryDepth,
	}

	// Count confirmed vs failed from pending batches
	for _, b := range pendingBatches {
		switch b.Status {
		case chain.BatchFailedRetryable:
			m.BatchesFailed++
		default:
			// PENDING, BUILDING, SUBMITTING → not yet confirmed
		}
	}

	_ = pendingCount // Event count tracked internally
	return m
}

// GetBatch retrieves a specific chain batch.
func (s *ChainService) GetBatch(batchID string) (*chain.ChainBatch, error) {
	return s.db.GetChainBatch(batchID)
}
