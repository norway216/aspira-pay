// Aspira Pay V3 — C++ Fee Engine (§5.6)
// Wise-like transparent fee calculation in the hot path.
// All amounts in smallest currency unit (cents).
//
// Fee formula (§11.2):
//   total_fee = fixed_fee + amount * variable_rate + channel_cost + risk_markup
//
// Fee types (§9.3):
//   CARD_SAME_CURRENCY_SPEND     → 0%
//   CARD_CROSS_CURRENCY_SPEND    → 0.45% + $0.20
//   CARD_ATM_WITHDRAWAL          → 1.0% + $1.50
//   CARD_REFUND                  → 0%
//   CARD_CHARGEBACK              → $15.00 flat

#pragma once

#include <cstdint>
#include <string>
#include <unordered_map>

namespace aspira {
namespace engine {

enum class FeeScenario : uint8_t {
    SAME_CURRENCY_SPEND  = 0,
    CROSS_CURRENCY_SPEND = 1,
    ATM_WITHDRAWAL       = 2,
    REFUND               = 3,
    CHARGEBACK           = 4,
    CROSS_BORDER_TRANSFER = 5
};

struct FeeRule {
    uint32_t basis_points;   // Variable fee in bps (1 bp = 0.01%)
    int64_t  fixed_fee;      // Fixed fee in cents
    int64_t  min_fee;        // Minimum fee in cents
    int64_t  max_fee;        // Maximum fee cap in cents
    int32_t  risk_bps;       // Risk markup in bps
};

struct FeeResult {
    int64_t total_fee;       // Total fee in cents
    int64_t fixed_part;      // Fixed component
    int64_t variable_part;   // Variable component (amount * bps)
    int64_t risk_markup;     // Risk surcharge
    uint32_t effective_bps;  // Total effective bps applied
    bool     capped;         // Whether fee hit the cap
};

class FeeEngine {
public:
    FeeEngine();

    // Calculate fee for a given scenario and amount.
    // Architecture doc §11.2: total = fixed + amount*bps/10000 + risk
    FeeResult calculate(FeeScenario scenario, int64_t amount,
                        const std::string& source_currency = "",
                        const std::string& target_currency = "",
                        const std::string& country = "",
                        int32_t risk_score = 0);

    // Quick same-currency check (zero fees, inline)
    static bool is_free(FeeScenario s) { return s == FeeScenario::SAME_CURRENCY_SPEND; }

    // Get the effective rate for display purposes (e.g., "0.45%")
    uint32_t effective_rate(FeeScenario scenario, bool is_cross_currency) const;

    // Update a fee rule at runtime
    void set_rule(FeeScenario scenario, const FeeRule& rule);

    // Load default Wise-like fee schedule
    void load_defaults();

private:
    std::unordered_map<uint8_t, FeeRule> rules_;

    // Risk markup per risk level (0-100 score)
    static constexpr int32_t RISK_LOW_BPS    = 0;
    static constexpr int32_t RISK_MEDIUM_BPS = 10;   // +0.10%
    static constexpr int32_t RISK_HIGH_BPS   = 50;   // +0.50%
};

} // namespace engine
} // namespace aspira
