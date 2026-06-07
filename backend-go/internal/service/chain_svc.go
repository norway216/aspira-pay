package service

import (
	"fmt"
	"log"
	"time"

	"github.com/aspira/aspira-pay/internal/domain/chain"
	"github.com/aspira/aspira-pay/internal/repository"
	"github.com/aspira/aspira-pay/pkg/crypto"
	"github.com/aspira/aspira-pay/pkg/idgen"
)

// ChainService manages the blockchain audit layer (Hash Chain implementation).
// It records payment events on an append-only hash chain for tamper-proof auditing.
type ChainService struct {
	db *repository.DB
}

// NewChainService creates a new ChainService.
func NewChainService(db *repository.DB) *ChainService {
	return &ChainService{db: db}
}

// RecordPaymentOnChain records a payment event on the hash chain.
// This is the main entry point for on-chain recording.
func (s *ChainService) RecordPaymentOnChain(paymentID string) error {
	order, err := s.db.GetPaymentOrder(paymentID)
	if err != nil {
		return fmt.Errorf("payment not found: %w", err)
	}

	// Record on chain
	eventID := idgen.EventID()
	payloadHash := s.computePaymentPayloadHash(order.PaymentID, order.SenderUserID,
		order.ReceiverUserID, order.SourceAmount, order.SourceCurrency, order.TargetCurrency,
		string(order.Status))

	if err := s.recordChainEvent(paymentID, chain.EventSettlementCompleted, payloadHash); err != nil {
		return fmt.Errorf("chain record failed: %w", err)
	}

	// Also record payment completion
	if order.Status == "COMPLETED" || order.Status == "CHAIN_CONFIRMED" {
		if err := s.recordChainEvent(paymentID, chain.EventPaymentCompleted,
			crypto.SHA256(fmt.Sprintf("completed:%s:%d", paymentID, time.Now().UnixNano()))); err != nil {
			return err
		}
	}

	log.Printf("Chain: payment %s recorded (event: %s)", paymentID, eventID)
	return nil
}

// recordChainEvent writes a single event to the chain, creating a new block if needed.
func (s *ChainService) recordChainEvent(paymentID, eventType, payloadHash string) error {
	// Get latest block
	latestBlock, err := s.db.GetLatestBlock()
	if err != nil {
		return fmt.Errorf("no chain blocks: %w", err)
	}

	// Decide whether to create a new block or append to current
	// For simplicity, create a new block for each event in Sandbox
	// Production would batch events per architecture §10.4
	newHeight := latestBlock.BlockHeight + 1

	// Compute new block hash
	blockHash := chainHashBlock(newHeight, latestBlock.BlockHash, []string{payloadHash})
	merkleRoot := crypto.MerkleRoot([]string{payloadHash})

	// Create block
	block := &chain.ChainBlock{
		BlockHeight: newHeight,
		BlockHash:   blockHash,
		PrevHash:    latestBlock.BlockHash,
		MerkleRoot:  merkleRoot,
		EventCount:  1,
	}

	if err := s.db.InsertChainBlock(block); err != nil {
		return fmt.Errorf("cannot insert chain block: %w", err)
	}

	// Create event
	event := &chain.ChainEvent{
		EventID:     idgen.EventID(),
		BlockHeight: newHeight,
		PaymentID:   paymentID,
		EventType:   eventType,
		PayloadHash: payloadHash,
	}

	if err := s.db.InsertChainEvent(event); err != nil {
		return fmt.Errorf("cannot insert chain event: %w", err)
	}

	return nil
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

// chainHashBlock computes a hash chain block hash.
func chainHashBlock(height int64, prevHash string, eventHashes []string) string {
	data := fmt.Sprintf("%d:%s", height, prevHash)
	for _, h := range eventHashes {
		data += ":" + h
	}
	return crypto.SHA256(data)
}
