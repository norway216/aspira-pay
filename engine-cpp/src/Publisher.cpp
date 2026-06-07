// Aspira Pay V2 — Event Publisher Implementation
// Architecture doc §4.7.1: Event Publisher — NATS/Kafka/Redpanda.

#include "engine/Publisher.h"
#include <iostream>

namespace aspira {
namespace engine {

void Publisher::register_callback(PublishCallback callback) {
    std::lock_guard lock(mutex_);
    callbacks_.push_back(std::move(callback));
}

void Publisher::publish(const EngineEvent& event) {
    if (!enabled_) return;

    total_published_++;

    std::lock_guard lock(mutex_);
    for (const auto& cb : callbacks_) {
        try {
            cb(event);
        } catch (const std::exception& e) {
            std::cerr << "[Publisher] Callback error: " << e.what() << std::endl;
        }
    }
}

void Publisher::publish_batch(const std::vector<EngineEvent>& events) {
    if (!enabled_) return;

    for (const auto& event : events) {
        publish(event);
    }
}

} // namespace engine
} // namespace aspira
