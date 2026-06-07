// Aspira Pay V2 — C++ Engine Main Entry
// High-performance trading & clearing engine.
// Architecture doc §4.7

#include "engine/Engine.h"
#include <iostream>
#include <csignal>
#include <thread>
#include <chrono>

using namespace aspira::engine;

// Global engine pointer for signal handler
static Engine* g_engine = nullptr;

void signal_handler(int signal) {
    std::cout << "\nReceived signal " << signal << ", shutting down..." << std::endl;
    if (g_engine) {
        g_engine->stop();
    }
}

int main(int argc, char* argv[]) {
    std::cout << "═══════════════════════════════════════════════════════" << std::endl;
    std::cout << "  Aspira Pay V2 — C++ Trading & Clearing Engine" << std::endl;
    std::cout << "  Version: 2.0.0-sandbox" << std::endl;
    std::cout << "  C++ Standard: C++20" << std::endl;
    std::cout << "═══════════════════════════════════════════════════════" << std::endl;

    // Setup signal handling
    std::signal(SIGINT, signal_handler);
    std::signal(SIGTERM, signal_handler);

    // Determine WAL path
    std::string wal_path = "engine.wal";
    if (argc > 1) {
        wal_path = argv[1];
    }

    // Create and initialize engine
    Engine engine;
    g_engine = &engine;

    if (!engine.init(wal_path)) {
        std::cerr << "Failed to initialize engine" << std::endl;
        return 1;
    }

    // Initialize some test accounts (Sandbox)
    engine.ledger().init_account("acc_test_sender_usd", 1000000);   // $10,000
    engine.ledger().init_account("acc_test_receiver_jpy", 0);       // 0 JPY
    engine.ledger().init_account("sys_fee_income_usd", 0);

    std::cout << "Engine initialized with test accounts" << std::endl;
    std::cout << "WAL path: " << wal_path << std::endl;

    // Start engine loop
    engine.start();
    std::cout << "Engine running. Press Ctrl+C to stop." << std::endl;

    // Main thread: print stats periodically
    while (engine.is_running()) {
        std::this_thread::sleep_for(std::chrono::seconds(5));
        std::cout << "[Stats] Processed: " << engine.commands_processed()
                  << " | Rejected: " << engine.commands_rejected()
                  << " | Queue depth: " << 0  // Would need queue accessor
                  << " | Events: " << engine.events_published()
                  << std::endl;
    }

    std::cout << "Engine stopped. Final stats: "
              << engine.commands_processed() << " commands processed, "
              << engine.commands_rejected() << " rejected" << std::endl;

    return 0;
}
