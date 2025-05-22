package main

import (
	"fmt"
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/kasbench/globeco-fix-engine/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := config.NewLogger(cfg.AppEnv)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	logger.Info("FIX Engine starting up", zap.String("env", cfg.AppEnv))

	// Run database migrations
	if err := config.RunMigrations(cfg.Postgres); err != nil {
		logger.Fatal("database migration failed", zap.Error(err))
	}
	logger.Info("Database migrations applied successfully")

	// Set up HTTP server and metrics
	mux := http.NewServeMux()
	config.RegisterMetricsHandler(mux)

	// Example: add a health check endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := ":" + fmt.Sprint(cfg.HTTPPort)
	logger.Info("HTTP server listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatal("server exited", zap.Error(err))
	}
}
