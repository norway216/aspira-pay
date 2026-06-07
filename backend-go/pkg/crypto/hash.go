// Package crypto provides cryptographic helpers for hashing, HMAC, hash chains,
// Merkle trees, and audit signatures.
package crypto

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// SHA256 returns the hex-encoded SHA-256 hash of data.
func SHA256(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

// SHA256Bytes returns the raw SHA-256 hash.
func SHA256Bytes(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// HMACSHA256 returns the hex-encoded HMAC-SHA256 signature.
func HMACSHA256(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// HashUserID hashes a user ID for blockchain storage (privacy).
func HashUserID(userID string) string {
	return SHA256("user:" + userID)
}

// HashPaymentID hashes a payment ID for blockchain storage.
func HashPaymentID(paymentID string) string {
	return SHA256("payment:" + paymentID)
}

// HashAmount hashes an amount value for blockchain storage.
func HashAmount(amount int64, currency string) string {
	return SHA256(fmt.Sprintf("amount:%d:%s", amount, currency))
}

// MerkleRoot computes the Merkle root hash from a list of leaf hashes.
// If the list is empty, returns all-zeros hash.
func MerkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return strings.Repeat("0", 64)
	}

	current := make([]string, len(hashes))
	copy(current, hashes)

	for len(current) > 1 {
		var next []string
		for i := 0; i < len(current); i += 2 {
			if i+1 < len(current) {
				// Hash the pair, sorted for consistency
				pair := current[i] + current[i+1]
				if current[i] > current[i+1] {
					pair = current[i+1] + current[i]
				}
				next = append(next, SHA256(pair))
			} else {
				// Odd one out — hash with itself
				next = append(next, SHA256(current[i]+current[i]))
			}
		}
		current = next
	}
	return current[0]
}

// MerkleProof generates a Merkle proof (sibling hashes) for a given leaf.
func MerkleProof(hashes []string, leafIndex int) []string {
	if leafIndex < 0 || leafIndex >= len(hashes) {
		return nil
	}

	var proof []string
	current := make([]string, len(hashes))
	copy(current, hashes)
	idx := leafIndex

	for len(current) > 1 {
		var next []string
		for i := 0; i < len(current); i += 2 {
			if i+1 < len(current) {
				pair := current[i] + current[i+1]
				if current[i] > current[i+1] {
					pair = current[i+1] + current[i]
				}
				next = append(next, SHA256(pair))

				// Record sibling
				if i == idx {
					proof = append(proof, current[i+1])
				} else if i+1 == idx {
					proof = append(proof, current[i])
				}
			} else {
				next = append(next, SHA256(current[i]+current[i]))
				if i == idx {
					proof = append(proof, current[i])
				}
			}
		}
		current = next
		idx = idx / 2
	}
	return proof
}

// VerifyMerkleProof verifies a Merkle proof for a leaf hash.
func VerifyMerkleProof(leafHash string, proof []string, root string, leafIndex int) bool {
	current := leafHash
	idx := leafIndex

	for _, sibling := range proof {
		var pair string
		if idx%2 == 0 {
			// Current is left child
			if current > sibling {
				pair = sibling + current
			} else {
				pair = current + sibling
			}
		} else {
			// Current is right child
			if sibling > current {
				pair = current + sibling
			} else {
				pair = sibling + current
			}
		}
		current = SHA256(pair)
		idx = idx / 2
	}

	return current == root
}

// HashChainBlock computes a hash chain block hash: SHA256(blockData + prevHash).
func HashChainBlock(blockData, prevHash string) string {
	return SHA256(blockData + prevHash)
}

// BlockHash computes the hash for a set of events in a block.
func BlockHash(eventHashes []string, prevHash string) string {
	sort.Strings(eventHashes)
	merkleRoot := MerkleRoot(eventHashes)
	return SHA256(fmt.Sprintf("%s:%s:%d", merkleRoot, prevHash, len(eventHashes)))
}

// ──────────────────────────────────────────────
// Deterministic Hashing (§8.2-8.3)
// Fixed field order, fixed encoding, no float values.
// ──────────────────────────────────────────────

// DeterministicEventHash computes an event hash with fixed field order.
// Architecture doc §8.3: All hash inputs must be deterministic.
// Format: event_id + payment_id_hash + event_type + sequence_id + ledger_entry_hash + timestamp
func DeterministicEventHash(eventID, paymentID, eventType, ledgerEntryHash string, sequenceID uint64, timestamp int64) string {
	data := fmt.Sprintf("%s:%s:%s:%d:%s:%d",
		eventID,
		HashPaymentID(paymentID),
		eventType,
		sequenceID,
		ledgerEntryHash,
		timestamp,
	)
	return SHA256(data)
}

// CanonicalBatchPayload creates a deterministic canonical payload for batch signing.
// Architecture doc §11.1: Fixed field order ensures consistent signatures.
func CanonicalBatchPayload(batchID, merkleRoot, ledgerRootHash string,
	startSeq, endSeq int64, eventCount int, timestamp int64) string {
	return fmt.Sprintf("%s:%s:%s:%d:%d:%d:%d",
		batchID,
		merkleRoot,
		ledgerRootHash,
		startSeq,
		endSeq,
		eventCount,
		timestamp,
	)
}

// BatchedHashChainBlock computes a hash chain block hash per §7.1 formula:
//
//	block_hash = sha256(prev_block_hash + merkle_root + batch_id + event_count + timestamp)
func BatchedHashChainBlock(prevHash, merkleRoot, batchID string, eventCount int, timestamp int64) string {
	data := fmt.Sprintf("%s:%s:%s:%d:%d", prevHash, merkleRoot, batchID, eventCount, timestamp)
	return SHA256(data)
}

// ──────────────────────────────────────────────
// Ed25519 Audit Signatures (§11)
// Architecture doc §11.2: Ed25519 for fast internal audit signatures.
// ──────────────────────────────────────────────

// GenerateKeyPair generates an Ed25519 key pair for audit signing.
func GenerateKeyPair() (publicKey, privateKey []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return pub, priv, nil
}

// SignAuditBatch signs a canonical batch payload with Ed25519.
// Architecture doc §11.1: audit_signature = sign(batch_id + merkle_root + ...)
func SignAuditBatch(privateKey ed25519.PrivateKey, payload string) string {
	sig := ed25519.Sign(privateKey, []byte(payload))
	return hex.EncodeToString(sig)
}

// VerifyAuditSignature verifies an Ed25519 audit signature for a batch payload.
func VerifyAuditSignature(publicKey ed25519.PublicKey, payload, signatureHex string) bool {
	sig, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, []byte(payload), sig)
}

// SignBytes signs arbitrary bytes with Ed25519 (used for batch payloads).
func SignBytes(privateKey ed25519.PrivateKey, data []byte) []byte {
	return ed25519.Sign(privateKey, data)
}

// VerifyBytes verifies an Ed25519 signature.
func VerifyBytes(publicKey ed25519.PublicKey, data, signature []byte) bool {
	return ed25519.Verify(publicKey, data, signature)
}
