// Aspira Pay V2 — SPSC Event Ring Buffer (§14.2)
// Single-Producer, Single-Consumer ring buffer for async event publishing.
// Decouples the engine core thread from the network publisher thread.
//
// Design:
//   Engine Core Thread (producer) → SPSC Queue → Publisher Thread (consumer) → NATS
//
// Properties:
//   - Lock-free SPSC (no contention between single producer and consumer)
//   - Pre-allocated, bounded
//   - Cache-line padded indices to prevent false sharing
//   - Backpressure: if full, events are dropped and counted

#pragma once

#include "Types.h"
#include <atomic>
#include <vector>
#include <cstddef>

namespace aspira {
namespace engine {

class EventRingBuffer {
public:
    // capacity must be a power of 2
    explicit EventRingBuffer(size_t capacity = 4096)
        : capacity_(capacity), mask_(capacity - 1),
          buffer_(capacity) {}

    // Producer: enqueue an event (called from Engine Core Thread)
    // Returns false if buffer is full → apply backpressure
    bool enqueue(const EngineEvent& event) {
        size_t write = write_index_.load(std::memory_order_relaxed);
        size_t read  = read_index_.load(std::memory_order_acquire);

        if (write - read >= capacity_) {
            dropped_count_.fetch_add(1, std::memory_order_relaxed);
            return false; // Buffer full — backpressure needed
        }

        buffer_[write & mask_] = event;
        write_index_.store(write + 1, std::memory_order_release);
        return true;
    }

    // Consumer: dequeue a batch of events (called from Publisher Thread)
    // Returns up to max_count events. Empty vector if none available.
    std::vector<EngineEvent> dequeue_batch(size_t max_count) {
        std::vector<EngineEvent> result;
        result.reserve(max_count);

        size_t read  = read_index_.load(std::memory_order_relaxed);
        size_t write = write_index_.load(std::memory_order_acquire);

        size_t available = write - read;
        if (available == 0) return result;

        size_t count = std::min(available, max_count);
        for (size_t i = 0; i < count; ++i) {
            result.push_back(std::move(buffer_[(read + i) & mask_]));
        }

        read_index_.store(read + count, std::memory_order_release);
        return result;
    }

    // Consumer: check if any events are available
    bool has_events() const {
        size_t write = write_index_.load(std::memory_order_acquire);
        size_t read  = read_index_.load(std::memory_order_relaxed);
        return write > read;
    }

    // Current queue depth
    size_t size() const {
        size_t write = write_index_.load(std::memory_order_acquire);
        size_t read  = read_index_.load(std::memory_order_relaxed);
        return write - read;
    }

    size_t capacity() const { return capacity_; }

    uint64_t dropped_count() const {
        return dropped_count_.load(std::memory_order_relaxed);
    }

private:
    size_t capacity_;
    size_t mask_;
    std::vector<EngineEvent> buffer_;

    // Cache-line separated to prevent false sharing between producer and consumer cores
    alignas(64) std::atomic<size_t> write_index_{0};
    alignas(64) std::atomic<size_t> read_index_{0};
    alignas(64) std::atomic<uint64_t> dropped_count_{0};
};

} // namespace engine
} // namespace aspira
