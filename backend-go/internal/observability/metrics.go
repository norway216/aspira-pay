// Package observability provides Prometheus metrics for Aspira Pay.
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// API metrics
	APIRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "api_request_total",
			Help: "Total number of API requests",
		},
		[]string{"method", "path", "status"},
	)

	APIRequestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "api_request_latency_ms",
			Help:    "API request latency in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"method", "path"},
	)

	// Payment metrics
	PaymentCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "payment_created_total",
			Help: "Total number of payments created",
		},
	)

	PaymentCompletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "payment_completed_total",
			Help: "Total number of payments completed",
		},
	)

	PaymentFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "payment_failed_total",
			Help: "Total number of payments failed",
		},
	)

	PaymentRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "risk_rejected_total",
			Help: "Total number of payments rejected by risk engine",
		},
		[]string{"reason"},
	)

	// Engine metrics
	EngineCommandLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "engine_command_latency_us",
			Help:    "Engine command processing latency in microseconds",
			Buckets: []float64{10, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
	)

	EngineTPS = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "engine_tps",
			Help: "Current engine transactions per second",
		},
	)

	EngineErrorTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "engine_error_total",
			Help: "Total number of engine errors",
		},
	)

	// Settlement metrics
	SettlementLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "settlement_lag_seconds",
			Help: "Settlement processing lag in seconds",
		},
	)

	// Chain metrics
	ChainSubmitLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "chain_submit_latency_ms",
			Help:    "Chain submission latency in milliseconds",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
		},
	)

	ChainSubmitFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "chain_submit_failed_total",
			Help: "Total number of failed chain submissions",
		},
	)
)
