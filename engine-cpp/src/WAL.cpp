// Aspira Pay V2 — Write-Ahead Log Implementation
// Architecture doc §4.7.1: Sequential WAL for recovery.

#include "engine/WAL.h"
#include <iostream>
#include <sstream>
#include <filesystem>

namespace aspira {
namespace engine {

WAL::WAL(const std::string& path) : path_(path) {
    // Open file in append mode
    file_.open(path_, std::ios::binary | std::ios::app);
    if (!file_.is_open()) {
        std::cerr << "[WAL] Cannot open " << path_ << " — creating new" << std::endl;
        file_.open(path_, std::ios::binary | std::ios::out | std::ios::app);
    }
}

WAL::~WAL() {
    sync();
    if (file_.is_open()) {
        file_.close();
    }
}

void WAL::log_command(const PaymentCommand& cmd) {
    // Simple text format: CMD|seq_id|payment_id|from|to|src_amt|tgt_amt|fee|ts
    std::ostringstream oss;
    oss << "CMD|" << cmd.sequence_id << "|"
        << cmd.payment_id << "|"
        << cmd.from_account << "|"
        << cmd.to_account << "|"
        << cmd.source_amount << "|"
        << cmd.target_amount << "|"
        << cmd.fee_amount << "|"
        << cmd.timestamp << "\n";

    write_entry(WALEntryType::COMMAND, oss.str());
    last_seq_id_ = cmd.sequence_id;
}

void WAL::log_event(const EngineEvent& event) {
    std::ostringstream oss;
    oss << "EVT|" << event.sequence_id << "|"
        << event.event_id << "|"
        << event.payment_id << "|"
        << event.event_type << "|"
        << event.result << "|"
        << event.timestamp << "\n";

    write_entry(WALEntryType::EVENT, oss.str());
    last_seq_id_ = event.sequence_id;
}

void WAL::write_snapshot(const std::string& snapshot_data) {
    write_entry(WALEntryType::SNAPSHOT, snapshot_data);
}

void WAL::write_entry(WALEntryType type, const std::string& data) {
    std::lock_guard lock(mutex_);

    // Format: [type_byte][4-byte length][data]
    uint8_t type_byte = static_cast<uint8_t>(type);
    uint32_t len = data.size();

    file_.write(reinterpret_cast<const char*>(&type_byte), 1);
    file_.write(reinterpret_cast<const char*>(&len), 4);
    file_.write(data.c_str(), len);
}

std::vector<WAL::Entry> WAL::read_all() {
    std::vector<Entry> entries;
    std::lock_guard lock(mutex_);

    // Re-open for reading (save current position)
    auto current_pos = file_.tellg();
    file_.seekg(0);

    while (file_.good()) {
        uint8_t type_byte;
        uint32_t len;

        file_.read(reinterpret_cast<char*>(&type_byte), 1);
        if (!file_.good()) break;

        file_.read(reinterpret_cast<char*>(&len), 4);
        if (!file_.good()) break;

        std::string data(len, '\0');
        file_.read(&data[0], len);
        if (!file_.good()) break;

        Entry entry;
        entry.type = static_cast<WALEntryType>(type_byte);
        entry.data = data;
        entries.push_back(entry);
    }

    // Restore position
    file_.clear();
    file_.seekg(current_pos);

    return entries;
}

uint64_t WAL::last_sequence_id() const {
    return last_seq_id_;
}

void WAL::sync() {
    std::lock_guard lock(mutex_);
    if (file_.is_open()) {
        file_.flush();
    }
}

void WAL::clear() {
    std::lock_guard lock(mutex_);
    file_.close();
    file_.open(path_, std::ios::binary | std::ios::out | std::ios::trunc);
    last_seq_id_ = 0;
}

size_t WAL::size_bytes() const {
    try {
        return std::filesystem::file_size(path_);
    } catch (...) {
        return 0;
    }
}

} // namespace engine
} // namespace aspira
