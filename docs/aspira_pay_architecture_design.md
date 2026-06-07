# Aspira Pay Architecture Design

**Version:** V2 Distributed Payment, Clearing, and Trading System  
**Author:** Aspira Studio  
**Target:** High-performance, stable, distributed, blockchain-auditable cross-border payment system  
**Core Stack:** Go Microservices + C++ Trading Engine + PostgreSQL Double-entry Ledger + NATS Event Streaming + Blockchain Audit Layer

---

## 1. Executive Summary

Aspira Pay is designed as a distributed cross-border payment, clearing, and transaction system.

The system follows this core principle:

> Go microservices handle payment orchestration, KYC, risk control, and settlement coordination.  
> The C++ engine handles high-performance transaction execution.  
> PostgreSQL stores the double-entry accounting ledger.  
> NATS handles event-driven communication.  
> Blockchain records trusted audit proofs for the transaction process.

Aspira Pay is not just a payment API. It is a complete financial transaction infrastructure with:

- User onboarding
- KYC and AML checks
- Multi-currency account management
- Payment order lifecycle management
- High-performance transaction execution
- Double-entry accounting
- Settlement and reconciliation
- Blockchain-based audit trail
- Event-driven distributed architecture
- High availability and observability

---

## 2. Design Goals

### 2.1 Functional Goals

Aspira Pay should support:

1. User registration and identity verification.
2. KYC and AML checks before payment execution.
3. Multi-currency account management.
4. Cross-border payment order creation.
5. FX quote locking.
6. Risk control and transaction limit checks.
7. High-performance transaction execution through a C++ engine.
8. Double-entry ledger recording.
9. Settlement batch generation.
10. Blockchain-based transaction audit.
11. Admin dashboard for operations, compliance, and monitoring.

### 2.2 Non-functional Goals

| Goal | Description |
|---|---|
| High performance | Low-latency transaction execution through C++ engine |
| High stability | Event-driven architecture, retry, compensation, and recovery |
| Low resource usage | Lightweight services, NATS, efficient data flow |
| High concurrency | Stateless Go services and asynchronous event processing |
| Auditability | Every critical event is traceable and verifiable |
| Security | KYC, AML, encryption, mTLS, API signatures |
| Scalability | Horizontally scalable microservices |
| Easy deployment | Docker Compose first, Kubernetes later |

---

## 3. System Boundary

### 3.1 What Aspira Pay V2 Supports

| Capability | Supported | Description |
|---|---:|---|
| User registration | Yes | Basic user identity and account creation |
| KYC workflow | Yes | Identity verification status and risk level |
| AML and risk rules | Yes | Rule-based checks and manual review |
| Multi-currency account | Yes | USD, EUR, JPY, HKD, CNY, etc. |
| Payment order | Yes | Cross-border payment order lifecycle |
| FX quote | Yes | Simulated or external FX quote provider |
| C++ transaction engine | Yes | Fund freeze, debit, credit, fee calculation |
| Double-entry ledger | Yes | PostgreSQL-based accounting ledger |
| Blockchain audit | Yes | Hash, Merkle root, settlement batch proof |
| Admin dashboard | Yes | User, payment, ledger, audit, risk views |
| Real banking channel | No in V2 | Can be integrated in later production phase |
| Real money movement | No in V2 | V2 should use sandbox balances |

### 3.2 What Aspira Pay V2 Should Not Do

Aspira Pay V2 should not directly handle real customer funds without proper licensing, banking partnerships, and compliance approval.

V2 should focus on:

```text
KYC -> Risk Check -> Payment Order -> C++ Engine Execution
    -> Double-entry Ledger -> Blockchain Audit -> Settlement Status
```

---

## 4. High-level Architecture

```text
┌──────────────────────────────────────────────────────────────────────┐
│                         Client / Merchant / Admin                    │
│                      Web App / Mobile App / API Client               │
└───────────────────────────────────┬──────────────────────────────────┘
                                    │ HTTPS
                                    ▼
┌──────────────────────────────────────────────────────────────────────┐
│                              API Gateway                             │
│              Go / JWT / API Key / HMAC Signature / Rate Limit         │
└───────────────────────────────────┬──────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        ▼                           ▼                           ▼
┌──────────────────┐      ┌──────────────────┐      ┌──────────────────┐
│ User Service      │      │ KYC Service       │      │ Risk/AML Service │
│ Go                │      │ Go                │      │ Go               │
│ User & accounts   │      │ Identity checks   │      │ Risk decisions   │
└────────┬─────────┘      └────────┬─────────┘      └────────┬─────────┘
         │                         │                         │
         └─────────────────────────┼─────────────────────────┘
                                   ▼
                          ┌──────────────────┐
                          │ FX Quote Service │
                          │ Go               │
                          │ FX rate & fee    │
                          └────────┬─────────┘
                                   ▼
                          ┌──────────────────┐
                          │ Payment Service  │
                          │ Go               │
                          │ Order state      │
                          │ Idempotency      │
                          │ Orchestration    │
                          └────────┬─────────┘
                                   ▼
                          ┌──────────────────┐
                          │ NATS JetStream   │
                          │ Event Streaming  │
                          └────────┬─────────┘
                                   ▼
                          ┌──────────────────┐
                          │ C++ Trading      │
                          │ Engine           │
                          │ Execution Core   │
                          └────────┬─────────┘
                                   ▼
                          ┌──────────────────┐
                          │ Settlement       │
                          │ Service          │
                          │ Go               │
                          │ Ledger & batch   │
                          └────────┬─────────┘
                                   ▼
                          ┌──────────────────┐
                          │ Blockchain       │
                          │ Audit Service    │
                          │ Hash / Merkle    │
                          │ Fabric optional  │
                          └────────┬─────────┘
                                   ▼
┌──────────────────────────────────────────────────────────────────────┐
│                              Data Layer                              │
│       PostgreSQL / Redis / ClickHouse / MinIO / RocksDB / WAL         │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 5. Core Architecture Principle

Aspira Pay separates payment orchestration, transaction execution, accounting, and audit.

```text
Go Microservices:
    Business workflow, KYC, AML, payment order, settlement coordination

C++ Trading Engine:
    Low-latency transaction execution, fund freeze, debit, credit, fee calculation

PostgreSQL:
    Source of truth for payment orders, accounts, and double-entry ledger

NATS:
    Event-driven communication and asynchronous transaction processing

Blockchain:
    Immutable audit proof, transaction hash, settlement batch root hash
```

This design avoids putting all logic into one system.

It also avoids using blockchain as the high-frequency transaction processor. The blockchain layer is used for trusted audit and verification, not for hot-path execution.

---

## 6. Service Decomposition

### 6.1 API Gateway

The API Gateway is the external entry point.

Responsibilities:

- TLS termination
- JWT authentication
- API key validation
- HMAC request signature verification
- Rate limiting
- IP allowlist and denylist
- Request routing
- Request ID and trace ID generation
- Access logging

Recommended stack:

```text
Go + Gin / Fiber / Echo
Nginx / Envoy
Redis for rate limiting
OpenTelemetry for tracing
```

Example APIs:

```text
POST /api/v2/users/register
POST /api/v2/kyc/submit
POST /api/v2/payments
GET  /api/v2/payments/{payment_id}
GET  /api/v2/ledger/{payment_id}
GET  /api/v2/audit/{payment_id}
```

### 6.2 User Service

The User Service manages users and account profiles.

Responsibilities:

- User registration
- Login session management
- User status management
- Account ownership management
- User risk level binding
- User freeze and unfreeze

User status:

```go
type UserStatus string

const (
    UserPendingKYC UserStatus = "PENDING_KYC"
    UserActive     UserStatus = "ACTIVE"
    UserFrozen     UserStatus = "FROZEN"
    UserRejected   UserStatus = "REJECTED"
)
```

Core tables:

```text
users
accounts
account_limits
user_devices
login_sessions
```

### 6.3 KYC Service

The KYC Service handles identity verification before payment is allowed.

Responsibilities:

- Identity data collection
- Document verification
- Face verification result integration
- Address verification
- KYC risk level
- Manual review
- KYC expiration management
- KYC audit logs

KYC status:

```go
type KYCStatus string

const (
    KYCPending      KYCStatus = "PENDING"
    KYCSubmitted    KYCStatus = "SUBMITTED"
    KYCReviewing    KYCStatus = "MANUAL_REVIEW"
    KYCApproved     KYCStatus = "APPROVED"
    KYCRejected     KYCStatus = "REJECTED"
    KYCExpired      KYCStatus = "EXPIRED"
)
```

A user cannot create a real payment order unless KYC status is `APPROVED`.

### 6.4 Risk and AML Service

The Risk and AML Service performs transaction-level checks before execution.

Responsibilities:

- User status check
- KYC status check
- Blacklist check
- Sanctions list simulation
- Country and region restriction
- Transaction amount limit
- Daily and monthly limit
- High-frequency transaction detection
- Suspicious split transaction detection
- IP and device anomaly detection
- Manual review decision

Risk decision:

```go
type RiskDecision string

const (
    RiskPass   RiskDecision = "PASS"
    RiskReject RiskDecision = "REJECT"
    RiskReview RiskDecision = "MANUAL_REVIEW"
)

type RiskResult struct {
    Decision RiskDecision `json:"decision"`
    Score    int          `json:"score"`
    Reasons  []string     `json:"reasons"`
}
```

Example rules:

```text
Rule 1: Reject payment if user KYC is not approved.
Rule 2: Send to manual review if payment amount exceeds user limit.
Rule 3: Reject payment if target country is restricted.
Rule 4: Send to manual review if user creates more than 10 payments in 1 minute.
Rule 5: Send to manual review if a new user creates a large transaction within 24 hours.
```

### 6.5 FX Quote Service

The FX Quote Service provides exchange rate and fee calculation.

Responsibilities:

- Currency pair management
- FX rate retrieval
- Quote generation
- Quote lock
- Fee calculation
- Quote expiration check

Important rule:

```text
Never use float or double for money.
Use int64 for minor currency units.
Use decimal or NUMERIC for FX rate.
```

Example:

```text
USD 100.25 -> 10025 cents
JPY 1000   -> 1000 yen
EUR 10.99  -> 1099 cents
```

Quote model:

```go
type FXQuote struct {
    QuoteID         string `json:"quote_id"`
    SourceCurrency string `json:"source_currency"`
    TargetCurrency string `json:"target_currency"`
    Rate           string `json:"rate"`
    SourceAmount   int64  `json:"source_amount"`
    TargetAmount   int64  `json:"target_amount"`
    FeeAmount      int64  `json:"fee_amount"`
    ExpiresAt      int64  `json:"expires_at"`
}
```

### 6.6 Payment Service

The Payment Service is the Go-based business orchestration core.

Responsibilities:

- Payment order creation
- Idempotency check
- User status check
- KYC status check
- Risk decision integration
- FX quote lock
- Payment state machine
- Outbox event writing
- Engine command publishing
- Payment status update

The Payment Service does not directly update balances. Balance operations are executed by the C++ Trading Engine and later persisted by the Settlement Service.

Payment status:

```go
type PaymentStatus string

const (
    PaymentCreated              PaymentStatus = "CREATED"
    PaymentKYCChecked           PaymentStatus = "KYC_CHECKED"
    PaymentRiskChecked          PaymentStatus = "RISK_CHECKED"
    PaymentQuoteLocked          PaymentStatus = "QUOTE_LOCKED"
    PaymentFundsFreezeRequested PaymentStatus = "FUNDS_FREEZE_REQUESTED"
    PaymentSubmittedToEngine    PaymentStatus = "SUBMITTED_TO_ENGINE"
    PaymentEngineExecuted       PaymentStatus = "ENGINE_EXECUTED"
    PaymentSettlementPending    PaymentStatus = "SETTLEMENT_PENDING"
    PaymentSettled              PaymentStatus = "SETTLED"
    PaymentChainConfirmed       PaymentStatus = "CHAIN_CONFIRMED"
    PaymentCompleted            PaymentStatus = "COMPLETED"
    PaymentRejected             PaymentStatus = "REJECTED"
    PaymentFailed               PaymentStatus = "FAILED"
    PaymentCancelled            PaymentStatus = "CANCELLED"
    PaymentRefunded             PaymentStatus = "REFUNDED"
)
```

Payment state flow:

```text
CREATED
  ↓
KYC_CHECKED
  ↓
RISK_CHECKED
  ↓
QUOTE_LOCKED
  ↓
FUNDS_FREEZE_REQUESTED
  ↓
SUBMITTED_TO_ENGINE
  ↓
ENGINE_EXECUTED
  ↓
SETTLEMENT_PENDING
  ↓
SETTLED
  ↓
CHAIN_CONFIRMED
  ↓
COMPLETED
```

Failure states:

```text
REJECTED
FAILED
CANCELLED
REFUNDED
MANUAL_REVIEW
```

### 6.7 C++ Trading Engine

The C++ Trading Engine is the high-performance transaction execution core.

It is not responsible for KYC, AML, user profile lookup, or complex business workflow.

Responsibilities:

- Receive transaction commands
- Validate sequence ID
- Validate idempotency key
- Check account balance from in-memory ledger
- Freeze funds
- Debit source account
- Credit target account
- Calculate fee
- Write WAL
- Generate engine events
- Publish execution result

#### 6.7.1 Internal Architecture

```text
┌─────────────────────────────────────────────┐
│                Engine Gateway               │
│           gRPC / NATS / TCP / FlatBuffers    │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│                Command Decoder              │
│       Signature / request_id / sequence_id   │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│              Lock-free Command Queue         │
│              MPSC Ring Buffer                │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│                Core Engine Loop              │
│       Single Writer / Ordered Execution       │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│                In-memory Ledger              │
│       account_id -> available/frozen/settled │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│                   WAL Log                    │
│       Command log / Event log / Snapshot     │
└──────────────────────┬──────────────────────┘
                       ▼
┌─────────────────────────────────────────────┐
│                Event Publisher               │
│          engine.executed / engine.rejected    │
└─────────────────────────────────────────────┘
```

#### 6.7.2 C++ Data Structures

```cpp
enum class CommandType {
    FREEZE_FUNDS,
    EXECUTE_PAYMENT,
    RELEASE_FUNDS,
    REFUND_PAYMENT,
    SETTLEMENT_BATCH
};

enum class EngineResult {
    ACCEPTED,
    REJECTED,
    EXECUTED,
    DUPLICATED,
    INSUFFICIENT_FUNDS,
    INVALID_SEQUENCE
};

struct PaymentCommand {
    uint64_t sequence_id;
    std::string request_id;
    std::string payment_id;
    std::string from_account;
    std::string to_account;
    std::string source_currency;
    std::string target_currency;
    int64_t source_amount;
    int64_t target_amount;
    int64_t fee_amount;
    int64_t timestamp;
};

struct AccountBalance {
    int64_t available;
    int64_t frozen;
    int64_t settled;
};

struct EngineEvent {
    uint64_t sequence_id;
    std::string event_id;
    std::string payment_id;
    std::string event_type;
    std::string result;
    int64_t timestamp;
};
```

#### 6.7.3 Engine Design Rules

```text
1. The hot path must not query PostgreSQL.
2. The hot path must not call KYC or Risk services.
3. The hot path should avoid JSON serialization.
4. All money values must be int64.
5. Use single-writer principle for ledger consistency.
6. Use lock-free queue where possible.
7. Use sequential WAL writes.
8. Generate snapshots for fast recovery.
9. Every command must be idempotent.
10. Every event must be replayable.
```

### 6.8 Settlement Service

The Settlement Service consumes engine events and writes the official accounting records.

Responsibilities:

- Consume `engine.executed`
- Generate double-entry ledger entries
- Create settlement batch
- Verify debit and credit balance
- Trigger blockchain audit recording
- Update payment status
- Handle compensation and reversal
- Generate reconciliation reports

Double-entry accounting rule:

```text
Every transaction must be balanced.
Total debit must equal total credit.
Ledger entries are append-only.
Reversal must be done through opposite entries.
No physical deletion is allowed.
```

Example payment:

```text
User A pays 100 USD.
Fee is 1 USD.
User B receives equivalent target currency.

Entry 1:
Debit:  User A available balance       101 USD
Credit: Platform clearing account      101 USD

Entry 2:
Debit:  Platform clearing account      100 USD
Credit: User B pending balance         target currency equivalent

Entry 3:
Debit:  Platform clearing account      1 USD
Credit: Platform fee income account    1 USD
```

### 6.9 Blockchain Audit Service

The Blockchain Audit Service records verifiable transaction proofs.

It should not store sensitive user data.

Responsibilities:

- Hash payment events
- Generate Merkle root for settlement batches
- Record transaction status hash
- Record ledger root hash
- Record settlement batch hash
- Submit proof to blockchain or internal hash chain
- Query audit proof

Recommended stages:

```text
Stage 1: Internal Hash Chain
Stage 2: Hyperledger Fabric
Stage 3: Aspira Consortium Chain based on Tendermint / CometBFT
```

#### 6.9.1 What Goes On-chain

| Data | On-chain | Description |
|---|---:|---|
| payment_id_hash | Yes | Hash of payment ID |
| sender_hash | Yes | Hash of sender ID |
| receiver_hash | Yes | Hash of receiver ID |
| amount_hash | Yes | Hash of amount data |
| event_type | Yes | Payment state event |
| settlement_batch_id | Yes | Batch tracking |
| ledger_root_hash | Yes | Merkle root of ledger entries |
| audit_signature | Yes | Signature from audit service |
| KYC raw data | No | Sensitive data |
| ID document | No | Sensitive data |
| bank card number | No | Sensitive data |
| personal address | No | Sensitive data |

#### 6.9.2 Chain Payment Record

```go
type ChainPaymentRecord struct {
    ChainTxID         string `json:"chain_tx_id"`
    PaymentIDHash     string `json:"payment_id_hash"`
    SenderHash        string `json:"sender_hash"`
    ReceiverHash      string `json:"receiver_hash"`
    AmountHash        string `json:"amount_hash"`
    CurrencyPair      string `json:"currency_pair"`
    EventType         string `json:"event_type"`
    SettlementBatchID string `json:"settlement_batch_id"`
    LedgerRootHash    string `json:"ledger_root_hash"`
    Timestamp         int64  `json:"timestamp"`
    Signature         string `json:"signature"`
}
```

#### 6.9.3 Chain State Flow

```text
PAYMENT_CREATED
  ↓
RISK_APPROVED
  ↓
FUNDS_FROZEN
  ↓
ENGINE_EXECUTED
  ↓
SETTLEMENT_COMPLETED
  ↓
PAYMENT_COMPLETED
```

---

## 7. Event-driven Architecture

Aspira Pay uses NATS JetStream as the primary event streaming layer.

### 7.1 Why NATS JetStream

NATS JetStream is suitable for V2 because it is:

- Lightweight
- Low-latency
- Easy to deploy
- Good for Go-based distributed systems
- Lower resource usage than Kafka
- Supports persistence and replay

Kafka or Redpanda can be used later for larger-scale deployments.

### 7.2 Event Topics

```text
payment.created
payment.kyc_checked
payment.risk_checked
payment.quote_locked
payment.submitted_to_engine
engine.command
engine.executed
engine.rejected
settlement.created
settlement.completed
chain.recorded
payment.completed
payment.failed
audit.event
```

### 7.3 Event Model

```go
type Event struct {
    EventID     string `json:"event_id"`
    EventType   string `json:"event_type"`
    AggregateID string `json:"aggregate_id"`
    PaymentID   string `json:"payment_id"`
    SequenceID  uint64 `json:"sequence_id"`
    PayloadHash string `json:"payload_hash"`
    CreatedAt   int64  `json:"created_at"`
}
```

### 7.4 Outbox Pattern

Payment Service must use the Outbox Pattern.

Within one database transaction:

```text
BEGIN
  INSERT payment_orders
  INSERT outbox_events
COMMIT
```

Then an Outbox Worker publishes events to NATS.

This prevents:

```text
Order created but event not published.
Event published but order not created.
```

---

## 8. Main Transaction Flow

### 8.1 Normal Payment Flow

```text
1. Client submits payment request.
2. API Gateway verifies JWT, signature, timestamp, nonce, and rate limit.
3. Payment Service checks request idempotency.
4. User Service checks user status.
5. KYC Service checks KYC approval.
6. Risk Service performs AML and risk checks.
7. FX Quote Service locks exchange rate.
8. Payment Service creates payment order.
9. Payment Service writes outbox event.
10. Outbox Worker publishes engine.command.
11. C++ Trading Engine receives command.
12. C++ Trading Engine checks in-memory balance.
13. C++ Trading Engine freezes funds.
14. C++ Trading Engine executes debit, credit, and fee calculation.
15. C++ Trading Engine writes WAL.
16. C++ Trading Engine publishes engine.executed.
17. Settlement Service consumes engine.executed.
18. Settlement Service writes double-entry ledger entries.
19. Settlement Service creates settlement batch.
20. Blockchain Audit Service calculates event hash and Merkle root.
21. Blockchain Audit Service writes audit proof to hash chain or blockchain.
22. Payment Service updates status to COMPLETED.
23. Admin dashboard can query the whole lifecycle.
```

### 8.2 Sequence Diagram

```text
Client
  │
  │ POST /payments
  ▼
API Gateway
  │
  ▼
Payment Service
  │ check idempotency
  │ check user
  │ check kyc
  │ check risk
  │ lock quote
  │ create order
  ▼
Outbox Events
  │
  ▼
NATS JetStream
  │
  ▼
C++ Trading Engine
  │ freeze funds
  │ execute payment
  │ write WAL
  ▼
Engine Event
  │
  ▼
Settlement Service
  │ write double-entry ledger
  │ create settlement batch
  ▼
Blockchain Audit Service
  │ write payment proof
  ▼
Payment Completed
```

---

## 9. Idempotency and Consistency

### 9.1 Idempotency

Every critical operation must include:

```text
request_id
idempotency_key
payment_id
event_id
sequence_id
```

Rules:

```text
Same request_id + same request_hash:
    return previous result.

Same request_id + different request_hash:
    reject request.

Same event_id:
    consume only once.

Same engine sequence_id:
    execute only once.
```

### 9.2 Saga Pattern

Distributed transaction flow:

```text
Create Payment
  ↓
Risk Check
  ↓
Freeze Funds
  ↓
Execute Payment
  ↓
Settlement
  ↓
Blockchain Audit
```

Compensation flow:

```text
Risk rejected:
    Payment status -> REJECTED

Engine failed:
    Release funds or mark as FAILED

Settlement failed:
    Retry settlement or send to manual review

Blockchain audit failed:
    Retry audit asynchronously
    Do not corrupt local accounting ledger
```

---

## 10. Database Design

### 10.1 Users

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    phone VARCHAR(64),
    status VARCHAR(32) NOT NULL,
    risk_level VARCHAR(32) NOT NULL DEFAULT 'LOW',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.2 Accounts

```sql
CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(64) UNIQUE NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    available_balance BIGINT NOT NULL DEFAULT 0,
    frozen_balance BIGINT NOT NULL DEFAULT 0,
    settled_balance BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'NORMAL',
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    UNIQUE(user_id, currency)
);
```

### 10.3 KYC Profiles

```sql
CREATE TABLE kyc_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    full_name_hash VARCHAR(255),
    nationality VARCHAR(64),
    document_type VARCHAR(64),
    document_hash VARCHAR(255),
    address_hash VARCHAR(255),
    kyc_status VARCHAR(32) NOT NULL,
    risk_level VARCHAR(32) NOT NULL DEFAULT 'LOW',
    reviewed_by VARCHAR(64),
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.4 Payment Orders

```sql
CREATE TABLE payment_orders (
    id BIGSERIAL PRIMARY KEY,
    payment_id VARCHAR(64) UNIQUE NOT NULL,
    request_id VARCHAR(128) UNIQUE NOT NULL,
    sender_user_id VARCHAR(64) NOT NULL,
    receiver_user_id VARCHAR(64) NOT NULL,
    source_currency VARCHAR(16) NOT NULL,
    target_currency VARCHAR(16) NOT NULL,
    source_amount BIGINT NOT NULL,
    target_amount BIGINT NOT NULL,
    fee_amount BIGINT NOT NULL,
    fx_rate NUMERIC(30, 12) NOT NULL,
    status VARCHAR(64) NOT NULL,
    risk_score INT DEFAULT 0,
    quote_id VARCHAR(64),
    chain_tx_id VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.5 Ledger Entries

```sql
CREATE TABLE ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    entry_id VARCHAR(64) UNIQUE NOT NULL,
    event_id VARCHAR(64) NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    account_id VARCHAR(64) NOT NULL,
    currency VARCHAR(16) NOT NULL,
    direction VARCHAR(16) NOT NULL,
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.6 Settlement Batches

```sql
CREATE TABLE settlement_batches (
    id BIGSERIAL PRIMARY KEY,
    batch_id VARCHAR(64) UNIQUE NOT NULL,
    currency VARCHAR(16) NOT NULL,
    total_debit BIGINT NOT NULL,
    total_credit BIGINT NOT NULL,
    entry_count INT NOT NULL,
    status VARCHAR(32) NOT NULL,
    ledger_root_hash VARCHAR(128),
    chain_tx_id VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.7 Idempotency Keys

```sql
CREATE TABLE idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(128) UNIQUE NOT NULL,
    request_hash VARCHAR(255) NOT NULL,
    response_body JSONB,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.8 Outbox Events

```sql
CREATE TABLE outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    aggregate_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    published BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.9 Chain Events

```sql
CREATE TABLE chain_events (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(64) UNIQUE NOT NULL,
    payment_id VARCHAR(64) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload_hash VARCHAR(128) NOT NULL,
    block_hash VARCHAR(128),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

### 10.10 Chain Blocks

```sql
CREATE TABLE chain_blocks (
    id BIGSERIAL PRIMARY KEY,
    block_height BIGINT NOT NULL UNIQUE,
    block_hash VARCHAR(128) NOT NULL,
    prev_hash VARCHAR(128) NOT NULL,
    merkle_root VARCHAR(128) NOT NULL,
    event_count INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);
```

---

## 11. Blockchain Audit Design

### 11.1 Hash Chain Model

For the first V2 version, an internal hash chain is recommended.

```text
event_hash = sha256(event_payload)
merkle_root = merkle(event_hash_1, event_hash_2, ...)
block_hash = sha256(prev_hash + merkle_root + timestamp)
```

This is lightweight, easy to deploy, and suitable for sandbox and prototype systems.

### 11.2 Merkle Batch Proof

Instead of writing every transaction directly to blockchain, Aspira Pay should batch events.

```text
1000 payment events
    ↓
Calculate event_hash for each event
    ↓
Build Merkle tree
    ↓
Generate ledger_root_hash
    ↓
Write root hash to blockchain
```

Benefits:

- Lower resource usage
- Higher throughput
- Lower blockchain cost
- Every transaction can still be verified through Merkle proof

### 11.3 Future Fabric Integration

Hyperledger Fabric can be integrated later.

Fabric responsibilities:

- Multi-organization audit
- Permissioned blockchain network
- Chaincode-based audit proof
- Settlement batch verification
- Regulator or partner node support

---

## 12. High Performance Design

### 12.1 API Layer

```text
Stateless API Gateway
Connection pooling
Redis-based rate limiting
Async logging
Request ID and trace ID
Horizontal scaling
```

### 12.2 Go Service Layer

```text
Goroutine concurrency
Database connection pool
Outbox Pattern
Saga Pattern
Batch inserts
Async event publishing
Cache for low-risk reference data
```

### 12.3 C++ Engine Layer

```text
Lock-free queue
Single-writer ledger core
Batch command processing
In-memory ledger
Sequential WAL
Snapshot recovery
CPU affinity
Avoid dynamic memory allocation in hot path
Avoid JSON in hot path
```

### 12.4 Database Layer

```text
Indexes on payment_id and request_id
Partition payment_orders by created_at
Partition ledger_entries by month or hash
Use PgBouncer
Batch insert ledger entries
Separate hot and cold data
```

### 12.5 Blockchain Layer

```text
Asynchronous blockchain write
Batch audit proof
Merkle root submission
Retry on failure
Do not block the main payment flow
```

---

## 13. High Availability Design

### 13.1 Service Replicas

Recommended deployment:

```text
API Gateway:              3 replicas
Payment Service:          3 replicas
User Service:             2 replicas
KYC Service:              2 replicas
Risk Service:             2 replicas
FX Quote Service:         2 replicas
Settlement Service:       2 replicas
Blockchain Audit Service: 2 replicas
C++ Engine:               Active + Standby
PostgreSQL:               Primary + Replica
Redis:                    Sentinel / Cluster
NATS:                     3-node JetStream cluster
Blockchain Nodes:         3 to 5 nodes
```

### 13.2 C++ Engine Active-Standby

The C++ engine should support active-standby mode.

```text
Active Engine
    ↓ WAL / Command Log / Snapshot
Standby Engine
```

Failover flow:

```text
1. Active engine fails.
2. Standby detects heartbeat timeout.
3. Standby reads the last sequence_id.
4. Standby replays missing commands.
5. Standby becomes active.
6. Engine Adapter switches traffic to new active engine.
```

### 13.3 PostgreSQL High Availability

```text
Primary-replica replication
WAL archiving
Daily full backup
Point-in-time recovery
Automatic failover
Regular restore tests
```

---

## 14. Security Design

### 14.1 API Security

```text
HTTPS
JWT
mTLS
API Key
HMAC request signature
Timestamp
Nonce
Rate limiting
IP allowlist
Replay attack prevention
```

Signature model:

```text
signature = HMAC_SHA256(
    method + path + timestamp + nonce + body_hash,
    api_secret
)
```

### 14.2 Data Security

```text
Use Argon2id or bcrypt for password hashing.
Encrypt KYC data.
Mask sensitive fields in logs.
Do not log ID numbers or bank card numbers.
Encrypt object storage.
Encrypt database backup.
Store keys in Vault or KMS.
```

### 14.3 Transaction Security

```text
Strict payment state machine.
Idempotency control.
Account status validation.
Balance validation.
Risk check before execution.
Manual review for large transactions.
Suspicious transaction freeze.
Reversal through accounting entries.
```

---

## 15. Observability

### 15.1 Metrics

Prometheus metrics:

```text
api_request_total
api_request_latency_ms
payment_created_total
payment_completed_total
payment_failed_total
risk_rejected_total
engine_command_latency_us
engine_tps
engine_error_total
settlement_lag_seconds
chain_submit_latency_ms
chain_submit_failed_total
```

### 15.2 Logs

Recommended logging stack:

```text
Go: zap / zerolog
C++: spdlog
Collection: Loki / Elasticsearch
```

Every log should include:

```text
trace_id
request_id
payment_id
event_id
user_id_hash
service_name
latency
status
error_code
```

### 15.3 Distributed Tracing

Use OpenTelemetry across:

```text
API Gateway
Payment Service
Risk Service
Engine Adapter
Settlement Service
Blockchain Audit Service
```

---

## 16. Deployment Architecture

### 16.1 Minimal Docker Compose Deployment

Minimal V2 deployment:

```text
aspira-api
aspira-engine-cpp
aspira-chain-service
postgres
redis
nats
prometheus
grafana
web-admin
```

### 16.2 Kubernetes Deployment

Namespace:

```text
aspira-pay
```

Deployments:

```text
api-gateway
user-service
kyc-service
risk-service
fx-service
payment-service
settlement-service
chain-service
engine-adapter
web-admin
```

StatefulSets:

```text
postgres
redis
nats
blockchain-node
cpp-engine-active
cpp-engine-standby
```

ConfigMaps:

```text
service-config
risk-rules
currency-config
```

Secrets:

```text
jwt-secret
db-password
api-signing-secret
chain-private-key
```

---

## 17. Project Structure

```text
aspira-pay/
├── backend-go/
│   ├── cmd/
│   │   ├── api-gateway/
│   │   ├── user-service/
│   │   ├── kyc-service/
│   │   ├── risk-service/
│   │   ├── fx-service/
│   │   ├── payment-service/
│   │   ├── settlement-service/
│   │   └── chain-service/
│   ├── internal/
│   │   ├── domain/
│   │   │   ├── user/
│   │   │   ├── kyc/
│   │   │   ├── risk/
│   │   │   ├── payment/
│   │   │   ├── ledger/
│   │   │   └── settlement/
│   │   ├── repository/
│   │   ├── service/
│   │   ├── transport/
│   │   ├── config/
│   │   ├── security/
│   │   └── observability/
│   ├── pkg/
│   │   ├── money/
│   │   ├── idgen/
│   │   ├── crypto/
│   │   ├── logger/
│   │   └── errors/
│   └── go.mod
│
├── engine-cpp/
│   ├── include/
│   │   ├── engine.hpp
│   │   ├── ledger.hpp
│   │   ├── command.hpp
│   │   ├── wal.hpp
│   │   └── publisher.hpp
│   ├── src/
│   │   ├── main.cpp
│   │   ├── engine.cpp
│   │   ├── ledger.cpp
│   │   ├── command_queue.cpp
│   │   ├── wal.cpp
│   │   └── publisher.cpp
│   ├── tests/
│   ├── proto/
│   └── CMakeLists.txt
│
├── proto/
│   ├── payment.proto
│   ├── engine.proto
│   ├── settlement.proto
│   └── chain.proto
│
├── web-admin/
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
│
├── deploy/
│   ├── docker-compose.yml
│   ├── k8s/
│   ├── helm/
│   └── scripts/
│
├── migrations/
│   ├── 001_init_users.sql
│   ├── 002_init_accounts.sql
│   ├── 003_init_payments.sql
│   ├── 004_init_ledger.sql
│   └── 005_init_chain.sql
│
├── docs/
│   ├── architecture.md
│   ├── api.md
│   ├── database.md
│   ├── engine.md
│   ├── blockchain.md
│   └── deployment.md
│
└── README.md
```

---

## 18. API Example

### 18.1 Create Payment

```http
POST /api/v2/payments
Content-Type: application/json
Authorization: Bearer <token>
Idempotency-Key: req_20260607_000001
```

Request:

```json
{
  "sender_user_id": "u_10001",
  "receiver_user_id": "u_20001",
  "source_currency": "USD",
  "target_currency": "JPY",
  "source_amount": 10000,
  "purpose": "family_support",
  "country_from": "US",
  "country_to": "JP"
}
```

Response:

```json
{
  "payment_id": "pay_20260607_000001",
  "status": "CREATED",
  "source_amount": 10000,
  "target_amount": 1560000,
  "fee_amount": 100,
  "fx_rate": "156.000000000000",
  "created_at": 1780848000
}
```

---

## 19. Performance Targets

| Metric | Target |
|---|---:|
| API P95 latency | < 100 ms |
| Risk check P95 latency | < 50 ms |
| C++ engine internal latency | < 1 ms |
| Single-node engine TPS | 10,000+ |
| Message queue latency | < 10 ms |
| Local ledger write latency | < 100 ms |
| Blockchain batch confirmation | Seconds to minutes |
| System availability | 99.9% |
| Audit trace coverage | 100% |
| Idempotency coverage | 100% |

---

## 20. Development Roadmap

### Phase 1: Core Payment Skeleton

Goal:

```text
User -> Account -> Payment Order
```

Tasks:

```text
1. Create Go project structure.
2. Create PostgreSQL schema.
3. Implement User Service.
4. Implement Account Service.
5. Implement Payment Service.
6. Implement API Gateway.
7. Implement basic Web Admin.
```

### Phase 2: KYC and Risk

Goal:

```text
Payment can only be created after KYC and risk checks.
```

Tasks:

```text
1. Implement KYC state machine.
2. Implement user risk level.
3. Implement blacklist rules.
4. Implement transaction limits.
5. Implement country restrictions.
6. Implement manual review queue.
```

### Phase 3: C++ Engine

Goal:

```text
C++ engine executes transaction commands.
```

Tasks:

```text
1. Define engine.proto.
2. Implement command queue.
3. Implement in-memory ledger.
4. Implement freeze, execute, release, and refund.
5. Implement WAL.
6. Publish engine.executed event.
7. Run TPS and latency benchmark.
```

### Phase 4: Double-entry Ledger

Goal:

```text
Every transaction is recorded as balanced accounting entries.
```

Tasks:

```text
1. Implement ledger_entries.
2. Implement settlement_batches.
3. Verify debit equals credit.
4. Implement reversal.
5. Implement reconciliation reports.
```

### Phase 5: Blockchain Audit

Goal:

```text
Transaction process becomes verifiable and tamper-evident.
```

Tasks:

```text
1. Implement internal Hash Chain.
2. Implement Merkle Tree.
3. Implement chain_events.
4. Submit batch root hash.
5. Integrate Hyperledger Fabric later.
6. Implement audit query API.
```

### Phase 6: High Availability and Observability

Goal:

```text
The system can run stably in a distributed environment.
```

Tasks:

```text
1. Docker Compose deployment.
2. Prometheus metrics.
3. Grafana dashboard.
4. Loki logs.
5. OpenTelemetry tracing.
6. C++ Engine Active-Standby.
7. PostgreSQL backup and recovery.
8. NATS JetStream cluster.
```

---

## 21. Minimal Viable Mature Version

The first mature version should include:

```text
Go API Gateway
Go User Service
Go KYC Service
Go Risk Service
Go Payment Service
C++ Trading Engine
Go Settlement Service
Go Blockchain Audit Service
PostgreSQL
Redis
NATS JetStream
Web Admin
```

Minimal transaction path:

```text
Register user
  ↓
KYC approved
  ↓
Add sandbox balance
  ↓
Create cross-border payment
  ↓
Risk approved
  ↓
FX quote locked
  ↓
Send engine.command
  ↓
C++ engine executes transaction
  ↓
Settlement writes double-entry ledger
  ↓
Blockchain service writes hash chain proof
  ↓
Payment completed
```

---

## 22. Final Design Principles

Aspira Pay should always follow these principles:

```text
1. KYC and risk control must happen before payment execution.
2. Every payment must follow a strict state machine.
3. The C++ engine must only handle high-performance execution.
4. Go services handle orchestration, compliance, and distributed coordination.
5. Blockchain stores audit proofs, not sensitive raw data.
6. PostgreSQL is the source of truth for the business ledger.
7. The ledger must use double-entry accounting.
8. Every request must be idempotent.
9. Every event must be replayable.
10. Every failure must be retryable or compensatable.
11. Blockchain confirmation should be asynchronous.
12. The system must support audit, reconciliation, reversal, and traceability.
13. Logs must never expose sensitive information.
14. All critical services must have health checks.
15. All critical flows must include trace_id, request_id, payment_id, and event_id.
```

---

## 23. Conclusion

The best architecture for Aspira Pay is:

> Go microservices handle payment, KYC, risk control, and settlement orchestration.  
> The C++ trading engine handles high-performance transaction execution.  
> PostgreSQL stores the official double-entry accounting ledger.  
> NATS moves events across the distributed system.  
> Blockchain records trusted audit proofs for the transaction process.

This design turns Aspira Pay from a simple payment demo into a mature distributed transaction system.

The next recommended step is to implement:

```text
Payment State Machine
    ↓
Idempotency
    ↓
NATS Event Flow
    ↓
C++ Engine MVP
    ↓
Double-entry Ledger
    ↓
Hash Chain Audit
```

Once this core path is stable, the system can evolve into a production-grade cross-border payment and clearing platform.
