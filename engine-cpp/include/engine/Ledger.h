// Aspira Pay V2 — In-Memory Ledger
// Thread-safe account balance manager.
// Architecture doc §4.7.1: Single Writer Principle ensures consistency.
// All amounts are int64 (smallest currency unit).

#pragma once

#include "Types.h"
#include <string>
#include <unordered_map>
#include <shared_mutex>

namespace aspira {
namespace engine {

class Ledger {
public:
    Ledger() = default;

    // Initialize an account with starting balance
    void init_account(const std::string& account_id, int64_t available = 0);

    // Check if account exists
    bool account_exists(const std::string& account_id) const;

    // Get account balance (read-only, shared lock)
    AccountBalance get_balance(const std::string& account_id) const;

    // Freeze funds: move from available to frozen
    // Returns false if insufficient available balance
    bool freeze(const std::string& account_id, int64_t amount);

    // Unfreeze funds: move from frozen to available
    bool unfreeze(const std::string& account_id, int64_t amount);

    // Debit: move from frozen to settled (funds leave account)
    bool debit(const std::string& account_id, int64_t amount);

    // Credit: add to available balance (funds enter account)
    void credit(const std::string& account_id, int64_t amount);

    // Get snapshot of entire ledger (for WAL snapshot)
    std::unordered_map<std::string, AccountBalance> snapshot() const;

    // Restore from snapshot
    void restore(const std::unordered_map<std::string, AccountBalance>& state);

    // Total number of accounts
    size_t account_count() const;

private:
    mutable std::shared_mutex mutex_;
    std::unordered_map<std::string, AccountBalance> accounts_;
};

} // namespace engine
} // namespace aspira
