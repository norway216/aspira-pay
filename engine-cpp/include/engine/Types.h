// Aspira Pay V2 — C++ Engine Types
// Defines all core data structures for the trading & clearing engine.
// Architecture doc §4.7.2

#pragma once

#include <cstdint>
#include <string>
#include <string_view>

namespace aspira {
namespace engine {

// Command types the engine can process
enum class CommandType : uint8_t {
    FREEZE_FUNDS      = 0,
    EXECUTE_PAYMENT   = 1,
    RELEASE_FUNDS     = 2,
    REFUND_PAYMENT    = 3,
    SETTLEMENT_BATCH  = 4
};

// Engine execution results
enum class EngineResult : uint8_t {
    ACCEPTED           = 0,
    REJECTED           = 1,
    EXECUTED           = 2,
    DUPLICATED         = 3,
    INSUFFICIENT_FUNDS = 4
};

// Payment command received from Go API
// Matches architecture doc §4.7.2
struct PaymentCommand {
    uint64_t    sequence_id;
    std::string request_id;
    std::string payment_id;
    CommandType command_type = CommandType::EXECUTE_PAYMENT;
    std::string from_account;
    std::string to_account;
    std::string source_currency;
    std::string target_currency;
    int64_t     source_amount;   // Smallest currency unit (cents)
    int64_t     target_amount;
    int64_t     fee_amount;
    int64_t     timestamp;       // Unix timestamp
};

// In-memory account balance
struct AccountBalance {
    int64_t available = 0;
    int64_t frozen    = 0;
    int64_t settled   = 0;

    int64_t total() const { return available + frozen + settled; }
    bool can_debit(int64_t amount) const { return available >= amount; }
};

// Engine event published to message queue
struct EngineEvent {
    uint64_t    sequence_id;
    std::string event_id;
    std::string payment_id;
    std::string event_type;
    std::string result;
    int64_t     timestamp;
};

// WAL entry types
enum class WALEntryType : uint8_t {
    COMMAND   = 0,
    EVENT     = 1,
    SNAPSHOT  = 2
};

} // namespace engine
} // namespace aspira
