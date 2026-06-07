// Aspira Pay V2 — In-Memory Ledger Implementation
// Thread-safe account balance management.
// Architecture doc §4.7.3: Use single writer thread for consistency.

#include "engine/Ledger.h"
#include <iostream>
#include <mutex>

namespace aspira {
namespace engine {

void Ledger::init_account(const std::string& account_id, int64_t available) {
    std::unique_lock lock(mutex_);
    if (accounts_.find(account_id) == accounts_.end()) {
        AccountBalance bal;
        bal.available = available;
        accounts_[account_id] = bal;
    }
}

bool Ledger::account_exists(const std::string& account_id) const {
    std::shared_lock lock(mutex_);
    return accounts_.find(account_id) != accounts_.end();
}

AccountBalance Ledger::get_balance(const std::string& account_id) const {
    std::shared_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it != accounts_.end()) {
        return it->second;
    }
    return AccountBalance{};
}

bool Ledger::freeze(const std::string& account_id, int64_t amount) {
    // In single-writer mode, no lock needed for write
    // But we keep the lock for safety in potential multi-threaded access
    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) return false;

    auto& bal = it->second;
    if (bal.available < amount) return false;

    bal.available -= amount;
    bal.frozen += amount;
    return true;
}

bool Ledger::unfreeze(const std::string& account_id, int64_t amount) {
    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) return false;

    auto& bal = it->second;
    if (bal.frozen < amount) return false;

    bal.frozen -= amount;
    bal.available += amount;
    return true;
}

bool Ledger::debit(const std::string& account_id, int64_t amount) {
    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) return false;

    auto& bal = it->second;
    if (bal.frozen < amount) return false;

    bal.frozen -= amount;
    bal.settled += amount;
    return true;
}

void Ledger::credit(const std::string& account_id, int64_t amount) {
    std::unique_lock lock(mutex_);
    auto it = accounts_.find(account_id);
    if (it == accounts_.end()) {
        // Auto-create account on credit
        AccountBalance bal;
        bal.available = amount;
        accounts_[account_id] = bal;
        return;
    }
    it->second.available += amount;
}

std::unordered_map<std::string, AccountBalance> Ledger::snapshot() const {
    std::shared_lock lock(mutex_);
    return accounts_;
}

void Ledger::restore(const std::unordered_map<std::string, AccountBalance>& state) {
    std::unique_lock lock(mutex_);
    accounts_ = state;
}

size_t Ledger::account_count() const {
    std::shared_lock lock(mutex_);
    return accounts_.size();
}

} // namespace engine
} // namespace aspira
