// Aspira Pay V3 — Fee Engine Implementation (§5.6)
#include "engine/FeeEngine.h"
#include <algorithm>
#include <iostream>

namespace aspira {
namespace engine {

FeeEngine::FeeEngine() {
    load_defaults();
}

void FeeEngine::load_defaults() {
    // Wise-like transparent fee schedule (§9.3)
    // Variable fee in basis points (1 bp = 0.01%), fixed fee in cents

    // Same-currency card spend: FREE
    rules_[(uint8_t)FeeScenario::SAME_CURRENCY_SPEND] = {
        0,     // 0%
        0,     // $0.00 fixed
        0,     // no min
        0,     // no max
        0      // no risk markup
    };

    // Cross-currency card spend: 0.45% + $0.20
    rules_[(uint8_t)FeeScenario::CROSS_CURRENCY_SPEND] = {
        45,    // 0.45%
        20,    // $0.20 fixed
        0,     // no min
        5000,  // $50.00 max cap
        0
    };

    // ATM withdrawal: 1.0% + $1.50 (capped at $5.00)
    rules_[(uint8_t)FeeScenario::ATM_WITHDRAWAL] = {
        100,   // 1.00%
        150,   // $1.50 fixed
        200,   // $2.00 min
        500,   // $5.00 max
        0
    };

    // Refund: FREE
    rules_[(uint8_t)FeeScenario::REFUND] = {0, 0, 0, 0, 0};

    // Chargeback: $15.00 flat penalty
    rules_[(uint8_t)FeeScenario::CHARGEBACK] = {
        0,     // 0%
        1500,  // $15.00 flat
        0, 0, 0
    };

    // Cross-border transfer: 0.35% + $0.50
    rules_[(uint8_t)FeeScenario::CROSS_BORDER_TRANSFER] = {
        35,    // 0.35%
        50,    // $0.50 fixed
        100,   // $1.00 min
        10000, // $100.00 max
        0
    };

    std::cout << "[FeeEngine] Default Wise-like fee schedule loaded (6 scenarios)" << std::endl;
}

FeeResult FeeEngine::calculate(FeeScenario scenario, int64_t amount,
                                const std::string&, const std::string&,
                                const std::string&, int32_t risk_score) {
    auto it = rules_.find((uint8_t)scenario);
    if (it == rules_.end()) {
        // Unknown scenario: apply conservative 1% + $1.00
        FeeRule conservative{100, 100, 0, 10000, 0};
        it = rules_.emplace((uint8_t)scenario, conservative).first;
    }

    const FeeRule& rule = it->second;

    // Determine risk markup
    int32_t risk_bps = rule.risk_bps;
    if (risk_score >= 80)       risk_bps += RISK_HIGH_BPS;
    else if (risk_score >= 40)  risk_bps += RISK_MEDIUM_BPS;

    // Calculate variable fee: amount * (bps + risk_bps) / 10000
    // Use 128-bit intermediate to avoid overflow on large amounts
    int64_t total_bps = static_cast<int64_t>(rule.basis_points) + risk_bps;
    __int128 variable = (__int128)amount * total_bps / 10000;

    // Total = fixed + variable
    int64_t fixed = rule.fixed_fee;
    int64_t var_part = static_cast<int64_t>(variable);
    int64_t total = fixed + var_part;

    bool capped = false;

    // Apply minimum fee
    if (rule.min_fee > 0 && total < rule.min_fee) {
        total = rule.min_fee;
    }

    // Apply maximum fee cap
    if (rule.max_fee > 0 && total > rule.max_fee) {
        total = rule.max_fee;
        capped = true;
    }

    return FeeResult{
        total,
        fixed,
        var_part,
        0, // risk markup already included in var_part
        static_cast<uint32_t>(total_bps),
        capped
    };
}

uint32_t FeeEngine::effective_rate(FeeScenario scenario, bool is_cross_currency) const {
    if (!is_cross_currency) return 0;
    auto it = rules_.find((uint8_t)scenario);
    return (it != rules_.end()) ? it->second.basis_points : 100;
}

void FeeEngine::set_rule(FeeScenario scenario, const FeeRule& rule) {
    rules_[(uint8_t)scenario] = rule;
}

} // namespace engine
} // namespace aspira
