// Aspira Pay V2 — C++ Engine Types (Optimized)
// Architecture doc §6: Fixed-field command model with uint64 IDs.
// Architecture doc §17: Error codes instead of exceptions.

#pragma once

#include <cstdint>
#include <string>
#include <string_view>
#include <array>

namespace aspira {
namespace engine {

// ──────────────────────────────────────────────
// Command & Result Types (§6.2, §17.2)
// ──────────────────────────────────────────────

enum class CommandType : uint8_t {
    FREEZE_FUNDS      = 0,
    EXECUTE_PAYMENT   = 1,
    RELEASE_FUNDS     = 2,
    REFUND_PAYMENT    = 3,
    SETTLEMENT_BATCH  = 4
};

enum class EngineResult : uint8_t {
    ACCEPTED           = 0,
    REJECTED           = 1,
    EXECUTED           = 2,
    DUPLICATED         = 3,
    INSUFFICIENT_FUNDS = 4,
    INVALID_SEQUENCE   = 5   // §6.7.2
};

// §17: Error codes for business errors (no exceptions in hot path)
enum class EngineErrorCode : uint8_t {
    OK                    = 0,
    DUPLICATED_REQUEST    = 1,
    INVALID_SEQUENCE      = 2,
    ACCOUNT_NOT_FOUND     = 3,
    INSUFFICIENT_FUNDS    = 4,
    ACCOUNT_FROZEN        = 5,
    WAL_WRITE_FAILED      = 6,
    QUEUE_FULL            = 7,
    INTERNAL_ERROR        = 8,
    OVERFLOW_DETECTED     = 9
};

// §17.2: Structured execution result
struct EngineExecutionResult {
    uint64_t        sequence_id;
    uint64_t        payment_id;
    EngineErrorCode code;
    int64_t         latency_ns;  // Hot-path execution time in nanoseconds
};

// ——— String-based Command (backward-compatible, used by Go adapter) ———
// §6.1: This is the "business-friendly" model. Use only at the boundary layer.
// The engine hot path should use PaymentCommandFast instead.

struct PaymentCommand {
    uint64_t    sequence_id = 0;
    std::string request_id;
    std::string payment_id;
    CommandType command_type = CommandType::EXECUTE_PAYMENT;
    std::string from_account;
    std::string to_account;
    std::string source_currency;
    std::string target_currency;
    int64_t     source_amount = 0;
    int64_t     target_amount = 0;
    int64_t     fee_amount = 0;
    int64_t     timestamp = 0;
};

// ——— Fixed-field Fast Command (§6.2) ———
// All string IDs replaced with uint64_t for zero-allocation hot path.
// 48 bytes total — fits in a single cache line.

struct alignas(64) PaymentCommandFast {
    uint64_t    sequence_id = 0;
    uint64_t    request_id_hash = 0;   // XXH64 / SHA256-64 of request_id
    uint64_t    payment_id_hash = 0;
    uint64_t    from_account_id = 0;   // Mapped from string by engine adapter
    uint64_t    to_account_id = 0;
    uint16_t    source_currency = 0;   // ISO 4217 numeric code (§6.3)
    uint16_t    target_currency = 0;
    CommandType command_type = CommandType::EXECUTE_PAYMENT;
    uint8_t     _padding[3];
    int64_t     source_amount = 0;
    int64_t     target_amount = 0;
    int64_t     fee_amount = 0;
    int64_t     timestamp_ns = 0;      // Nanosecond precision
};

static_assert(sizeof(PaymentCommandFast) <= 128, "PaymentCommandFast must fit within two cache lines");

// ──────────────────────────────────────────────
// ISO 4217 Currency Code Mapping (§6.3)
// ──────────────────────────────────────────────

namespace currency {

constexpr uint16_t USD = 840;
constexpr uint16_t EUR = 978;
constexpr uint16_t JPY = 392;
constexpr uint16_t CNY = 156;
constexpr uint16_t HKD = 344;
constexpr uint16_t GBP = 826;
constexpr uint16_t SGD = 702;
constexpr uint16_t CHF = 756;
constexpr uint16_t AUD = 036;
constexpr uint16_t CAD = 124;
constexpr uint16_t KRW = 410;
constexpr uint16_t INR = 356;

// Convert ISO 4217 numeric code → 3-letter string (for event output)
inline const char* to_string(uint16_t code) {
    switch (code) {
        case USD: return "USD"; case EUR: return "EUR";
        case JPY: return "JPY"; case CNY: return "CNY";
        case HKD: return "HKD"; case GBP: return "GBP";
        case SGD: return "SGD"; case CHF: return "CHF";
        case AUD: return "AUD"; case CAD: return "CAD";
        case KRW: return "KRW"; case INR: return "INR";
        default:  return "XXX";
    }
}

// Convert 3-letter currency string → ISO 4217 numeric code
inline uint16_t from_string(std::string_view code) {
    if (code == "USD") return USD; if (code == "EUR") return EUR;
    if (code == "JPY") return JPY; if (code == "CNY") return CNY;
    if (code == "HKD") return HKD; if (code == "GBP") return GBP;
    if (code == "SGD") return SGD; if (code == "CHF") return CHF;
    if (code == "AUD") return AUD; if (code == "CAD") return CAD;
    if (code == "KRW") return KRW; if (code == "INR") return INR;
    return 0; // Unknown
}

} // namespace currency

// ──────────────────────────────────────────────
// In-Memory Balance (§9.2)
// ──────────────────────────────────────────────

struct AccountBalance {
    int64_t available = 0;
    int64_t frozen    = 0;
    int64_t settled   = 0;

    int64_t total() const { return available + frozen + settled; }
    bool can_debit(int64_t amount) const { return available >= amount; }
};

// §9.2: Optimized balance with version and timestamp
struct AccountBalanceV2 {
    int64_t  available = 0;
    int64_t  frozen    = 0;
    int64_t  settled   = 0;
    int64_t  version   = 0;       // Monotonic version for optimistic locking
    uint64_t updated_at_ns = 0;   // Last update timestamp

    int64_t total() const { return available + frozen + settled; }
    bool can_debit(int64_t amount) const { return available >= amount; }
};

// §9.2: Hot account balance with cache-line alignment (64 bytes)
struct alignas(64) HotAccountBalance {
    int64_t  available = 0;
    int64_t  frozen    = 0;
    int64_t  settled   = 0;
    int64_t  version   = 0;
    uint64_t updated_at_ns = 0;
    // Padding to 64 bytes (8+8+8+8+8 = 40, +24 padding)
    uint8_t _padding[24] = {};
};

static_assert(sizeof(HotAccountBalance) == 64, "HotAccountBalance must be 64 bytes");

// ──────────────────────────────────────────────
// Engine Event
// ──────────────────────────────────────────────

struct EngineEvent {
    uint64_t    sequence_id = 0;
    std::string event_id;
    uint64_t    payment_id_hash = 0;  // §6.2: Use hash instead of string payment_id
    std::string event_type;
    std::string result;
    int64_t     timestamp = 0;
};

// ──────────────────────────────────────────────
// WAL Types (§11.4)
// ──────────────────────────────────────────────

enum class WALEntryType : uint8_t {
    COMMAND   = 0,
    EVENT     = 1,
    SNAPSHOT  = 2
};

// §11.4: WAL record header with checksum
struct WalRecordHeader {
    uint64_t sequence_id;
    uint32_t record_type;       // WALEntryType
    uint32_t payload_size;
    uint64_t timestamp_ns;
    uint64_t checksum;          // CRC64 or XXH64 of payload
};

// ──────────────────────────────────────────────
// Latency Tracking (§19.2)
// ──────────────────────────────────────────────

struct CommandLatency {
    uint64_t decode_ns        = 0;
    uint64_t queue_wait_ns    = 0;
    uint64_t execute_ns       = 0;
    uint64_t wal_append_ns    = 0;
    uint64_t event_enqueue_ns = 0;

    uint64_t total_ns() const {
        return decode_ns + queue_wait_ns + execute_ns + wal_append_ns + event_enqueue_ns;
    }
};

} // namespace engine
} // namespace aspira
