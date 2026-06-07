// Aspira Pay V2 — Core Engine Loop
// Single Writer Principle: processes commands sequentially.
// Architecture doc §4.7.1: Core Engine Loop.

#pragma once

#include "Types.h"
#include "Ledger.h"
#include "CommandQueue.h"
#include "WAL.h"
#include "Publisher.h"
#include <memory>
#include <atomic>
#include <thread>

namespace aspira {
namespace engine {

class Engine {
public:
    Engine();
    ~Engine();

    // Initialize engine with WAL path
    bool init(const std::string& wal_path);

    // Start the engine processing loop (runs in background thread)
    void start();

    // Stop the engine gracefully
    void stop();

    // Submit a command to the engine (producer-side, thread-safe)
    bool submit(const PaymentCommand& cmd);

    // Get ledger reference (for balance queries)
    Ledger& ledger() { return ledger_; }

    // Get publisher reference
    Publisher& publisher() { return publisher_; }

    // Engine statistics
    uint64_t commands_processed() const { return commands_processed_; }
    uint64_t commands_rejected() const { return commands_rejected_; }
    uint64_t events_published() const { return publisher_.total_published(); }
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

    // Generate and publish event
    void emit_event(const PaymentCommand& cmd, const std::string& event_type,
                    const std::string& result);

    Ledger ledger_;
    CommandQueue queue_;
    std::unique_ptr<WAL> wal_;
    Publisher publisher_;
    std::unique_ptr<std::thread> worker_thread_;
    std::atomic<bool> running_{false};
    std::atomic<uint64_t> commands_processed_{0};
    std::atomic<uint64_t> commands_rejected_{0};
    std::atomic<uint64_t> sequence_{0};
};

} // namespace engine
} // namespace aspira
