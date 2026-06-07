// Aspira Pay V2 — Object Pool (§10.3)
// Pre-allocated pool for zero-allocation object reuse in the hot path.
// Thread-safe for single-producer use; use separate pools per thread.

#pragma once

#include <array>
#include <atomic>
#include <cstddef>

namespace aspira {
namespace engine {

template <typename T, size_t N>
class ObjectPool {
public:
    ObjectPool() {
        // Initialize free list: all indices are free initially
        for (size_t i = 0; i < N; ++i) {
            free_list_[i] = i;
        }
        free_head_.store(0, std::memory_order_relaxed);
    }

    // Acquire an object from the pool. Returns nullptr if pool exhausted.
    T* acquire() {
        size_t head = free_head_.load(std::memory_order_acquire);
        if (head >= N) return nullptr; // Pool exhausted

        size_t next = head + 1;
        free_head_.store(next, std::memory_order_release);
        return &pool_[free_list_[head]];
    }

    // Release an object back to the pool.
    void release(T* obj) {
        if (!obj) return;
        size_t idx = obj - &pool_[0];
        if (idx >= N) return; // Not from this pool

        // Reset object state
        *obj = T{};

        size_t head = free_head_.load(std::memory_order_acquire);
        if (head > 0) {
            size_t prev = head - 1;
            free_list_[prev] = idx;
            free_head_.store(prev, std::memory_order_release);
        }
    }

    // Number of objects currently in use
    size_t in_use() const {
        return N - free_head_.load(std::memory_order_relaxed);
    }

    // Total pool capacity
    constexpr size_t capacity() const { return N; }

    // Check if pool is exhausted
    bool exhausted() const {
        return free_head_.load(std::memory_order_relaxed) >= N;
    }

private:
    std::array<T, N> pool_;
    std::array<size_t, N> free_list_;
    std::atomic<size_t> free_head_;
};

} // namespace engine
} // namespace aspira
