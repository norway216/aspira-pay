// Aspira Pay V2 — Core Engine Loop
// Single Writer Principle: processes commands sequentially.
// Architecture doc §4.7.1: Core Engine Loop.
// Architecture doc §6.7.3: Engine Design Rules.

#pragma once

#include "Types.h"
#include "Ledger.h"
#include "CommandQueue.h"
#include "WAL.h"
#include "Publisher.h"
#include <memory>
#include <atomic>
#include <thread>
#include <unordered_set>

namespace aspira {
namespace engine {

class Engine {
public:
    Engine();
    ~Engine();

    // Initialize engine with WAL path and snapshot interval
    bool init(const std::string& wal_path, int snapshot_interval_sec = 300);

    // Start the engine processing loop (runs in background thread)
    void start();

    // Stop the engine gracefully
    void stop();

    // Submit a command to the engine (producer-side, thread-safe)
    bool submit(const PaymentCommand& cmd);

    // Take a snapshot of current ledger state to WAL
    void take_snapshot();

    // Restore from WAL on startup
    bool restore_from_wal();

    // Get ledger reference (for balance queries)
    Ledger& ledger() { return ledger_; }

    // Get publisher reference
    Publisher& publisher() { return publisher_; }

    // Engine statistics
    uint64_t commands_processed() const { return commands_processed_; }
    uint64_t commands_rejected() const { return commands_rejected_; }
    uint64_t commands_duplicated() const { return commands_duplicated_; }
    uint64_t events_published() const { return publisher_.total_published(); }
    uint64_t last_snapshot_seq() const { return last_snapshot_seq_; }
    bool is_running() const { return running_; }

private:
    // Core processing loop
    void run_loop();

    // Process a single command
    EngineResult process_command(const PaymentCommand& cmd);

    // Specific command handlers
    EngineResult handle_freeze(const PaymentCommand& cmd);
    EngineResult handle_execute(const PaymentCommand& cmd);
    EngineResult handle_release(const PaymentCommand& cmd);
    EngineResult handle_refund(const PaymentCommand& cmd);
    EngineResult handle_settlement_batch(const PaymentCommand& cmd);

    // Generate and publish event
    void emit_event(const PaymentCommand& cmd, const std::string& event_type,
                    const std::string& result);

    // Dedup check per architecture doc §9.1
    bool is_duplicate(const std::string& request_id);

    // Periodic snapshot worker
    void snapshot_loop();

    Ledger ledger_;
    CommandQueue queue_;
    std::unique_ptr<WAL> wal_;
    Publisher publisher_;
    std::unique_ptr<std::thread> worker_thread_;
    std::unique_ptr<std::thread> snapshot_thread_;
    std::atomic<bool> running_{false};
    std::atomic<uint64_t> commands_processed_{0};
    std::atomic<uint64_t> commands_rejected_{0};
    std::atomic<uint64_t> commands_duplicated_{0};
    std::atomic<uint64_t> sequence_{0};
    std::atomic<uint64_t> last_snapshot_seq_{0};

    // Dedup set: store up to 1M recent request_ids (LRU-like)
    // Architecture doc §9.1: same request_id → return previous result
    static constexpr size_t MAX_DEDUP_SIZE = 1'000'000;
    std::unordered_set<std::string> processed_requests_;
    std::mutex dedup_mutex_;

    int snapshot_interval_sec_{300};
};

} // namespace engine
} // namespace aspira
