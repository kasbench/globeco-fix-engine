package config

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RegisterMetricsHandler registers the Prometheus metrics handler on the given mux.
func RegisterMetricsHandler(mux *http.ServeMux) {
	mux.Handle("/metrics", promhttp.Handler())
}
