// Aspira Pay V2 — C++ Engine Main Entry (Optimized)
// Architecture doc §24: V2 Engine MVP with 6 dedicated threads.
// Uses EngineMetrics for all statistics reporting.

#include "engine/Engine.h"
#include <iostream>
#include <csignal>
#include <thread>
#include <chrono>

using namespace aspira::engine;

static Engine* g_engine = nullptr;

void signal_handler(int signal) {
    std::cout << "\n[Engine] Received signal " << signal << ", shutting down..." << std::endl;
    if (g_engine) {
        g_engine->stop();
    }
}

int main(int argc, char* argv[]) {
    std::cout << "═══════════════════════════════════════════════════════" << std::endl;
    std::cout << "  Aspira Pay V2 — C++ Trading & Clearing Engine" << std::endl;
    std::cout << "  Version: 2.0.0-sandbox (Optimized)" << std::endl;
    std::cout << "  C++ Standard: C++20" << std::endl;
    std::cout << "═══════════════════════════════════════════════════════" << std::endl;

    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);

    // WAL path from command line or default
    std::string wal_path = "engine.wal";
    if (argc > 1) {
        wal_path = argv[1];
    }

    // Configure WAL for batch flush (§11.3)
    WalConfig wal_config;
    wal_config.flush_policy = WalConfig::FlushPolicy::BATCH;
    wal_config.batch_size = 1000;
    wal_config.flush_interval_us = 1000;
    wal_config.fsync_on_flush = true;
    wal_config.enable_checksum = true;

    // Create and initialize engine
    Engine engine;
    g_engine = &engine;

    if (!engine.init(wal_path, 300, wal_config)) {
        std::cerr << "Failed to initialize engine" << std::endl;
        return 1;
    }

    // Initialize test accounts using register_account (string → uint64 mapping)
    // Architecture doc §6.4: Adapter layer maps external IDs to internal uint64
    engine.ledger().register_account("acc_test_sender_usd", 1001, 1000000);     // $10,000
    engine.ledger().register_account("acc_test_receiver_jpy", 2001, 0);         // 0 JPY
    engine.ledger().register_account("sys_fee_income_usd", 9001, 0);            // Fee account

    std::cout << "[Engine] Test accounts registered (Sandbox mode)" << std::endl;

    // Start engine with 5 background threads (§16)
    engine.start();

    // Main thread: print metrics periodically (§19)
    while (engine.is_running()) {
        std::this_thread::sleep_for(std::chrono::seconds(5));

        auto snap = engine.metrics().get_snapshot();
        std::cout << "[Stats] "
                  << "proc=" << snap.commands_processed
                  << " rej=" << snap.commands_rejected
                  << " dup=" << snap.commands_duplicated
                  << " drop=" << snap.commands_dropped
                  << " events=" << snap.events_published
                  << " ev_drop=" << snap.events_dropped
                  << " dedup=" << snap.dedup_cache_size
                  << " accts=" << snap.account_count
                  << std::endl;
    }

    auto final = engine.metrics().get_snapshot();
    std::cout << "[Engine] Final stats: "
              << final.commands_processed << " processed, "
              << final.commands_rejected << " rejected, "
              << final.commands_duplicated << " duplicated" << std::endl;

    return 0;
}
