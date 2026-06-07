// Aspira Pay V2 — Optimized In-Memory Ledger Implementation
// Architecture doc §9: uint64_t keys, version tracking, overflow detection.
// Single Writer Principle eliminates need for per-account locks in the hot path.

#include "engine/Ledger.h"
#include <iostream>
#include <chrono>
#include <mutex>

namespace aspira {
namespace engine {

// ──────────────────────────────────────────────
// Fast-path: uint64_t account ID (§9.1)
// ──────────────────────────────────────────────

void Ledger::init_account(uint64_t account_id, int64_t available) {
    std::unique_lock lock(mutex_);
    if (accounts_.find(account_id) == accounts_.end()) {
        AccountBalanceV2 bal;
        bal.available = available;
        bal.updated_at_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
            std::chrono::system_clock::now().time_since_epoch()).count();
        accounts_[account_id] = bal;
    }
}

bool Ledger::account_exists(uint64_t account_id) const {
    std::shared_lock lock(mutex_);
    return accounts_.find(account_id) != accounts_.end();
}

AccountBalanceV2 Ledger::get_balance(uint64_t account_id) const {
    std::shared_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it != accounts_.end()) return it->second;
    return AccountBalanceV2{};
}

EngineErrorCode Ledger::freeze(uint64_t account_id, int64_t amount) {
    uint64_t now_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();

    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) return EngineErrorCode::ACCOUNT_NOT_FOUND;

    auto& bal = it->second;

    // §9.3: Enforce no negative available balance
    if (bal.available < amount) return EngineErrorCode::INSUFFICIENT_FUNDS;

    // §9.3: Overflow detection
    int64_t new_frozen;
    if (add_overflow(bal.frozen, amount, new_frozen)) {
        return EngineErrorCode::OVERFLOW_DETECTED;
    }

    bal.available -= amount;
    bal.frozen = new_frozen;
    bal.version++;
    bal.updated_at_ns = now_ns;

    total_frozen_ += amount;
    return EngineErrorCode::OK;
}

EngineErrorCode Ledger::unfreeze(uint64_t account_id, int64_t amount) {
    uint64_t now_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();

    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) return EngineErrorCode::ACCOUNT_NOT_FOUND;

    auto& bal = it->second;
    if (bal.frozen < amount) return EngineErrorCode::INSUFFICIENT_FUNDS;

    int64_t new_avail;
    if (add_overflow(bal.available, amount, new_avail)) {
        return EngineErrorCode::OVERFLOW_DETECTED;
    }

    bal.frozen -= amount;
    bal.available = new_avail;
    bal.version++;
    bal.updated_at_ns = now_ns;

    total_frozen_ -= amount;
    return EngineErrorCode::OK;
}

EngineErrorCode Ledger::debit(uint64_t account_id, int64_t amount) {
    uint64_t now_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();

    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) return EngineErrorCode::ACCOUNT_NOT_FOUND;

    auto& bal = it->second;
    if (bal.frozen < amount) return EngineErrorCode::INSUFFICIENT_FUNDS;

    int64_t new_settled;
    if (add_overflow(bal.settled, amount, new_settled)) {
        return EngineErrorCode::OVERFLOW_DETECTED;
    }

    bal.frozen -= amount;
    bal.settled = new_settled;
    bal.version++;
    bal.updated_at_ns = now_ns;

    total_frozen_ -= amount;
    total_settled_ += amount;
    return EngineErrorCode::OK;
}

EngineErrorCode Ledger::credit(uint64_t account_id, int64_t amount) {
    uint64_t now_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();

    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) {
        // Auto-create account on first credit (with initial balance)
        AccountBalanceV2 bal;
        bal.available = amount;
        bal.updated_at_ns = now_ns;
        accounts_[account_id] = bal;
        return EngineErrorCode::OK;
    }

    int64_t new_avail;
    if (add_overflow(it->second.available, amount, new_avail)) {
        return EngineErrorCode::OVERFLOW_DETECTED;
    }

    it->second.available = new_avail;
    it->second.version++;
    it->second.updated_at_ns = now_ns;
    return EngineErrorCode::OK;
}

// ──────────────────────────────────────────────
// String-key wrappers (§6.4: adapter layer)
// ──────────────────────────────────────────────

void Ledger::register_account(const std::string& account_id, uint64_t numeric_id,
                               int64_t available) {
    {
        std::unique_lock lock(map_mutex_);
        account_id_map_[account_id] = numeric_id;
    }
    init_account(numeric_id, available);
}

std::optional<uint64_t> Ledger::resolve_account_id(const std::string& account_id) const {
    std::shared_lock lock(map_mutex_);
    auto it = account_id_map_.find(account_id);
    if (it != account_id_map_.end()) return it->second;
    return std::nullopt;
}

bool Ledger::freeze(const std::string& account_id, int64_t amount) {
    auto id = resolve_account_id(account_id);
    if (!id) return false;
    return ::aspira::engine::Ledger::freeze(*id, amount) == EngineErrorCode::OK;
}

bool Ledger::unfreeze(const std::string& account_id, int64_t amount) {
    auto id = resolve_account_id(account_id);
    if (!id) return false;
    return ::aspira::engine::Ledger::unfreeze(*id, amount) == EngineErrorCode::OK;
}

bool Ledger::debit(const std::string& account_id, int64_t amount) {
    auto id = resolve_account_id(account_id);
    if (!id) return false;
    return ::aspira::engine::Ledger::debit(*id, amount) == EngineErrorCode::OK;
}

void Ledger::credit(const std::string& account_id, int64_t amount) {
    auto id = resolve_account_id(account_id);
    if (!id) {
        // Generate a deterministic numeric ID from string hash for Sandbox
        uint64_t numeric_id = std::hash<std::string>{}(account_id);
        register_account(account_id, numeric_id, 0);
        id = numeric_id;
    }
    ::aspira::engine::Ledger::credit(*id, amount);
}

// ──────────────────────────────────────────────
// Snapshot / Restore (§12.2)
// ──────────────────────────────────────────────

Ledger::LedgerState Ledger::snapshot() const {
    std::shared_lock lock(mutex_);
    return accounts_;
}

void Ledger::restore(const LedgerState& state) {
    std::unique_lock lock(mutex_);
    accounts_ = state;

    // Recompute aggregates
    total_frozen_ = 0;
    total_settled_ = 0;
    for (const auto& [id, bal] : state) {
        total_frozen_ += bal.frozen;
        total_settled_ += bal.settled;
    }
}

size_t Ledger::account_count() const {
    std::shared_lock lock(mutex_);
    return accounts_.size();
}

} // namespace engine
} // namespace aspira
