// Package observability provides OpenTelemetry tracing setup.
package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer is the global tracer instance.
var Tracer trace.Tracer

// InitTracer initializes OpenTelemetry tracing.
// For Sandbox, we use a no-op tracer by default.
func InitTracer(serviceName, otlpEndpoint string) error {
	if otlpEndpoint == "" {
		// Use no-op tracer for local development
		Tracer = otel.Tracer(serviceName)
		return nil
	}

	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("2.0.0"),
		),
	)
	if err != nil {
		return err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	Tracer = tp.Tracer(serviceName)
	return nil
}

// WithTraceContext extracts trace context from HTTP headers (simplified).
func WithTraceContext(ctx context.Context, traceID, spanID string) context.Context {
	// In production, this would use W3C Trace Context propagation
	ctx = context.WithValue(ctx, CtxTraceID, traceID)
	return ctx
}
