package config

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitOTel sets up OpenTelemetry tracing and metrics with OTLP gRPC exporters.
// Returns a shutdown function to flush and close the providers.
func InitOTel(ctx context.Context, cfg *Config) (func(context.Context) error, error) {
	// Build resource attributes
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.OTEL.ServiceName),
		semconv.ServiceVersionKey.String(cfg.OTEL.ServiceVersion),
		semconv.ServiceNamespaceKey.String(cfg.OTEL.ServiceNamespace),
	}

	// Parse additional resource attributes from environment
	if cfg.OTEL.ResourceAttributes != "" {
		for _, pair := range strings.Split(cfg.OTEL.ResourceAttributes, ",") {
			pair = strings.TrimSpace(pair)
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				attrs = append(attrs, attribute.String(key, value))
			}
		}
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, err
	}

	// Traces exporter
	traceOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTEL.TraceEndpoint),
	}
	if cfg.OTEL.Insecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
	}
	traceExp, err := otlptracegrpc.New(ctx, traceOpts...)
	if err != nil {
		return nil, err
	}
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExp),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	// Metrics exporter
	metricOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTEL.MetricEndpoint),
	}
	if cfg.OTEL.Insecure {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	}
	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		return nil, err
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExp)),
		metric.WithResource(res),
	)
	otel.SetMeterProvider(meterProvider)

	// Return shutdown function
	return func(ctx context.Context) error {
		err1 := tracerProvider.Shutdown(ctx)
		err2 := meterProvider.Shutdown(ctx)
		if err1 != nil {
			return err1
		}
		return err2
	}, nil
}
