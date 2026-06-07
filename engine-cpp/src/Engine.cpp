// Aspira Pay V2 — Core Engine Implementation
// Single Writer Principle for deterministic, consistent execution.
// Architecture doc §4.7.1: Core Engine Loop.
// Architecture doc §6.7.3: Engine Design Rules (10 rules).

#include "engine/Engine.h"
#include <cstdio>
#include <iostream>
#include <chrono>
#include <sstream>
#include <openssl/sha.h>

namespace aspira {
namespace engine {

Engine::Engine() : queue_(4096) {}

Engine::~Engine() {
    stop();
}

bool Engine::init(const std::string& wal_path, int snapshot_interval_sec) {
    wal_ = std::make_unique<WAL>(wal_path);
    snapshot_interval_sec_ = snapshot_interval_sec;
    std::cout << "[Engine] Initialized with WAL: " << wal_path
              << " (snapshot every " << snapshot_interval_sec_ << "s)" << std::endl;

    // Restore from WAL if possible
    restore_from_wal();

    return true;
}

void Engine::start() {
    running_ = true;
    worker_thread_ = std::make_unique<std::thread>(&Engine::run_loop, this);

    // Start snapshot thread
    snapshot_thread_ = std::make_unique<std::thread>(&Engine::snapshot_loop, this);

    std::cout << "[Engine] Started: worker + snapshot threads" << std::endl;
}

void Engine::stop() {
    running_ = false;

    // Take final snapshot before stopping
    take_snapshot();

    if (snapshot_thread_ && snapshot_thread_->joinable()) {
        snapshot_thread_->join();
    }
    if (worker_thread_ && worker_thread_->joinable()) {
        worker_thread_->join();
    }
    if (wal_) {
        wal_->sync();
    }
    std::cout << "[Engine] Stopped. Stats: processed=" << commands_processed_
              << " rejected=" << commands_rejected_
              << " duplicated=" << commands_duplicated_ << std::endl;
}

bool Engine::submit(const PaymentCommand& cmd) {
    if (!running_) return false;

    // Assign sequence ID if not set
    PaymentCommand cmd_with_seq = cmd;
    if (cmd_with_seq.sequence_id == 0) {
        cmd_with_seq.sequence_id = ++sequence_;
    }

    return queue_.enqueue(cmd_with_seq);
}

// ──────────────────────────────────────────────
// Snapshot
// Architecture doc §6.7.3 rule 8: Generate snapshots for fast recovery.
// ──────────────────────────────────────────────

void Engine::take_snapshot() {
    if (!wal_) return;

    uint64_t current_seq = sequence_.load();
    if (current_seq <= last_snapshot_seq_) return;

    // Serialize ledger state
    auto state = ledger_.snapshot();
    std::ostringstream oss;
    for (const auto& [account_id, balance] : state) {
        oss << account_id << "|"
            << balance.available << "|"
            << balance.frozen << "|"
            << balance.settled << "\n";
    }

    wal_->write_snapshot(oss.str());
    last_snapshot_seq_ = current_seq;

    if (state.size() > 0) {
        std::cout << "[Engine] Snapshot taken at seq=" << current_seq
                  << " accounts=" << state.size() << std::endl;
    }
}

void Engine::snapshot_loop() {
    while (running_) {
        std::this_thread::sleep_for(std::chrono::seconds(snapshot_interval_sec_));
        if (running_) {
            take_snapshot();
        }
    }
}

// ──────────────────────────────────────────────
// WAL Recovery
// Architecture doc §6.7.3 rule 8 & §13.2: Replay missing commands.
// ──────────────────────────────────────────────

bool Engine::restore_from_wal() {
    if (!wal_) return false;

    auto entries = wal_->read_all();
    if (entries.empty()) {
        std::cout << "[Engine] WAL is empty — fresh start" << std::endl;
        return true;
    }

    size_t restored = 0;
    for (const auto& entry : entries) {
        if (entry.type == WALEntryType::SNAPSHOT) {
            // Parse snapshot data and restore ledger
            std::unordered_map<std::string, AccountBalance> state;
            std::istringstream iss(entry.data);
            std::string line;
            while (std::getline(iss, line)) {
                if (line.empty()) continue;
                // Format: account_id|available|frozen|settled
                size_t p1 = line.find('|');
                size_t p2 = line.find('|', p1 + 1);
                size_t p3 = line.find('|', p2 + 1);
                if (p1 == std::string::npos || p2 == std::string::npos || p3 == std::string::npos) continue;

                AccountBalance bal;
                std::string id = line.substr(0, p1);
                bal.available = std::stoll(line.substr(p1 + 1, p2 - p1 - 1));
                bal.frozen = std::stoll(line.substr(p2 + 1, p3 - p2 - 1));
                bal.settled = std::stoll(line.substr(p3 + 1));
                state[id] = bal;
            }
            ledger_.restore(state);
            restored = state.size();
            std::cout << "[Engine] WAL snapshot restored: " << restored << " accounts" << std::endl;
        }
    }

    // Restore last sequence_id
    uint64_t recovered_seq = wal_->last_sequence_id();
    sequence_.store(recovered_seq);
    last_snapshot_seq_.store(recovered_seq);

    std::cout << "[Engine] WAL recovery complete: seq=" << sequence_
              << " accounts=" << ledger_.account_count() << std::endl;
    return true;
}

// ──────────────────────────────────────────────
// Core Loop
// ──────────────────────────────────────────────

void Engine::run_loop() {
    std::cout << "[Engine] Core loop started (Single Writer)" << std::endl;

    while (running_) {
        // Batch dequeue for efficiency (architecture doc §6.7.3: batch processing)
        auto commands = queue_.dequeue_batch(64);

        if (commands.empty()) {
            // No commands — brief sleep to avoid busy-waiting
            std::this_thread::sleep_for(std::chrono::microseconds(100));
            continue;
        }

        for (const auto& cmd : commands) {
            // Architecture doc §9.1: Dedup by request_id
            if (is_duplicate(cmd.request_id)) {
                commands_duplicated_++;
                emit_event(cmd, "COMMAND_DUPLICATED", "DUPLICATED");
                continue;
            }

            // Process command
            EngineResult result = process_command(cmd);

            if (result == EngineResult::EXECUTED || result == EngineResult::ACCEPTED) {
                commands_processed_++;
            } else if (result == EngineResult::REJECTED ||
                       result == EngineResult::INSUFFICIENT_FUNDS) {
                commands_rejected_++;
            }
        }

        // Periodic WAL sync (every ~100ms if there were commands)
        static auto last_sync = std::chrono::steady_clock::now();
        auto now = std::chrono::steady_clock::now();
        if (wal_ && std::chrono::duration_cast<std::chrono::milliseconds>(now - last_sync).count() > 100) {
            wal_->sync();
            last_sync = now;
        }
    }

    if (wal_) {
        wal_->sync();
    }
    std::cout << "[Engine] Core loop stopped" << std::endl;
}

// ──────────────────────────────────────────────
// Dedup (Architecture doc §9.1)
// ──────────────────────────────────────────────

bool Engine::is_duplicate(const std::string& request_id) {
    if (request_id.empty()) return false;

    std::lock_guard lock(dedup_mutex_);

    if (processed_requests_.count(request_id)) {
        return true;
    }

    // Evict old entries if set grows too large (simple LRU-like: clear and rebuild)
    if (processed_requests_.size() >= MAX_DEDUP_SIZE) {
        std::cout << "[Engine] Dedup set at capacity, clearing old entries" << std::endl;
        // Keep only entries that are still relevant (in production, use LRU cache)
        // For now, clear 25% of the set
        auto it = processed_requests_.begin();
        size_t to_remove = MAX_DEDUP_SIZE / 4;
        for (size_t i = 0; i < to_remove && it != processed_requests_.end(); ++i) {
            it = processed_requests_.erase(it);
        }
    }

    processed_requests_.insert(request_id);
    return false;
}

// ──────────────────────────────────────────────
// Command Processing
// ──────────────────────────────────────────────

EngineResult Engine::process_command(const PaymentCommand& cmd) {
    // Architecture doc §6.7.1: Command Decoder validates sequence_id, request_id

    switch (cmd.command_type) {
        case CommandType::FREEZE_FUNDS:
            return handle_freeze(cmd);
        case CommandType::EXECUTE_PAYMENT:
            return handle_execute(cmd);
        case CommandType::RELEASE_FUNDS:
            return handle_release(cmd);
        case CommandType::REFUND_PAYMENT:
            return handle_refund(cmd);
        case CommandType::SETTLEMENT_BATCH:
            return handle_settlement_batch(cmd);
        default:
            std::cerr << "[Engine] Unknown command type: " << static_cast<int>(cmd.command_type) << std::endl;
            return EngineResult::REJECTED;
    }
}

EngineResult Engine::handle_freeze(const PaymentCommand& cmd) {
    // Write to WAL first (WAL-before-action per architecture doc §6.7.3 rule 7)
    if (wal_) wal_->log_command(cmd);

    int64_t total_required = cmd.source_amount + cmd.fee_amount;

    if (!ledger_.freeze(cmd.from_account, total_required)) {
        emit_event(cmd, "FREEZE_FAILED", "INSUFFICIENT_FUNDS");
        return EngineResult::INSUFFICIENT_FUNDS;
    }

    emit_event(cmd, "FUNDS_FROZEN", "EXECUTED");
    return EngineResult::EXECUTED;
}

EngineResult Engine::handle_execute(const PaymentCommand& cmd) {
    if (wal_) wal_->log_command(cmd);

    int64_t total_required = cmd.source_amount + cmd.fee_amount;

    // Debit sender (frozen → settled, money leaves account)
    if (!ledger_.debit(cmd.from_account, total_required)) {
        emit_event(cmd, "EXECUTE_FAILED", "INSUFFICIENT_FUNDS");
        return EngineResult::INSUFFICIENT_FUNDS;
    }

    // Credit receiver (in target currency)
    ledger_.credit(cmd.to_account, cmd.target_amount);

    // Credit fee to platform
    std::string fee_account = "sys_fee_income_" + cmd.source_currency;
    ledger_.credit(fee_account, cmd.fee_amount);

    emit_event(cmd, "ENGINE_EXECUTED", "EXECUTED");
    return EngineResult::EXECUTED;
}

EngineResult Engine::handle_release(const PaymentCommand& cmd) {
    if (wal_) wal_->log_command(cmd);

    int64_t total_required = cmd.source_amount + cmd.fee_amount;

    if (!ledger_.unfreeze(cmd.from_account, total_required)) {
        emit_event(cmd, "RELEASE_FAILED", "REJECTED");
        return EngineResult::REJECTED;
    }

    emit_event(cmd, "FUNDS_RELEASED", "EXECUTED");
    return EngineResult::EXECUTED;
}

EngineResult Engine::handle_refund(const PaymentCommand& cmd) {
    if (wal_) wal_->log_command(cmd);

    // Credit back to sender (source amount + fee)
    ledger_.credit(cmd.from_account, cmd.source_amount + cmd.fee_amount);

    // Note: In production, this would require sufficient receiver balance
    // For Sandbox, we allow the receiver to go negative

    emit_event(cmd, "PAYMENT_REFUNDED", "EXECUTED");
    return EngineResult::EXECUTED;
}

EngineResult Engine::handle_settlement_batch(const PaymentCommand& cmd) {
    // Architecture doc §4.8: Settlement batch processing through engine
    if (wal_) wal_->log_command(cmd);

    // Settlement batch commands mark the completion of a batch
    // The actual ledger entries are written by the Settlement Service in Go
    emit_event(cmd, "SETTLEMENT_BATCH_PROCESSED", "EXECUTED");
    return EngineResult::EXECUTED;
}

// ──────────────────────────────────────────────
// Event Publishing
// ──────────────────────────────────────────────

void Engine::emit_event(const PaymentCommand& cmd, const std::string& event_type,
                        const std::string& result) {
    EngineEvent event;
    event.sequence_id = cmd.sequence_id;
    event.payment_id = cmd.payment_id;
    event.event_type = event_type;
    event.result = result;
    event.timestamp = std::chrono::duration_cast<std::chrono::seconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();

    // Generate event_id: SHA256(sequence_id + payment_id + event_type + timestamp)
    std::string event_data = std::to_string(event.sequence_id) + event.payment_id +
                             event_type + std::to_string(event.timestamp);
    unsigned char hash[SHA256_DIGEST_LENGTH];
    SHA256(reinterpret_cast<const unsigned char*>(event_data.c_str()),
           event_data.size(), hash);

    char hex_hash[SHA256_DIGEST_LENGTH * 2 + 1];
    for (int i = 0; i < SHA256_DIGEST_LENGTH; i++) {
        sprintf(hex_hash + i * 2, "%02x", hash[i]);
    }
    event.event_id = std::string("evt_") + std::string(hex_hash, 16);

    // Log to WAL
    if (wal_) wal_->log_event(event);

    // Publish to message queue
    publisher_.publish(event);
}

} // namespace engine
} // namespace aspira
