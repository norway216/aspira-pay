# Aspira Pay C++ Trading Engine Optimization Plan

**Version:** 1.0  
**Author:** Aspira Studio  
**Target System:** Aspira Pay V2 Distributed Payment, Clearing, and Trading System  
**Core Focus:** High-performance, low-latency, deterministic, replayable, and recoverable C++ transaction execution engine  

---

## 1. Executive Summary

The C++ Trading Engine is the performance-critical execution core of Aspira Pay.

In the overall Aspira Pay architecture:

- Go microservices handle payment orchestration, KYC, AML, FX quote, and settlement coordination.
- PostgreSQL stores the official double-entry ledger.
- NATS JetStream handles event-driven communication.
- Blockchain records trusted audit proofs.
- The C++ engine executes high-performance transaction commands.

The C++ engine should not behave like a normal backend service. It should not query PostgreSQL, call KYC services, call risk services, or perform complex business orchestration in the hot path.

The optimized C++ engine should be designed as:

```text
In-memory deterministic state machine
    + ordered command processor
    + high-speed account ledger
    + WAL-based recovery
    + asynchronous event publisher
```

The main optimization target is:

```text
Receive approved transaction commands
    -> validate sequence and idempotency
    -> update in-memory account state
    -> append WAL
    -> emit deterministic execution events
```

---

## 2. Optimization Objectives

### 2.1 Performance Objectives

| Metric | Target |
|---|---:|
| Internal execution latency | < 1 ms |
| Single-core pure engine TPS | 100,000+ target for benchmark mode |
| Full engine service TPS | 10,000+ for V2 sandbox |
| Queue wait latency | measurable and bounded |
| WAL append latency | microsecond-level in batch mode |
| Event publishing impact on core thread | zero blocking |
| Recovery time | seconds to minutes depending on snapshot size |

### 2.2 Engineering Objectives

The C++ engine must be:

- Deterministic
- Idempotent
- Replayable
- Recoverable
- Low-latency
- Low-resource
- Observable
- Easy to benchmark
- Safe under duplicate messages
- Safe under process crash
- Safe under partial WAL writes

---

## 3. Current Baseline Design

The current design defines the C++ Trading Engine as the high-performance transaction execution core.

Its main responsibilities are:

```text
Receive transaction commands
Validate sequence ID
Validate idempotency key
Check account balance from in-memory ledger
Freeze funds
Debit source account
Credit target account
Calculate fee
Write WAL
Generate engine events
Publish execution result
```

The current internal architecture is:

```text
Engine Gateway
    -> Command Decoder
    -> Lock-free Command Queue
    -> Core Engine Loop
    -> In-memory Ledger
    -> WAL Log
    -> Event Publisher
```

This direction is correct. The optimization work should improve the implementation details around memory layout, serialization, queueing, WAL, snapshotting, idempotency, and event publishing.

---

## 4. Core Design Principle

The most important rule:

> The hot path must be isolated from slow systems.

The C++ engine hot path must not:

```text
Query PostgreSQL
Call KYC Service
Call Risk Service
Call FX Quote Service
Write blockchain directly
Perform JSON serialization
Wait for NATS publish acknowledgment
Perform heavy logging per transaction
Perform dynamic memory allocation per transaction
```

The hot path should only do:

```text
Decode command
Validate command
Check idempotency
Check balance
Update in-memory ledger
Append WAL
Enqueue event
Return execution result
```

---

## 5. Optimized Engine Architecture

```text
┌──────────────────────────────────────────────┐
│              Command Receiver                │
│        NATS Consumer / gRPC Stream / TCP       │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│              Command Decoder                 │
│     Protobuf / FlatBuffers / Binary Protocol  │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│             MPSC Ring Buffer                 │
│      Pre-allocated / Lock-free / Bounded      │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│              Engine Core Thread              │
│      Single Writer / Ordered / Deterministic  │
└──────────────┬───────────────┬───────────────┘
               │               │
               ▼               ▼
┌──────────────────────┐ ┌──────────────────────┐
│ In-memory Ledger     │ │ Idempotency Cache     │
│ uint64_t -> Balance  │ │ request_id -> result  │
└──────────────────────┘ └──────────────────────┘
               │
               ▼
┌──────────────────────────────────────────────┐
│                  WAL Buffer                  │
│        Batch Flush / Checksum / fsync policy  │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│                Event Ring Buffer             │
│        SPSC Queue to Publisher Thread         │
└───────────────────────┬──────────────────────┘
                        ▼
┌──────────────────────────────────────────────┐
│              Event Publisher                 │
│        NATS / Kafka / Settlement Service      │
└──────────────────────────────────────────────┘
```

---

## 6. Command Model Optimization

### 6.1 Problem with String-heavy Commands

A business-friendly command model may look like this:

```cpp
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
```

This is easy to understand but not optimal for the hot path.

Problems:

```text
Dynamic memory allocation
String copying
Poor cache locality
Variable-length fields
Higher serialization cost
Higher comparison cost
```

### 6.2 Optimized Fixed-field Command

Use numeric IDs and fixed-size fields in the engine hot path:

```cpp
struct PaymentCommandFast {
    uint64_t sequence_id;
    uint64_t request_id_hash;
    uint64_t payment_id;
    uint64_t from_account_id;
    uint64_t to_account_id;
    uint16_t source_currency;
    uint16_t target_currency;
    int64_t source_amount;
    int64_t target_amount;
    int64_t fee_amount;
    int64_t timestamp_ns;
};
```

### 6.3 Currency Code Mapping

Use ISO 4217 numeric currency codes:

```text
USD -> 840
EUR -> 978
JPY -> 392
CNY -> 156
HKD -> 344
GBP -> 826
SGD -> 702
```

### 6.4 Account ID Mapping

Do not use string account IDs inside the engine hot path.

External account ID:

```text
acc_20260607_000001
```

Internal engine account ID:

```text
uint64_t account_id
```

The mapping can be handled by Go services or an Engine Adapter before the command reaches the C++ engine.

---

## 7. Serialization Optimization

### 7.1 Avoid JSON in Hot Path

JSON should not be used between the engine adapter and the C++ core.

Problems with JSON:

```text
High parsing cost
High allocation cost
Large payload size
Weak schema control
Poor cache efficiency
```

### 7.2 Recommended Protocol Roadmap

| Stage | Protocol | Use Case |
|---|---|---|
| V2 MVP | Protobuf | Easy integration with Go and C++ |
| V2 Optimized | FlatBuffers | Lower latency and zero-copy style access |
| V3 Advanced | Custom binary protocol | Maximum performance after protocol is stable |

### 7.3 Recommended V2 Choice

Use Protobuf for the first optimized version.

```text
Go Payment Service / Engine Adapter
    -> Protobuf
    -> NATS or gRPC stream
    -> C++ Engine
```

Protobuf is a good balance between:

```text
Performance
Schema safety
Go/C++ ecosystem support
Development speed
Maintainability
```

---

## 8. Queue Model Optimization

### 8.1 Recommended Queue Model

Use an MPSC queue:

```text
Multiple Producer Single Consumer
```

Architecture:

```text
Network Thread 1 ┐
Network Thread 2 ├──> MPSC Ring Buffer ──> Engine Core Thread
Network Thread 3 ┘
```

### 8.2 Why Single Consumer

The Engine Core Thread should be a single writer to the in-memory ledger.

Benefits:

```text
Deterministic transaction order
No per-account locks in the basic version
Simple replay
Simple recovery
No deadlock
Stable latency
```

### 8.3 Ring Buffer Requirements

The command queue should be:

```text
Pre-allocated
Bounded
Lock-free or low-lock
Cache-friendly
Backpressure-aware
Observable through queue depth metrics
```

### 8.4 Backpressure Policy

When the queue is full, the engine should not crash or allocate memory dynamically.

Recommended policies:

```text
Reject new command with ENGINE_BUSY
Apply upstream rate limiting
Expose queue_depth and queue_reject_total metrics
Let Go layer retry with idempotency key
```

---

## 9. In-memory Ledger Optimization

### 9.1 Avoid String Keys

Do not use:

```cpp
std::unordered_map<std::string, AccountBalance>
```

Use:

```cpp
std::unordered_map<uint64_t, AccountBalance>
```

or a more cache-friendly hash map:

```text
absl::flat_hash_map
robin_hood::unordered_flat_map
ska::flat_hash_map
```

### 9.2 Account Balance Layout

Recommended structure:

```cpp
struct AccountBalance {
    int64_t available;
    int64_t frozen;
    int64_t settled;
    int64_t version;
    uint64_t updated_at_ns;
};
```

For extremely hot accounts, cache line alignment can be considered:

```cpp
struct alignas(64) HotAccountBalance {
    int64_t available;
    int64_t frozen;
    int64_t settled;
    int64_t version;
    uint64_t updated_at_ns;
};
```

### 9.3 Balance Update Rules

The ledger must enforce:

```text
No negative available balance
No overflow
No unknown currency
No frozen account execution
No duplicate payment execution
Strict state transition
```

Example logic:

```cpp
if (balance.available < command.source_amount + command.fee_amount) {
    return EngineErrorCode::INSUFFICIENT_FUNDS;
}

balance.available -= command.source_amount + command.fee_amount;
balance.frozen += command.source_amount + command.fee_amount;
balance.version++;
```

---

## 10. Memory Allocation Optimization

### 10.1 Avoid Hot-path Allocation

Avoid in the hot path:

```text
new/delete
malloc/free
std::string temporary creation
std::vector dynamic growth
std::shared_ptr reference counting
exceptions for normal errors
per-transaction logging allocation
```

### 10.2 Recommended Techniques

Use:

```text
Pre-allocated ring buffers
Object pools
Fixed-size command buffers
Fixed-size event buffers
string_view for non-owning data
unique_ptr instead of shared_ptr where ownership is clear
No exceptions in hot path
```

### 10.3 Object Pool Example

```cpp
template <typename T, size_t N>
class ObjectPool {
public:
    T* acquire();
    void release(T* obj);

private:
    std::array<T, N> pool_;
    std::atomic<size_t> index_;
};
```

---

## 11. WAL Optimization

### 11.1 Why WAL Is Required

WAL is required for:

```text
Crash recovery
Command replay
Audit support
Active-standby replication
Deterministic state rebuild
```

### 11.2 Avoid Per-transaction fsync

Bad design:

```text
Process one transaction
    -> fsync once
```

This causes high latency and low throughput.

### 11.3 Batch Flush Policy

Recommended design:

```text
Append command/event to WAL buffer
Flush every N records or every X microseconds
fsync according to configured durability level
```

Example configuration:

```yaml
wal:
  flush_policy: batch
  batch_size: 1000
  flush_interval_us: 1000
  fsync_policy: every_flush
```

### 11.4 WAL Record Format

```cpp
struct WalRecordHeader {
    uint64_t sequence_id;
    uint32_t record_type;
    uint32_t payload_size;
    uint64_t timestamp_ns;
    uint64_t checksum;
};
```

Payload:

```text
PaymentCommandFast
EngineEvent
SnapshotMetadata
```

### 11.5 WAL Checksum

Every WAL record must include a checksum.

Benefits:

```text
Detect partial writes
Detect corrupted records
Stop replay safely
Improve recovery reliability
```

---

## 12. Snapshot Optimization

### 12.1 Why Snapshot Is Required

If the engine only has WAL, restart may require replaying millions or billions of records.

Snapshot reduces recovery time.

### 12.2 Snapshot Content

A snapshot should include:

```text
Account balance map
Last sequence ID
Idempotency cache state
Engine configuration version
Snapshot timestamp
Checksum
```

### 12.3 Snapshot Strategy

Recommended:

```text
Light snapshot every 1 minute
Full snapshot every 10 minutes
Keep WAL records only after the latest stable snapshot
```

### 12.4 Recovery Flow

```text
1. Load latest valid snapshot.
2. Verify snapshot checksum.
3. Read WAL records after snapshot sequence ID.
4. Verify WAL checksums.
5. Replay commands/events in sequence order.
6. Rebuild in-memory ledger.
7. Rebuild idempotency cache.
8. Start receiving new commands.
```

---

## 13. Idempotency Optimization

### 13.1 Why Engine-level Idempotency Is Required

The Go layer already performs idempotency checks, but the C++ engine must also protect itself.

Reason:

```text
NATS may deliver duplicate messages.
Go services may retry.
Network failures may cause uncertain delivery.
Engine failover may replay commands.
```

### 13.2 Recommended Dedup State

The engine should maintain:

```text
request_id_hash -> execution result
payment_id -> last engine status
sequence_id -> executed record
```

### 13.3 Dedup Cache Options

| Option | Description |
|---|---|
| HashMap | Fast in-memory dedup |
| LRU Cache | Limit memory usage |
| Bloom Filter + HashMap | Fast negative check |
| RocksDB | Persistent dedup state |
| Snapshot + WAL | Recommended baseline |

### 13.4 Dedup Retention

For V2:

```text
Keep recent request IDs for 24 hours
or keep last 10 million request IDs
```

This should be configurable.

---

## 14. Event Publishing Optimization

### 14.1 Avoid Blocking Publish in Core Thread

Bad design:

```text
Execute transaction
    -> publish to NATS
    -> wait for ACK
    -> process next transaction
```

This makes the engine latency depend on network and NATS.

### 14.2 Recommended Async Event Publisher

```text
Engine Core Thread
    -> SPSC Event Queue
    -> Publisher Thread
    -> NATS JetStream
```

### 14.3 Event Queue Requirements

The event queue should be:

```text
Pre-allocated
Bounded
SPSC
Observable
Backpressure-aware
```

### 14.4 Publisher Failure Handling

If publisher fails:

```text
Do not rollback already executed in-memory state immediately.
Keep events in WAL.
Retry publish from event log.
Expose publish lag metric.
If lag exceeds threshold, apply backpressure to command intake.
```

---

## 15. Network Model Optimization

### 15.1 V2 Recommended Model

For V2, use:

```text
Go Payment Service
    -> NATS JetStream
    -> C++ Engine Consumer
```

This is simple and stable.

### 15.2 Lower-latency Model

For lower latency:

```text
Go Engine Adapter
    -> gRPC streaming
    -> C++ Engine
```

### 15.3 Advanced Model

For future extreme performance:

```text
Go Engine Adapter
    -> Custom binary TCP protocol
    -> C++ Engine
```

### 15.4 Recommended Separation

```text
Business events: NATS
High-frequency engine commands: gRPC stream or TCP
Audit events: NATS
```

For V2, keeping NATS for both commands and events is acceptable.

---

## 16. Thread Model Optimization

### 16.1 Recommended Threads

```text
Thread 1: Command Receiver
Thread 2: Engine Core
Thread 3: WAL Writer
Thread 4: Event Publisher
Thread 5: Snapshot Writer
Thread 6: Metrics Reporter
```

### 16.2 CPU Affinity

The Engine Core Thread should avoid CPU contention.

Example Linux command:

```bash
taskset -c 2 ./aspira-engine
```

Recommended:

```text
Engine Core: dedicated CPU core
WAL Writer: dedicated or low-contention CPU core
Publisher: separate CPU core if throughput is high
```

### 16.3 Avoid Oversubscription

Do not create too many threads.

The engine should use a small, predictable number of threads.

---

## 17. Error Handling Optimization

### 17.1 Avoid Exceptions for Normal Errors

Do not use exceptions for expected business errors.

Use error codes:

```cpp
enum class EngineErrorCode {
    OK = 0,
    DUPLICATED_REQUEST = 1,
    INVALID_SEQUENCE = 2,
    ACCOUNT_NOT_FOUND = 3,
    INSUFFICIENT_FUNDS = 4,
    ACCOUNT_FROZEN = 5,
    WAL_WRITE_FAILED = 6,
    INTERNAL_ERROR = 7
};
```

### 17.2 Result Model

```cpp
struct EngineExecutionResult {
    uint64_t sequence_id;
    uint64_t payment_id;
    EngineErrorCode code;
    int64_t latency_ns;
};
```

---

## 18. Logging Optimization

### 18.1 Do Not Log Every Successful Transaction

Bad design:

```cpp
logger.info("payment executed: {}", payment_id);
```

This can destroy throughput.

### 18.2 Recommended Logging Strategy

```text
INFO: startup, shutdown, configuration changes
WARN: recoverable abnormal conditions
ERROR: execution failure, WAL failure, state corruption
DEBUG: only in development or benchmark mode
```

### 18.3 Audit Events Are Not Logs

Do not use logs as the source of truth.

Use:

```text
Engine events
WAL records
Settlement ledger
Blockchain audit proof
```

Logs are for diagnosis, not accounting.

---

## 19. Metrics and Observability

### 19.1 Required Metrics

The engine should expose:

```text
engine_tps
engine_latency_p50
engine_latency_p90
engine_latency_p95
engine_latency_p99
engine_queue_depth
engine_queue_reject_total
wal_flush_latency
wal_pending_records
event_publish_lag
event_publish_failed_total
account_count
dedup_cache_size
snapshot_duration
replay_duration
```

### 19.2 Latency Breakdown

Track:

```text
decode_latency
queue_wait_latency
execute_latency
wal_append_latency
event_enqueue_latency
total_latency
```

Without latency breakdown, optimization becomes guesswork.

---

## 20. Account Sharding Strategy

### 20.1 Start Without Sharding

V2 should start with one engine core:

```text
Engine Core 1
    -> all accounts
```

Benefits:

```text
Simple
Deterministic
Easy to test
Easy to recover
Easy to replay
```

### 20.2 Add Sharding After Bottleneck

When one core becomes a bottleneck:

```text
Engine Shard 0 -> account_id % N == 0
Engine Shard 1 -> account_id % N == 1
Engine Shard 2 -> account_id % N == 2
Engine Shard 3 -> account_id % N == 3
```

Each shard has its own single-writer core.

### 20.3 Cross-shard Transfer

Cross-shard transfer requires careful design.

Example:

```text
From account: Shard 1
To account:   Shard 3
```

Recommended flow:

```text
1. Shard 1 freezes source funds.
2. Shard 1 emits TRANSFER_PREPARED.
3. Shard 3 credits target account.
4. Both shards emit final settlement events.
```

For V2, avoid cross-shard complexity unless required.

---

## 21. High Availability Optimization

### 21.1 Active-Standby Engine

The engine should support:

```text
Active Engine
    -> WAL / Command Log / Snapshot
Standby Engine
```

### 21.2 Failover Flow

```text
1. Active engine fails.
2. Standby detects heartbeat timeout.
3. Standby loads latest snapshot.
4. Standby replays WAL after snapshot.
5. Standby checks last sequence ID.
6. Standby becomes active.
7. Engine Adapter redirects commands to new active engine.
```

### 21.3 Standby Sync Methods

Options:

```text
WAL file replication
NATS command replay
Shared persistent volume
Object storage snapshot replication
Dedicated replication stream
```

Recommended V2 baseline:

```text
Snapshot + WAL + command replay
```

---

## 22. Benchmark Plan

### 22.1 Benchmark Layer 1: Pure Engine Benchmark

No network, no NATS, no PostgreSQL.

```text
Generate 1 million PaymentCommandFast records in memory
Feed directly into Engine Core
Measure pure execution TPS and latency
```

Purpose:

```text
Measure the engine core itself
```

### 22.2 Benchmark Layer 2: Engine Service Benchmark

```text
Go Engine Adapter
    -> Protobuf/gRPC or NATS
    -> C++ Engine
```

Purpose:

```text
Measure serialization and transport overhead
```

### 22.3 Benchmark Layer 3: Full System Benchmark

```text
API Gateway
    -> Payment Service
    -> NATS
    -> C++ Engine
    -> Settlement Service
    -> PostgreSQL
```

Purpose:

```text
Measure real system throughput and bottlenecks
```

### 22.4 Benchmark Metrics

Collect:

```text
TPS
P50 latency
P90 latency
P95 latency
P99 latency
Max latency
CPU usage
Memory usage
Queue depth
WAL latency
Publish lag
Error rate
Recovery time
```

---

## 23. Optimization Priority

### 23.1 Priority 1: Correctness and Determinism

```text
Do not query database in hot path
Do not call KYC/Risk in hot path
Use single-writer ledger core
Use sequence_id
Use engine-level idempotency
Write WAL
Emit replayable events
```

### 23.2 Priority 2: Basic Performance

```text
Replace string IDs with uint64_t IDs
Use Protobuf instead of JSON
Use uint64_t hash map keys
Use MPSC Ring Buffer
Use async event publishing
Use WAL batch flush
```

### 23.3 Priority 3: Stability and Recovery

```text
Snapshot
WAL checksum
Crash recovery
Replay test
Active-standby
Publisher retry
Chaos testing
```

### 23.4 Priority 4: Scalability

```text
Account sharding
Multiple engine cores
Cross-shard transaction protocol
Multi-active architecture
```

---

## 24. Recommended V2 Engine MVP

The first optimized C++ engine should include:

```text
Single process
Single Engine Core Thread
MPSC Command Queue
In-memory Ledger
Protobuf Command Model
WAL Append
Snapshot
Async Event Publisher
NATS Output
Prometheus Metrics
```

Supported commands:

```text
FREEZE_FUNDS
EXECUTE_PAYMENT
RELEASE_FUNDS
REFUND_PAYMENT
```

Supported events:

```text
ENGINE_ACCEPTED
FUNDS_FROZEN
PAYMENT_EXECUTED
PAYMENT_REJECTED
FUNDS_RELEASED
PAYMENT_REFUNDED
```

---

## 25. Anti-patterns to Avoid

Do not design the C++ engine like this:

```text
C++ Engine queries PostgreSQL for every transaction.
C++ Engine calls Risk Service per command.
C++ Engine parses JSON in the hot path.
C++ Engine uses string IDs everywhere.
C++ Engine fsyncs every transaction.
C++ Engine waits for NATS ACK before next transaction.
C++ Engine logs every successful payment.
C++ Engine lets multiple threads update account balance without deterministic ordering.
C++ Engine has no WAL.
C++ Engine has no snapshot.
C++ Engine has no idempotency cache.
```

These patterns will destroy performance, stability, or recoverability.

---

## 26. Final Recommended Architecture

The optimized C++ engine should be:

```text
Fixed-field command model
    + Protobuf or FlatBuffers
    + MPSC Ring Buffer
    + Single-writer ledger core
    + uint64_t account IDs
    + in-memory balance map
    + engine-level idempotency
    + WAL batch flush
    + snapshot recovery
    + async event publisher
    + Prometheus metrics
    + active-standby failover
```

This design turns the engine from a normal service into a high-performance financial execution core.

---

## 27. Conclusion

The most important optimization direction is not simply to make C++ code faster.

The real goal is to make the engine:

```text
Deterministic
Replayable
Recoverable
Low-latency
Low-resource
Safe under duplicate messages
Safe under crash recovery
Observable under production load
```

The recommended first implementation is:

```text
Single-core deterministic engine
    -> MPSC queue
    -> in-memory ledger
    -> WAL
    -> snapshot
    -> async event publisher
```

After this version is stable and benchmarked, account sharding and multi-engine scaling can be added.

The correct evolution path is:

```text
Correctness first
    -> deterministic execution
    -> WAL and replay
    -> benchmark
    -> low-latency optimization
    -> active-standby
    -> account sharding
```
