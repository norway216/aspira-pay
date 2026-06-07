// Aspira Pay V2 — Event Publisher
// Publishes engine events to NATS/Kafka/Redpanda message queue.
// Architecture doc §4.7.1: Event Publisher.

#pragma once

#include "Types.h"
#include <string>
#include <functional>
#include <vector>
#include <mutex>
#include <atomic>

namespace aspira {
namespace engine {

// Callback type for event publishing
using PublishCallback = std::function<void(const EngineEvent&)>;

class Publisher {
public:
    Publisher() = default;

    // Register a publish callback (e.g., NATS client, gRPC stream)
    void register_callback(PublishCallback callback);

    // Publish a single event
    void publish(const EngineEvent& event);

    // Publish multiple events in batch
    void publish_batch(const std::vector<EngineEvent>& events);

    // Set whether publishing is enabled
    void set_enabled(bool enabled) { enabled_ = enabled; }

    // Get total published event count
    uint64_t total_published() const { return total_published_; }

private:
    std::vector<PublishCallback> callbacks_;
    std::mutex mutex_;
    bool enabled_ = true;
    std::atomic<uint64_t> total_published_{0};
};

} // namespace engine
} // namespace aspira
