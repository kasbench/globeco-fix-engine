package config

import (
	"context"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// InitTracer sets up OpenTelemetry tracing with a stdout exporter (for development).
// Returns a shutdown function to flush and close the tracer provider.
func InitTracer(serviceName string) (func(context.Context) error, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	rsrc := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(rsrc),
	)
	otel.SetTracerProvider(tp)

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	log.Println("OpenTelemetry tracing initialized (stdout exporter)")
	return shutdown, nil
}
