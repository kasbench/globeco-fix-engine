package config

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NOTE: Metrics are now exported via OpenTelemetry OTLP exporter to the collector.
// The /metrics endpoint is optional and not required for Prometheus scraping if using OTLP pipeline.
// You may remove this handler if not needed, or keep it for legacy Prometheus support.
func RegisterMetricsHandler(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())
}
