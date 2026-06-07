# Aspira Pay Blockchain Layer Optimization Plan

**Version:** 1.0  
**Author:** Aspira Studio  
**Target System:** Aspira Pay V2 Distributed Payment, Clearing, and Trading System  
**Core Focus:** Lightweight, high-throughput, privacy-preserving, auditable, and non-blocking blockchain audit layer  

---

## 1. Executive Summary

The blockchain layer in Aspira Pay should not be designed as the transaction execution layer.

In the Aspira Pay architecture:

- The C++ Trading Engine executes high-performance transactions.
- PostgreSQL stores the official double-entry accounting ledger.
- NATS handles event-driven communication.
- The Settlement Service writes accounting records and settlement batches.
- The Blockchain Audit Service records tamper-evident audit proofs.

The blockchain layer should be optimized as:

```text
A trusted audit proof layer for an off-chain high-performance transaction system.
```

It should not process every balance update directly on-chain. Instead, it should store hashes, Merkle roots, settlement batch proofs, and audit signatures.

The correct design principle is:

```text
Execute transactions off-chain.
Record proofs on-chain.
Verify integrity through Merkle proofs and hash chains.
```

---

## 2. Blockchain Layer Positioning

### 2.1 What the Blockchain Layer Should Do

The blockchain layer should provide:

```text
Immutable audit proof
Transaction status traceability
Settlement batch verification
Ledger root hash recording
Merkle proof verification
Cross-organization audit support
Tamper-evident event history
```

### 2.2 What the Blockchain Layer Should Not Do

The blockchain layer should not:

```text
Execute high-frequency transactions
Store KYC raw data
Store ID documents
Store bank card numbers
Store plaintext user information
Store full account balances
Replace PostgreSQL as the official accounting ledger
Block the main payment flow
Perform complex AML or risk checks
```

### 2.3 Correct Responsibility Split

```text
C++ Trading Engine:
    High-performance transaction execution

PostgreSQL:
    Official double-entry ledger and business state

Settlement Service:
    Ledger entries, settlement batches, reconciliation

Blockchain Audit Service:
    Hashes, Merkle roots, audit signatures, proof verification

External Blockchain / Fabric:
    Cross-party trusted proof storage
```

---

## 3. Main Optimization Principle

The most important blockchain optimization is:

> Do not write every transaction individually to blockchain.

Instead, use:

```text
Batching + Merkle Tree + Hash Chain + Asynchronous Submission
```

This allows Aspira Pay to reduce blockchain load while still keeping every transaction verifiable.

---

## 4. Batch-based On-chain Proof

### 4.1 Bad Design: Single Transaction On-chain Write

Do not design the blockchain layer like this:

```text
Payment 1 -> write blockchain
Payment 2 -> write blockchain
Payment 3 -> write blockchain
Payment 4 -> write blockchain
```

Problems:

```text
High latency
Low throughput
High cost
Higher operational complexity
Blockchain becomes a bottleneck
Main transaction flow becomes slow
```

### 4.2 Recommended Design: Merkle Batch Proof

Recommended flow:

```text
1000 payment events
    ↓
Generate event_hash for each event
    ↓
Build Merkle Tree
    ↓
Generate merkle_root
    ↓
Write only merkle_root + batch_id to blockchain
```

Example:

```text
payment_event_1 -> hash_1
payment_event_2 -> hash_2
payment_event_3 -> hash_3
...
payment_event_1000 -> hash_1000

hash_1 + hash_2 + ... -> Merkle Root

On-chain record:
    batch_id
    merkle_root
    ledger_root_hash
    event_count
    timestamp
    audit_signature
```

Benefits:

```text
Fewer blockchain writes
Lower resource usage
Higher throughput
Lower blockchain cost
Every transaction remains verifiable
```

---

## 5. Recommended Blockchain Audit Architecture

The Blockchain Audit Service should be internally divided into several modules.

```text
┌──────────────────────────────────────────────┐
│              Chain Event Consumer            │
│        Consume settlement.completed events    │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│              Hash Builder                    │
│        event_hash / payload_hash / signature │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│              Merkle Batch Builder            │
│        Batch events -> merkle_root            │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│              Chain Committer                 │
│        Hash Chain / Fabric / Tendermint       │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│              Audit Query API                 │
│        Proof query / verify / export          │
└──────────────────────────────────────────────┘
```

Core workflow:

```text
Consume settlement events
    -> normalize event payload
    -> generate event hash
    -> write chain_events
    -> build Merkle batch
    -> sign batch
    -> write internal hash chain
    -> asynchronously submit to external blockchain
    -> expose audit verification API
```

---

## 6. On-chain Data Design

### 6.1 Data That Must Not Be Stored On-chain

Do not store sensitive raw data on-chain:

```text
KYC raw data
ID number
Passport number
Residential address
Bank card number
User real name
ID document image
Full transaction detail
Full account balance
```

Reason:

```text
Blockchain data is difficult to delete.
Sensitive data creates privacy and compliance risks.
On-chain storage is expensive and slow.
```

### 6.2 Data Suitable for On-chain Storage

The blockchain layer should store only proofs and summaries:

```text
payment_id_hash
sender_hash
receiver_hash
amount_hash
currency_pair
event_type
settlement_batch_id
ledger_root_hash
merkle_root
timestamp
audit_signature
previous_block_hash
```

### 6.3 Chain Audit Record

```go
type ChainAuditRecord struct {
    BatchID          string `json:"batch_id"`
    MerkleRoot       string `json:"merkle_root"`
    LedgerRootHash   string `json:"ledger_root_hash"`
    EventCount       int    `json:"event_count"`
    StartSequenceID  uint64 `json:"start_sequence_id"`
    EndSequenceID    uint64 `json:"end_sequence_id"`
    PreviousHash     string `json:"previous_hash"`
    Timestamp        int64  `json:"timestamp"`
    AuditSignature   string `json:"audit_signature"`
}
```

---

## 7. Internal Hash Chain Design

For Aspira Pay V2, the recommended first implementation is:

```text
Internal Hash Chain + Merkle Batch Proof
```

This is lightweight, easy to deploy, and suitable for a sandbox or prototype system.

### 7.1 Hash Chain Formula

```text
block_hash = sha256(
    prev_block_hash
    + merkle_root
    + batch_id
    + event_count
    + timestamp
)
```

Each internal block records one batch proof.

### 7.2 chain_blocks Table

```sql
CREATE TABLE chain_blocks (
    id BIGSERIAL PRIMARY KEY,
    block_height BIGINT NOT NULL UNIQUE,
    block_hash VARCHAR(128) NOT NULL,
    prev_hash VARCHAR(128) NOT NULL,
    merkle_root VARCHAR(128) NOT NULL,
    event_count INT NOT NULL,
    start_sequence_id BIGINT NOT NULL,
    end_sequence_id BIGINT NOT NULL,
    audit_signature VARCHAR(256),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 7.3 chain_events Table

```sql
CREATE TABLE chain_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload_hash VARCHAR(128) NOT NULL,
    batch_id VARCHAR(64),
    block_height BIGINT,
    merkle_proof JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 7.4 Verification Path

Given a `payment_id`, the system should be able to:

```text
Find payment event
    -> calculate event_hash
    -> find merkle_proof
    -> verify event_hash belongs to merkle_root
    -> verify merkle_root belongs to chain_block
    -> verify block_hash links to previous block hash
    -> verify audit signature
```

---

## 8. Merkle Tree Optimization

### 8.1 Configurable Batch Trigger

Do not use a fixed batch strategy only.

Use both size-based and time-based triggers:

```yaml
chain:
  batch_size: 1000
  batch_interval_ms: 1000
```

Trigger rules:

```text
Create a batch when event count reaches 1000.
Create a batch when 1000 ms has passed.
```

This balances:

```text
High throughput under heavy traffic
Acceptable audit delay under low traffic
```

### 8.2 Merkle Leaf Design

Do not hash raw unordered JSON directly.

Bad design:

```text
event_hash = sha256(raw_json)
```

This may generate inconsistent hashes if JSON field order changes.

Recommended design:

```text
canonical_payload = canonical_json(event)
event_hash = sha256(canonical_payload)
```

Better design:

```text
event_hash = sha256(
    event_id
    + payment_id_hash
    + event_type
    + sequence_id
    + ledger_entry_hash
    + timestamp
)
```

### 8.3 Deterministic Hashing

All hash inputs must be deterministic:

```text
Fixed field order
Fixed encoding
Fixed timestamp format
Fixed currency representation
Fixed amount representation
No floating-point values
```

---

## 9. Asynchronous Blockchain Submission

The blockchain layer must not block the main transaction flow.

### 9.1 Main Payment Flow

The main flow should be:

```text
Payment Service
    ↓
C++ Trading Engine
    ↓
Settlement Service
    ↓
PostgreSQL Ledger Completed
```

### 9.2 Blockchain Audit Flow

The blockchain audit flow should be:

```text
Settlement Completed
    ↓
Write chain_events asynchronously
    ↓
Build Merkle Root asynchronously
    ↓
Submit batch proof asynchronously
    ↓
Update chain_confirmed status
```

### 9.3 Recommended State Flow

```text
SETTLED
  ↓
CHAIN_PENDING
  ↓
CHAIN_CONFIRMED
  ↓
COMPLETED
```

For V2, a payment can be considered business-settled after the local double-entry ledger is completed. Blockchain confirmation can be treated as an asynchronous audit confirmation.

---

## 10. Failure Handling and Retry Strategy

Blockchain submission failure must not corrupt the official accounting ledger.

### 10.1 Failure Flow

```text
Settlement completed
    ↓
Chain submission failed
    ↓
Add batch to chain_retry_queue
    ↓
Background retry
    ↓
If retries exceed threshold, trigger manual alert
```

### 10.2 Chain Status

```text
CHAIN_PENDING
CHAIN_SUBMITTING
CHAIN_CONFIRMED
CHAIN_FAILED_RETRYABLE
CHAIN_FAILED_MANUAL_REVIEW
```

### 10.3 Retry Configuration

```yaml
chain_retry:
  max_retry: 10
  initial_delay_ms: 1000
  max_delay_ms: 60000
  backoff: exponential
```

### 10.4 Important Rule

Do not roll back completed double-entry ledger records only because blockchain submission failed.

The blockchain layer is an audit proof layer, not the primary accounting execution layer.

---

## 11. Batch Signature Design

Each batch should be signed by the Aspira Pay audit private key.

### 11.1 Signature Payload

```text
audit_signature = sign(
    batch_id
    + merkle_root
    + ledger_root_hash
    + start_sequence_id
    + end_sequence_id
    + timestamp
)
```

### 11.2 Recommended Algorithms

| Algorithm | Use Case |
|---|---|
| Ed25519 | Fast internal audit signature |
| ECDSA secp256k1 | Web3-compatible signature |
| Multi-signature | Future multi-party audit network |

### 11.3 Why Sign Batch Proofs

Batch signatures provide:

```text
Proof that the batch was generated by Aspira Pay Audit Service
Protection against forged database batch records
Third-party verifiability
Non-repudiation
```

---

## 12. Multi-layer Audit Model

Aspira Pay should not rely on blockchain alone.

Use a four-layer audit structure:

```text
Layer 1: C++ Engine WAL
Layer 2: PostgreSQL ledger_entries
Layer 3: Internal Hash Chain / Merkle Root
Layer 4: External Consortium Chain / Fabric / Tendermint
```

Flow:

```text
Engine Event
  ↓
WAL Record
  ↓
Ledger Entry
  ↓
Merkle Batch
  ↓
Blockchain Proof
```

Cross-verification:

```text
WAL proves the engine executed the command.
ledger_entries prove the accounting result.
Merkle root proves events were not modified.
Blockchain proof proves the audit batch existed at a specific time.
```

---

## 13. Evolution Path

### 13.1 Stage 1: Internal Hash Chain

Suitable for:

```text
MVP
V2 Sandbox
Low-resource deployment
Development and testing
```

Features:

```text
Simple deployment
Low resource usage
Easy debugging
Easy verification
```

### 13.2 Stage 2: Hyperledger Fabric Audit Network

Suitable for:

```text
Multi-organization cooperation
Bank partners
Clearing partners
External audit nodes
```

Fabric should store only:

```text
batch_id
merkle_root
ledger_root_hash
event_count
signature
timestamp
```

### 13.3 Stage 3: Aspira Consortium Chain

Suitable for future:

```text
Custom clearing network
Multiple validator nodes
Cross-organization payment clearing
Regulator nodes
Auditor nodes
```

Possible technologies:

```text
Tendermint / CometBFT
Substrate
Cosmos SDK
```

---

## 14. Local Database Design for Blockchain Service

The Blockchain Audit Service should maintain its own local state.

Recommended tables:

```text
chain_events
chain_batches
chain_blocks
chain_submit_logs
chain_retry_queue
chain_proofs
```

### 14.1 chain_batches Table

```sql
CREATE TABLE chain_batches (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) UNIQUE NOT NULL,
    merkle_root VARCHAR(128) NOT NULL,
    ledger_root_hash VARCHAR(128),
    event_count INT NOT NULL,
    start_sequence_id BIGINT NOT NULL,
    end_sequence_id BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL,
    audit_signature VARCHAR(256),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    submitted_at TIMESTAMP,
    confirmed_at TIMESTAMP
);
```

### 14.2 chain_submit_logs Table

```sql
CREATE TABLE chain_submit_logs (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) NOT NULL,
    chain_type VARCHAR(32) NOT NULL,
    chain_tx_id VARCHAR(128),
    status VARCHAR(32) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 14.3 chain_retry_queue Table

```sql
CREATE TABLE chain_retry_queue (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) NOT NULL,
    retry_count INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMP NOT NULL,
    last_error TEXT,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

---

## 15. Audit Verification API

Aspira Pay should provide APIs to verify payment audit proofs.

### 15.1 Query Payment Audit Proof

```http
GET /api/v2/audit/payments/{payment_id}
```

Response:

```json
{
  "payment_id": "pay_001",
  "event_hash": "abc...",
  "batch_id": "batch_20260607_001",
  "merkle_root": "def...",
  "merkle_proof": [
    "hash_left_1",
    "hash_right_2"
  ],
  "block_hash": "block_hash_001",
  "chain_tx_id": "chain_tx_001",
  "verified": true
}
```

### 15.2 Verification Process

```text
1. Recalculate event_hash.
2. Use merkle_proof to verify that event_hash belongs to merkle_root.
3. Verify that merkle_root belongs to the batch.
4. Verify batch signature.
5. Verify block_hash and previous hash linkage.
6. If external chain is integrated, verify chain_tx_id.
```

---

## 16. Metrics and Observability

The Blockchain Audit Service should expose its own metrics.

Required metrics:

```text
chain_events_received_total
chain_batch_created_total
chain_batch_size
chain_batch_build_latency_ms
chain_submit_latency_ms
chain_submit_failed_total
chain_retry_queue_depth
chain_confirmed_total
chain_pending_total
merkle_build_latency_ms
audit_verify_latency_ms
```

Important warning signals:

```text
chain_retry_queue_depth continuously increasing
chain_submit_failed_total increasing rapidly
chain_batch_build_latency_ms too high
chain_pending_total too high
```

---

## 17. Anti-patterns to Avoid

Avoid these blockchain design mistakes:

```text
Write every transaction synchronously to blockchain.
Store KYC data on-chain.
Store ID document images on-chain.
Store plaintext balances on-chain.
Let the C++ engine call blockchain SDK directly.
Make Payment Service wait for blockchain confirmation.
Put complex risk logic inside smart contracts.
Put complex settlement logic inside smart contracts.
Rollback local ledger only because blockchain submission failed.
Use blockchain as a replacement for PostgreSQL ledger.
```

These patterns will make the system slower, more expensive, less private, and harder to operate.

---

## 18. Recommended Optimization Priority

### 18.1 Phase 1: Lightweight Hash Chain

Implement first:

```text
chain_events
chain_batches
chain_blocks
event_hash
merkle_root
block_hash
prev_hash
audit_signature
verify API
```

### 18.2 Phase 2: Batch Submission and Retry

Add:

```text
batch_size
batch_interval
retry_queue
submit_logs
CHAIN_PENDING / CHAIN_CONFIRMED states
```

### 18.3 Phase 3: External Consortium Chain

Integrate:

```text
Hyperledger Fabric
or Tendermint / CometBFT
```

### 18.4 Phase 4: Multi-party Audit

Add:

```text
Partner node
Auditor node
Regulator node
Multi-signature batch proof
```

---

## 19. Final Recommended Blockchain Flow

```text
Settlement Service
  ↓ settlement.completed event
NATS
  ↓
Blockchain Audit Service
  ↓
Normalize Event
  ↓
Generate event_hash
  ↓
Write chain_events
  ↓
Build Merkle Batch
  ↓
Sign Batch
  ↓
Write Internal Hash Chain
  ↓
Async Submit to Fabric / Consortium Chain
  ↓
Update chain_confirmed
```

Core principles:

```text
Asynchronous
Batched
Hash-only
Privacy-preserving
Verifiable
Retryable
Non-blocking
Cross-verifiable with PostgreSQL ledger and C++ WAL
```

---

## 20. Conclusion

The blockchain layer in Aspira Pay should not be optimized by making the chain more complex.

It should be optimized by making blockchain usage more precise:

```text
Use blockchain for audit proof.
Use Merkle batches for scalability.
Use internal hash chain for V2.
Use external consortium chain for future cross-party trust.
Keep sensitive data off-chain.
Keep the main transaction flow non-blocking.
```

The best design is:

```text
C++ Engine executes transactions.
PostgreSQL records the official double-entry ledger.
Settlement Service creates settlement batches.
Blockchain Audit Service creates tamper-evident proofs.
Fabric or Consortium Chain stores external trusted proof.
```

The final goal is:

```text
A high-performance off-chain transaction system
    + a lightweight on-chain audit proof layer.
```
