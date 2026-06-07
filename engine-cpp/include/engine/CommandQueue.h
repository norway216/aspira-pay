// Aspira Pay V2 — Lock-Free MPSC Command Queue
// Multi-Producer, Single-Consumer ring buffer for engine commands.
// Architecture doc §4.7.1: Lock-Free Command Queue.

#pragma once

#include "Types.h"
#include <atomic>
#include <vector>
#include <memory>

namespace aspira {
namespace engine {

class CommandQueue {
public:
    // capacity must be a power of 2
    explicit CommandQueue(size_t capacity = 1024);

    // Producer: enqueue a command (lock-free, multiple producers safe)
    // Returns false if queue is full
    bool enqueue(const PaymentCommand& cmd);

    // Consumer: dequeue a command (single consumer)
    // Returns nullptr if queue is empty
    std::unique_ptr<PaymentCommand> dequeue();

    // Consumer: dequeue batch of commands (batch processing)
    // Returns up to max_count commands
    std::vector<PaymentCommand> dequeue_batch(size_t max_count);

    // Consumer: drain all available commands
    std::vector<PaymentCommand> drain();

    // Check if queue is empty
    bool empty() const;

    // Current size
    size_t size() const;

    // Total capacity
    size_t capacity() const { return capacity_; }

private:
    size_t capacity_;
    std::vector<PaymentCommand> buffer_;
    std::atomic<size_t> write_index_{0};   // Multiple producers advance this
    std::atomic<size_t> read_index_{0};    // Single consumer advances this
    std::atomic<size_t> committed_{0};     // Up to this index, all writes are complete

    static constexpr size_t index_mask(size_t idx, size_t cap) {
        return idx & (cap - 1);
    }
};

} // namespace engine
} // namespace aspira
