// Aspira Pay V2 — Optimized Engine Core (§24)
// Architecture doc §24: Recommended V2 Engine MVP.
//
// Optimized architecture (§5):
//   Command Receiver → MPSC Ring Buffer → Engine Core Thread
//     → In-memory Ledger (uint64 keys)
//     → WAL Buffer (batch flush + checksum)
//     → SPSC Event Ring Buffer → Async Publisher Thread
//
// Thread model (§16):
//   Thread 1: Command Receiver (NATS consumer)
//   Thread 2: Engine Core (single writer, deterministic)
//   Thread 3: WAL Writer (batch flush)
//   Thread 4: Event Publisher (async NATS output)
//   Thread 5: Snapshot Writer (periodic)
//   Thread 6: Metrics Reporter (Prometheus scrape)

#pragma once

#include "Types.h"
#include "Ledger.h"
#include "CommandQueue.h"
#include "WAL.h"
#include "Publisher.h"
#include "EventRingBuffer.h"
#include "EngineMetrics.h"
#include "ObjectPool.h"
#include <memory>
#include <atomic>
#include <thread>
#include <unordered_set>
#include <mutex>

namespace aspira {
namespace engine {

class Engine {
public:
    Engine();
    ~Engine();

    // ── Lifecycle ────────────────────────────

    // Initialize with WAL path, snapshot interval, and optional config
    bool init(const std::string& wal_path, int snapshot_interval_sec = 300,
              const WalConfig& wal_config = WalConfig{});

    // Start all engine threads
    void start();

    // Graceful shutdown: final snapshot, flush WAL, drain events
    void stop();

    // ── Command Intake (§8) ──────────────────

    // Submit a string-based command (from Go adapter, backward-compatible)
    bool submit(const PaymentCommand& cmd);

    // Submit a fast fixed-field command (§6.2)
    bool submit_fast(const PaymentCommandFast& cmd);

    // ── Snapshots (§12) ──────────────────────

    // Take a full snapshot of current state
    void take_snapshot();

    // Restore from WAL on startup
    bool restore_from_wal();

    // ── Accessors ────────────────────────────

    Ledger& ledger() { return ledger_; }
    Publisher& publisher() { return publisher_; }
    EngineMetrics& metrics() { return metrics_; }

    bool is_running() const { return running_; }

private:
    // ── Threads (§16) ────────────────────────
    void core_loop();        // Thread 2: Engine Core
    void wal_flush_loop();   // Thread 3: WAL Writer
    void event_publish_loop(); // Thread 4: Event Publisher
    void snapshot_loop();    // Thread 5: Snapshot Writer
    void metrics_loop();     // Thread 6: Metrics Reporter

    // ── Command Processing (§17) ─────────────
    EngineErrorCode process_command_fast(const PaymentCommandFast& cmd);
    EngineExecutionResult build_result(const PaymentCommandFast& cmd, EngineErrorCode code, uint64_t latency_ns);

    // ── Command Handlers ─────────────────────
    EngineErrorCode handle_freeze(const PaymentCommandFast& cmd);
    EngineErrorCode handle_execute(const PaymentCommandFast& cmd);
    EngineErrorCode handle_release(const PaymentCommandFast& cmd);
    EngineErrorCode handle_refund(const PaymentCommandFast& cmd);
    EngineErrorCode handle_settlement_batch(const PaymentCommandFast& cmd);

    // ── Event Emission ───────────────────────
    void emit_event(const PaymentCommandFast& cmd, const std::string& event_type,
                    const std::string& result);

    // ── Dedup (§13) ──────────────────────────
    bool is_duplicate(uint64_t request_id_hash);
    void record_dedup_result(uint64_t request_id_hash, EngineErrorCode code);

    // ── Adapter Layer (§6.4) ─────────────────
    PaymentCommandFast convert_to_fast(const PaymentCommand& cmd);

    // ── Components ───────────────────────────
    Ledger ledger_;
    CommandQueue queue_;
    std::unique_ptr<WAL> wal_;
    Publisher publisher_;
    EventRingBuffer event_ring_{4096};
    EngineMetrics metrics_;

    // ── Threads ──────────────────────────────
    std::unique_ptr<std::thread> core_thread_;
    std::unique_ptr<std::thread> wal_thread_;
    std::unique_ptr<std::thread> publish_thread_;
    std::unique_ptr<std::thread> snapshot_thread_;
    std::unique_ptr<std::thread> metrics_thread_;
    std::atomic<bool> running_{false};

    // ── Dedup Cache (§13.3) ──────────────────
    // HashMap + LRU for 24h dedup window
    struct DedupEntry {
        uint64_t request_id_hash;
        EngineErrorCode result;
        uint64_t timestamp_ns;
    };
    static constexpr size_t MAX_DEDUP_SIZE = 10'000'000;  // §13.4
    std::unordered_map<uint64_t, DedupEntry> dedup_cache_;
    std::mutex dedup_mutex_;
    uint64_t dedup_evictions_{0};

    // ── State ────────────────────────────────
    std::atomic<uint64_t> sequence_{0};
    int snapshot_interval_sec_{300};
    WalConfig wal_config_{};
};

} // namespace engine
} // namespace aspira
