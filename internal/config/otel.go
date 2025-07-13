package config

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitOTel sets up OpenTelemetry tracing and metrics with OTLP gRPC exporters.
// Returns a shutdown function to flush and close the providers.
func InitOTel(ctx context.Context) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("globeco-fix-engine"),
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.ServiceNamespaceKey.String("globeco"),
		),
	)
	if err != nil {
		return nil, err
	}

	// Traces exporter
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("otel-collector-collector.monitoring.svc.cluster.local:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExp),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tracerProvider)

	// Metrics exporter
	metricExp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint("otel-collector-collector.monitoring.svc.cluster.local:4317"),
		otlpmetricgrpc.WithInsecure(),
	)
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
