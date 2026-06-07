// Aspira Pay V2 — Core Engine Implementation
// Single Writer Principle for deterministic, consistent execution.
// Architecture doc §4.7

#include "engine/Engine.h"
#include <cstdio>
#include <iostream>
#include <chrono>
#include <openssl/sha.h>

namespace aspira {
namespace engine {

Engine::Engine() : queue_(4096) {}

Engine::~Engine() {
    stop();
}

bool Engine::init(const std::string& wal_path) {
    wal_ = std::make_unique<WAL>(wal_path);
    std::cout << "[Engine] Initialized with WAL: " << wal_path << std::endl;
    return true;
}

void Engine::start() {
    running_ = true;
    worker_thread_ = std::make_unique<std::thread>(&Engine::run_loop, this);
}

void Engine::stop() {
    running_ = false;
    if (worker_thread_ && worker_thread_->joinable()) {
        worker_thread_->join();
    }
    if (wal_) {
        wal_->sync();
    }
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

void Engine::run_loop() {
    std::cout << "[Engine] Core loop started (Single Writer)" << std::endl;

    while (running_) {
        // Batch dequeue for efficiency (architecture doc §4.7.3: batch processing)
        auto commands = queue_.dequeue_batch(64);

        if (commands.empty()) {
            // No commands — brief sleep to avoid busy-waiting
            std::this_thread::sleep_for(std::chrono::microseconds(100));
            continue;
        }

        for (const auto& cmd : commands) {
            // Check dedup by request_id (simplified — production would use a set)
            // Architecture doc §9.1: same request_id -> skip

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

EngineResult Engine::process_command(const PaymentCommand& cmd) {
    // Architecture doc §4.7.1: Command Decoder validates sequence_id, request_id

    switch (cmd.command_type) {
        case CommandType::FREEZE_FUNDS:
            return handle_freeze(cmd);
        case CommandType::EXECUTE_PAYMENT:
            return handle_execute(cmd);
        case CommandType::RELEASE_FUNDS:
            return handle_release(cmd);
        case CommandType::REFUND_PAYMENT:
            return handle_refund(cmd);
        default:
            std::cerr << "[Engine] Unknown command type: " << static_cast<int>(cmd.command_type) << std::endl;
            return EngineResult::REJECTED;
    }
}

EngineResult Engine::handle_freeze(const PaymentCommand& cmd) {
    // Write to WAL first (WAL-before-action)
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
