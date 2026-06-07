// Aspira Pay V2 — Optimized Write-Ahead Log
// Architecture doc §11: Checksum, batch flush, configurable fsync policy.
//
// §11.3 Batch Flush Policy:
//   - Append to WAL buffer, flush every N records or X microseconds
//   - fsync according to configured durability level
//
// §11.5 Checksum:
//   - Every WAL record includes a 64-bit checksum
//   - Detects partial writes and corrupted records
//   - Stops replay safely at first corrupted record

#pragma once

#include "Types.h"
#include <string>
#include <vector>
#include <fstream>
#include <mutex>
#include <cstdint>

namespace aspira {
namespace engine {

// §11.3: WAL flush configuration
struct WalConfig {
    enum class FlushPolicy {
        EVERY_RECORD,   // fsync every record (safest, slowest)
        BATCH,          // fsync every N records
        INTERVAL        // fsync every X microseconds
    };

    FlushPolicy flush_policy = FlushPolicy::BATCH;
    size_t      batch_size = 1000;
    uint64_t    flush_interval_us = 1000;
    bool        fsync_on_flush = true;
    bool        enable_checksum = true;
};

class WAL {
public:
    explicit WAL(const std::string& path, const WalConfig& config = WalConfig{});
    ~WAL();

    // ── Write ────────────────────────────────

    // Append a command to the WAL buffer (does NOT fsync immediately)
    void log_command(const PaymentCommandFast& cmd);

    // Append an event to the WAL buffer
    void log_event(const EngineEvent& event);

    // Write a snapshot of the current ledger state
    void write_snapshot(const std::string& snapshot_data);

    // Flush buffered records to disk (called by batch policy or periodically)
    void flush();

    // Force fsync (full durability guarantee)
    void sync();

    // ── Read / Recovery (§12.4) ──────────────

    struct Entry {
        WALEntryType type;
        std::string  data;
        bool         checksum_valid = true;
    };

    // Read all WAL entries, verifying checksums
    std::vector<Entry> read_all();

    // Get the last sequence ID from the WAL
    uint64_t last_sequence_id() const;

    // ── Management ────────────────────────────

    // Clear the WAL (after successful snapshot)
    void clear();

    // Current file size in bytes
    size_t size_bytes() const;

    // Number of buffered (not yet flushed) records
    size_t buffered_count() const;

    // Update configuration at runtime
    void set_config(const WalConfig& config) { config_ = config; }

private:
    std::string path_;
    std::fstream file_;
    std::mutex mutex_;
    WalConfig config_;

    uint64_t last_seq_id_{0};
    size_t   buffered_records_{0};

    // §11.4: Write a record with checksum header
    void write_entry(WALEntryType type, const std::string& data);

    // §11.5: Compute XXH64 checksum of data
    static uint64_t compute_checksum(const uint8_t* data, size_t len);

    // Simple XXH64 implementation (avoid external dependency for V2)
    static uint64_t xxh64(const uint8_t* data, size_t len, uint64_t seed = 0);
};

} // namespace engine
} // namespace aspira
