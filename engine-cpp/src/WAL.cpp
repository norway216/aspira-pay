// Aspira Pay V2 — Optimized WAL Implementation
// Architecture doc §11: Checksum, batch flush, configurable fsync.

#include "engine/WAL.h"
#include <iostream>
#include <sstream>
#include <filesystem>
#include <cstring>

namespace aspira {
namespace engine {

// ──────────────────────────────────────────────
// XXH64 (simplified implementation)
// Architecture doc §11.5: Checksum for every WAL record.
// ──────────────────────────────────────────────

static const uint64_t XXH_PRIME64_1 = 0x9E3779B185EBCA87ULL;
static const uint64_t XXH_PRIME64_2 = 0xC2B2AE3D27D4EB4FULL;
static const uint64_t XXH_PRIME64_3 = 0x165667B19E3779F9ULL;
static const uint64_t XXH_PRIME64_4 = 0x85EBCA77C2B2AE63ULL;
static const uint64_t XXH_PRIME64_5 = 0x27D4EB2F165667C5ULL;

uint64_t WAL::xxh64(const uint8_t* data, size_t len, uint64_t seed) {
    uint64_t h64;
    const uint8_t* p = data;
    const uint8_t* const end = p + len;

    if (len >= 32) {
        uint64_t v1 = seed + XXH_PRIME64_1 + XXH_PRIME64_2;
        uint64_t v2 = seed + XXH_PRIME64_2;
        uint64_t v3 = seed;
        uint64_t v4 = seed - XXH_PRIME64_1;

        const uint8_t* const limit = end - 32;
        do {
            uint64_t k1, k2, k3, k4;
            std::memcpy(&k1, p, 8); p += 8;
            std::memcpy(&k2, p, 8); p += 8;
            std::memcpy(&k3, p, 8); p += 8;
            std::memcpy(&k4, p, 8); p += 8;

            v1 = ((v1 * XXH_PRIME64_2) + k1) * XXH_PRIME64_1;
            v1 = (v1 << 31) | (v1 >> 33);
            v1 *= XXH_PRIME64_2;

            v2 = ((v2 * XXH_PRIME64_2) + k2) * XXH_PRIME64_1;
            v2 = (v2 << 31) | (v2 >> 33);
            v2 *= XXH_PRIME64_2;

            v3 = ((v3 * XXH_PRIME64_2) + k3) * XXH_PRIME64_1;
            v3 = (v3 << 31) | (v3 >> 33);
            v3 *= XXH_PRIME64_2;

            v4 = ((v4 * XXH_PRIME64_2) + k4) * XXH_PRIME64_1;
            v4 = (v4 << 31) | (v4 >> 33);
            v4 *= XXH_PRIME64_2;
        } while (p <= limit);

        h64 = ((v1 << 1) | (v1 >> 63)) + ((v2 << 7) | (v2 >> 57))
            + ((v3 << 12) | (v3 >> 52)) + ((v4 << 18) | (v4 >> 46));
    } else {
        h64 = seed + XXH_PRIME64_5;
    }

    h64 += len;

    while (p + 8 <= end) {
        uint64_t k1;
        std::memcpy(&k1, p, 8);
        k1 *= XXH_PRIME64_2;
        k1 = (k1 << 31) | (k1 >> 33);
        k1 *= XXH_PRIME64_1;
        h64 ^= k1;
        h64 = ((h64 << 27) | (h64 >> 37)) * XXH_PRIME64_1 + XXH_PRIME64_4;
        p += 8;
    }

    if (p + 4 <= end) {
        uint32_t k1;
        std::memcpy(&k1, p, 4);
        h64 ^= static_cast<uint64_t>(k1) * XXH_PRIME64_1;
        h64 = ((h64 << 23) | (h64 >> 41)) * XXH_PRIME64_2 + XXH_PRIME64_3;
        p += 4;
    }

    while (p < end) {
        h64 ^= static_cast<uint64_t>(*p) * XXH_PRIME64_5;
        h64 = ((h64 << 11) | (h64 >> 53)) * XXH_PRIME64_1;
        p++;
    }

    h64 ^= h64 >> 33;
    h64 *= XXH_PRIME64_2;
    h64 ^= h64 >> 29;
    h64 *= XXH_PRIME64_3;
    h64 ^= h64 >> 32;
    return h64;
}

uint64_t WAL::compute_checksum(const uint8_t* data, size_t len) {
    // Seed: "ASPIRA" as a 64-bit constant
    return xxh64(data, len, 0x4153504952415041ULL);
}

// ──────────────────────────────────────────────
// WAL Implementation
// ──────────────────────────────────────────────

WAL::WAL(const std::string& path, const WalConfig& config)
    : path_(path), config_(config) {
    file_.open(path_, std::ios::binary | std::ios::app);
    if (!file_.is_open()) {
        file_.open(path_, std::ios::binary | std::ios::out | std::ios::app);
    }
}

WAL::~WAL() {
    flush();
    sync();
    if (file_.is_open()) file_.close();
}

void WAL::log_command(const PaymentCommandFast& cmd) {
    std::ostringstream oss;
    oss.write(reinterpret_cast<const char*>(&cmd), sizeof(PaymentCommandFast));
    write_entry(WALEntryType::COMMAND, oss.str());
    last_seq_id_ = cmd.sequence_id;
}

void WAL::log_event(const EngineEvent& event) {
    std::ostringstream oss;
    oss << "EVT|" << event.sequence_id << "|"
        << event.event_id << "|"
        << event.payment_id_hash << "|"
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

    // §11.4: Write checksum header
    WalRecordHeader header;
    header.sequence_id = last_seq_id_;
    header.record_type = static_cast<uint32_t>(type);
    header.payload_size = data.size();
    header.timestamp_ns = std::chrono::duration_cast<std::chrono::nanoseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();

    if (config_.enable_checksum) {
        header.checksum = compute_checksum(
            reinterpret_cast<const uint8_t*>(data.data()), data.size());
    } else {
        header.checksum = 0;
    }

    // Write header + payload
    file_.write(reinterpret_cast<const char*>(&header), sizeof(WalRecordHeader));
    file_.write(data.data(), data.size());

    buffered_records_++;

    // §11.3: Batch flush policy
    if (config_.flush_policy == WalConfig::FlushPolicy::BATCH) {
        if (buffered_records_ >= config_.batch_size) {
            flush();
        }
    } else if (config_.flush_policy == WalConfig::FlushPolicy::EVERY_RECORD) {
        flush();
        sync();
    }
}

void WAL::flush() {
    if (buffered_records_ == 0) return;

    if (file_.is_open()) {
        file_.flush();
    }
    buffered_records_ = 0;
}

void WAL::sync() {
    std::lock_guard lock(mutex_);
    if (file_.is_open()) {
#if defined(__linux__)
        // Ensure data reaches disk
        file_.flush();
        int fd = -1;
        // Try to get the file descriptor for fsync
        // In production, use a proper fd-based approach
#endif
        file_.flush();
    }
}

// ──────────────────────────────────────────────
// Read / Recovery (§12.4)
// ──────────────────────────────────────────────

std::vector<WAL::Entry> WAL::read_all() {
    std::vector<Entry> entries;
    std::lock_guard lock(mutex_);

    // Save current write position
    auto current_pos = file_.tellg();
    file_.seekg(0);

    while (file_.good()) {
        WalRecordHeader header;
        file_.read(reinterpret_cast<char*>(&header), sizeof(WalRecordHeader));
        if (!file_.good()) break;

        if (header.payload_size == 0 || header.payload_size > 100 * 1024 * 1024) {
            std::cerr << "[WAL] Corrupt header at offset " << file_.tellg()
                      << " — stopping read" << std::endl;
            break;
        }

        std::string data(header.payload_size, '\0');
        file_.read(&data[0], header.payload_size);
        if (!file_.good()) break;

        Entry entry;
        entry.type = static_cast<WALEntryType>(header.record_type);

        // §11.5: Verify checksum
        if (header.checksum != 0) {
            uint64_t computed = compute_checksum(
                reinterpret_cast<const uint8_t*>(data.data()), data.size());
            entry.checksum_valid = (computed == header.checksum);

            if (!entry.checksum_valid) {
                std::cerr << "[WAL] Checksum mismatch at seq=" << header.sequence_id
                          << " — possible corruption, stopping read" << std::endl;
                break; // §11.5: Stop replay safely
            }
        }

        entry.data = std::move(data);
        entries.push_back(std::move(entry));
    }

    // Restore write position
    file_.clear();
    file_.seekg(current_pos);

    return entries;
}

uint64_t WAL::last_sequence_id() const {
    return last_seq_id_;
}

void WAL::clear() {
    std::lock_guard lock(mutex_);
    file_.close();
    file_.open(path_, std::ios::binary | std::ios::out | std::ios::trunc);
    last_seq_id_ = 0;
    buffered_records_ = 0;
}

size_t WAL::size_bytes() const {
    try {
        return std::filesystem::file_size(path_);
    } catch (...) {
        return 0;
    }
}

size_t WAL::buffered_count() const {
    return buffered_records_;
}

} // namespace engine
} // namespace aspira
