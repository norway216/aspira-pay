// Aspira Pay V2 — Engine Metrics (§19)
// Latency breakdown, queue depth, and throughput tracking.
// All counters are atomic for safe cross-thread access.
//
// §19.2: Latency breakdown dimensions:
//   decode → queue_wait → execute → wal_append → event_enqueue → total

#pragma once

#include <atomic>
#include <cstdint>
#include <array>

namespace aspira {
namespace engine {

class EngineMetrics {
public:
    // ── Throughput ──────────────────────────
    std::atomic<uint64_t> commands_processed{0};
    std::atomic<uint64_t> commands_rejected{0};
    std::atomic<uint64_t> commands_duplicated{0};
    std::atomic<uint64_t> commands_dropped{0};      // Queue full drops
    std::atomic<uint64_t> events_published{0};
    std::atomic<uint64_t> events_dropped{0};         // Event ring buffer full

    // ── Latency Histogram (nanoseconds) ──────
    // Simple bucketed histogram: <1us, 1-10us, 10-100us, 100us-1ms, >1ms
    static constexpr size_t NUM_BUCKETS = 5;
    static constexpr uint64_t BUCKET_BOUNDARIES[NUM_BUCKETS] = {
        1'000ULL,        // < 1 microsecond
        10'000ULL,       // 1-10 us
        100'000ULL,      // 10-100 us
        1'000'000ULL,    // 100 us - 1 ms
        10'000'000ULL    // 1-10 ms (P99 should be < 1ms per §2.1)
    };

    std::array<std::atomic<uint64_t>, NUM_BUCKETS + 1> latency_buckets{};

    // ── Queue ──────────────────────────────
    std::atomic<uint64_t> queue_depth_max{0};

    // ── WAL ────────────────────────────────
    std::atomic<uint64_t> wal_bytes_written{0};
    std::atomic<uint64_t> wal_sync_count{0};
    std::atomic<uint64_t> wal_sync_latency_ns{0};     // Total sync latency (for avg calc)

    // ── Snapshot ───────────────────────────
    std::atomic<uint64_t> snapshot_count{0};
    std::atomic<uint64_t> snapshot_duration_ns{0};     // Last snapshot duration
    std::atomic<uint64_t> last_snapshot_seq{0};

    // ── Recovery ───────────────────────────
    std::atomic<uint64_t> replay_duration_ns{0};
    std::atomic<uint64_t> replay_commands_count{0};

    // ── Account ────────────────────────────
    std::atomic<uint64_t> account_count{0};
    std::atomic<uint64_t> dedup_cache_size{0};

    // ── Methods ────────────────────────────

    // Record a latency sample into the histogram
    void record_latency(uint64_t total_ns) {
        for (size_t i = 0; i < NUM_BUCKETS; ++i) {
            if (total_ns < BUCKET_BOUNDARIES[i]) {
                latency_buckets[i].fetch_add(1, std::memory_order_relaxed);
                return;
            }
        }
        // >= largest boundary → last bucket
        latency_buckets[NUM_BUCKETS].fetch_add(1, std::memory_order_relaxed);
    }

    // Record detailed latency breakdown (§19.2)
    void record_latency_breakdown(uint64_t decode_ns, uint64_t queue_ns,
                                   uint64_t exec_ns, uint64_t wal_ns,
                                   uint64_t event_ns) {
        record_latency(decode_ns + queue_ns + exec_ns + wal_ns + event_ns);
    }

    // Update max queue depth (call from core thread)
    void observe_queue_depth(size_t depth) {
        uint64_t current = queue_depth_max.load(std::memory_order_relaxed);
        while (depth > current) {
            if (queue_depth_max.compare_exchange_weak(current, depth,
                    std::memory_order_release, std::memory_order_relaxed)) {
                break;
            }
        }
    }

    // Reset max queue depth (call after reporting)
    void reset_queue_depth_max() {
        queue_depth_max.store(0, std::memory_order_relaxed);
    }

    // ── Snapshot for Prometheus export ─────
    struct Snapshot {
        uint64_t commands_processed;
        uint64_t commands_rejected;
        uint64_t commands_duplicated;
        uint64_t commands_dropped;
        uint64_t events_published;
        uint64_t events_dropped;
        uint64_t queue_depth_max;
        uint64_t wal_bytes_written;
        uint64_t wal_sync_count;
        uint64_t account_count;
        uint64_t dedup_cache_size;
        uint64_t latency_counts[NUM_BUCKETS + 1];
    };

    Snapshot get_snapshot() const {
        Snapshot s;
        s.commands_processed  = commands_processed.load(std::memory_order_relaxed);
        s.commands_rejected   = commands_rejected.load(std::memory_order_relaxed);
        s.commands_duplicated = commands_duplicated.load(std::memory_order_relaxed);
        s.commands_dropped    = commands_dropped.load(std::memory_order_relaxed);
        s.events_published    = events_published.load(std::memory_order_relaxed);
        s.events_dropped      = events_dropped.load(std::memory_order_relaxed);
        s.queue_depth_max     = queue_depth_max.load(std::memory_order_relaxed);
        s.wal_bytes_written   = wal_bytes_written.load(std::memory_order_relaxed);
        s.wal_sync_count      = wal_sync_count.load(std::memory_order_relaxed);
        s.account_count       = account_count.load(std::memory_order_relaxed);
        s.dedup_cache_size    = dedup_cache_size.load(std::memory_order_relaxed);
        for (size_t i = 0; i <= NUM_BUCKETS; ++i) {
            s.latency_counts[i] = latency_buckets[i].load(std::memory_order_relaxed);
        }
        return s;
    }
};

} // namespace engine
} // namespace aspira
