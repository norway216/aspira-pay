// Aspira Pay V2 — Optimized In-Memory Ledger
// Architecture doc §9: uint64_t account IDs, flat hash map, version tracking.
// Single-writer design: no per-account locks needed.
//
// Key optimizations over V1:
//   §9.1 - uint64_t keys instead of std::string → faster hash, less memory
//   §9.2 - version field for optimistic locking
//   §9.3 - overflow detection on credit/debit operations
//   §9.3 - balance invariant enforcement (no negative available)

#pragma once

#include "Types.h"
#include <cstdint>
#include <unordered_map>
#include <shared_mutex>
#include <string>
#include <optional>

namespace aspira {
namespace engine {

// Overflow-safe addition: returns true if a + b would overflow int64_t
inline bool add_overflow(int64_t a, int64_t b, int64_t& result) {
    if (b > 0 && a > INT64_MAX - b) return true;
    if (b < 0 && a < INT64_MIN - b) return true;
    result = a + b;
    return false;
}

class Ledger {
public:
    Ledger() = default;

    // ── Fast-path: uint64_t account ID (§9.1) ──────────────

    // Initialize an account with starting balance
    void init_account(uint64_t account_id, int64_t available = 0);

    bool account_exists(uint64_t account_id) const;

    AccountBalanceV2 get_balance(uint64_t account_id) const;

    // Freeze: available → frozen. Returns error code.
    EngineErrorCode freeze(uint64_t account_id, int64_t amount);

    // Unfreeze: frozen → available. Returns error code.
    EngineErrorCode unfreeze(uint64_t account_id, int64_t amount);

    // Debit: frozen → settled (money leaves account). Returns error code.
    EngineErrorCode debit(uint64_t account_id, int64_t amount);

    // Credit: add to available balance. Returns error code.
    EngineErrorCode credit(uint64_t account_id, int64_t amount);

    // ── String-key wrappers (for backward compat with Go adapter) ──

    // Register a string → uint64 mapping (called at account creation)
    void register_account(const std::string& account_id, uint64_t numeric_id,
                          int64_t available = 0);

    // Look up numeric ID from string
    std::optional<uint64_t> resolve_account_id(const std::string& account_id) const;

    // String-key methods delegate to uint64 after resolution
    bool freeze(const std::string& account_id, int64_t amount);
    bool unfreeze(const std::string& account_id, int64_t amount);
    bool debit(const std::string& account_id, int64_t amount);
    void credit(const std::string& account_id, int64_t amount);

    // ── Snapshot / Restore (§12.2) ────────────────────────

    using LedgerState = std::unordered_map<uint64_t, AccountBalanceV2>;
    LedgerState snapshot() const;
    void restore(const LedgerState& state);

    // ── Statistics ────────────────────────────────────────

    size_t account_count() const;
    uint64_t total_frozen() const { return total_frozen_; }
    uint64_t total_settled() const { return total_settled_; }

private:
    mutable std::shared_mutex mutex_;
    std::unordered_map<uint64_t, AccountBalanceV2> accounts_;

    // String → uint64_t ID mapping (for adapter layer)
    std::unordered_map<std::string, uint64_t> account_id_map_;
    mutable std::shared_mutex map_mutex_;

    // Aggregate statistics
    uint64_t total_frozen_{0};
    uint64_t total_settled_{0};
};

} // namespace engine
} // namespace aspira
