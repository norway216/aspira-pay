// Aspira Pay V2 — Lock-Free MPSC Command Queue Implementation
// Architecture doc §4.7.1: Ring Buffer / MPSC Queue / Batch Pull.

#include "engine/CommandQueue.h"
#include <cstring>
#include <thread>
#include <chrono>

namespace aspira {
namespace engine {

CommandQueue::CommandQueue(size_t capacity)
    : capacity_(capacity)
    , buffer_(capacity) {
    // Ensure capacity is power of 2
    if ((capacity & (capacity - 1)) != 0) {
        // Round up to next power of 2
        capacity_ = 1;
        while (capacity_ < capacity) capacity_ <<= 1;
        buffer_.resize(capacity_);
    }
}

bool CommandQueue::enqueue(const PaymentCommand& cmd) {
    size_t write_idx = write_index_.load(std::memory_order_relaxed);
    size_t read_idx = read_index_.load(std::memory_order_acquire);

    // Check if queue is full
    if (write_idx - read_idx >= capacity_) {
        return false; // Queue full
    }

    // Write to slot
    buffer_[index_mask(write_idx, capacity_)] = cmd;

    // Commit the write
    write_index_.store(write_idx + 1, std::memory_order_release);
    return true;
}

std::unique_ptr<PaymentCommand> CommandQueue::dequeue() {
    size_t read_idx = read_index_.load(std::memory_order_relaxed);
    size_t committed = write_index_.load(std::memory_order_acquire);

    if (read_idx >= committed) {
        return nullptr; // Queue empty
    }

    auto cmd = std::make_unique<PaymentCommand>(
        buffer_[index_mask(read_idx, capacity_)]);

    read_index_.store(read_idx + 1, std::memory_order_release);
    return cmd;
}

std::vector<PaymentCommand> CommandQueue::dequeue_batch(size_t max_count) {
    std::vector<PaymentCommand> result;
    result.reserve(max_count);

    for (size_t i = 0; i < max_count; i++) {
        auto cmd = dequeue();
        if (!cmd) break;
        result.push_back(std::move(*cmd));
    }

    return result;
}

std::vector<PaymentCommand> CommandQueue::drain() {
    std::vector<PaymentCommand> result;

    while (true) {
        auto cmd = dequeue();
        if (!cmd) break;
        result.push_back(std::move(*cmd));
    }

    return result;
}

bool CommandQueue::empty() const {
    return read_index_.load(std::memory_order_acquire) >=
           write_index_.load(std::memory_order_acquire);
}

size_t CommandQueue::size() const {
    size_t write = write_index_.load(std::memory_order_acquire);
    size_t read = read_index_.load(std::memory_order_acquire);
    return write - read;
}

} // namespace engine
} // namespace aspira
