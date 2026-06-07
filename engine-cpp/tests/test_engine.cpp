// Aspira Pay V2 — Engine Unit Tests
// Basic smoke tests for the C++ engine components.

#include "engine/Engine.h"
#include "engine/Ledger.h"
#include "engine/CommandQueue.h"
#include <cassert>
#include <iostream>

using namespace aspira::engine;

void test_ledger_basic() {
    std::cout << "Test: Ledger basic operations..." << std::endl;

    Ledger ledger;
    ledger.init_account("acc_A", 100000); // $1,000

    assert(ledger.account_exists("acc_A"));
    assert(!ledger.account_exists("acc_B"));

    auto bal = ledger.get_balance("acc_A");
    assert(bal.available == 100000);
    assert(bal.frozen == 0);

    // Freeze funds
    assert(ledger.freeze("acc_A", 50000));
    bal = ledger.get_balance("acc_A");
    assert(bal.available == 50000);
    assert(bal.frozen == 50000);

    // Insufficient funds
    assert(!ledger.freeze("acc_A", 100000));

    // Debit
    assert(ledger.debit("acc_A", 30000));
    bal = ledger.get_balance("acc_A");
    assert(bal.frozen == 20000);
    assert(bal.settled == 30000);

    // Unfreeze
    assert(ledger.unfreeze("acc_A", 20000));
    bal = ledger.get_balance("acc_A");
    assert(bal.available == 70000);
    assert(bal.frozen == 0);

    // Credit (auto-create account)
    ledger.credit("acc_B", 50000);
    assert(ledger.account_exists("acc_B"));
    bal = ledger.get_balance("acc_B");
    assert(bal.available == 50000);

    std::cout << "  PASSED" << std::endl;
}

void test_command_queue() {
    std::cout << "Test: Command queue..." << std::endl;

    CommandQueue q(16); // Power of 2

    assert(q.empty());
    assert(q.size() == 0);

    // Enqueue
    PaymentCommand cmd;
    cmd.sequence_id = 1;
    cmd.payment_id = "pay_test_001";
    cmd.source_amount = 10000;

    assert(q.enqueue(cmd));
    assert(!q.empty());
    assert(q.size() == 1);

    // Dequeue
    auto result = q.dequeue();
    assert(result != nullptr);
    assert(result->payment_id == "pay_test_001");
    assert(result->source_amount == 10000);

    assert(q.empty());

    // Batch dequeue
    for (int i = 0; i < 5; i++) {
        PaymentCommand c;
        c.sequence_id = i;
        c.payment_id = "pay_batch_" + std::to_string(i);
        assert(q.enqueue(c));
    }

    auto batch = q.dequeue_batch(3);
    assert(batch.size() == 3);
    assert(q.size() == 2);

    auto remaining = q.dequeue_batch(10);
    assert(remaining.size() == 2);
    assert(q.empty());

    std::cout << "  PASSED" << std::endl;
}

void test_double_entry_balance() {
    std::cout << "Test: Double-entry balance..." << std::endl;

    Ledger ledger;

    // Setup accounts
    ledger.init_account("sender_usd", 100000);    // Sender has $1,000
    ledger.init_account("receiver_jpy", 0);       // Receiver has 0 JPY
    ledger.init_account("fee_income_usd", 0);     // Platform fee account

    // Simulate a cross-border payment:
    // Sender pays $100 + $1 fee, receiver gets ~15,600 JPY (at rate 156)

    int64_t source_amount = 10000;   // $100.00 in cents
    int64_t fee_amount = 100;        // $1.00 in cents
    int64_t target_amount = 1560000; // ¥15,600 in sen/smallest unit
    int64_t total_debit = source_amount + fee_amount; // $101.00

    // 1. Freeze sender funds
    assert(ledger.freeze("sender_usd", total_debit));
    auto bal = ledger.get_balance("sender_usd");
    assert(bal.available == 100000 - total_debit);
    assert(bal.frozen == total_debit);

    // 2. Execute payment: debit sender, credit receiver, credit fee
    assert(ledger.debit("sender_usd", total_debit));
    ledger.credit("receiver_jpy", target_amount);
    ledger.credit("fee_income_usd", fee_amount);

    // Verify final balances
    auto sender = ledger.get_balance("sender_usd");
    assert(sender.available == 100000 - total_debit);
    assert(sender.frozen == 0);
    assert(sender.settled == total_debit);
    // Total: available(89900) + frozen(0) + settled(10100) = 100000 ✓

    auto receiver = ledger.get_balance("receiver_jpy");
    assert(receiver.available == target_amount);

    auto fee = ledger.get_balance("fee_income_usd");
    assert(fee.available == fee_amount);

    // 3. Double-entry check:
    // Debits: sender settled increased by 10100 (money LEFT sender) ✓
    // Credits: receiver got 1560000 JPY, platform got 100 USD fee ✓
    // Money is conserved across accounts

    std::cout << "  PASSED (sender settled=" << sender.settled
              << ", receiver=" << receiver.available
              << ", fee=" << fee.available << ")" << std::endl;
}

int main() {
    std::cout << "═══════════════════════════════════════════════" << std::endl;
    std::cout << "  Aspira Pay V2 — Engine Unit Tests" << std::endl;
    std::cout << "═══════════════════════════════════════════════" << std::endl;

    test_ledger_basic();
    test_command_queue();
    test_double_entry_balance();

    std::cout << "═══════════════════════════════════════════════" << std::endl;
    std::cout << "  All tests PASSED" << std::endl;
    std::cout << "═══════════════════════════════════════════════" << std::endl;

    return 0;
}
