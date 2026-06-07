// Aspira Pay V2 — Write-Ahead Log
// Sequential WAL for crash recovery and audit trail.
// Architecture doc §4.7.1: WAL Log — command log / event log / snapshot.

#pragma once

#include "Types.h"
#include <string>
#include <vector>
#include <fstream>
#include <mutex>

namespace aspira {
namespace engine {

class WAL {
public:
    explicit WAL(const std::string& path);
    ~WAL();

    // Append a command to the WAL
    void log_command(const PaymentCommand& cmd);

    // Append an event to the WAL
    void log_event(const EngineEvent& event);

    // Write a snapshot of the current ledger state
    void write_snapshot(const std::string& snapshot_data);

    // Read all WAL entries (for replay on startup)
    struct Entry {
        WALEntryType type;
        std::string  data;
    };
    std::vector<Entry> read_all();

    // Get the last sequence ID from the WAL
    uint64_t last_sequence_id() const;

    // Sync to disk
    void sync();

    // Clear the WAL (after successful snapshot)
    void clear();

    // Current file size
    size_t size_bytes() const;

private:
    std::string path_;
    std::fstream file_;
    std::mutex mutex_;
    uint64_t last_seq_id_{0};

    void write_entry(WALEntryType type, const std::string& data);
};

} // namespace engine
} // namespace aspira
