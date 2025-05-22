package main

import (
	"fmt"
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/kasbench/globeco-fix-engine/internal/api"
	"github.com/kasbench/globeco-fix-engine/internal/config"
	"github.com/kasbench/globeco-fix-engine/internal/repository"
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

	// Open DB connection
	db, err := config.OpenDB(cfg.Postgres)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Set up repository
	repo := repository.NewExecutionRepository(db)

	// Set up chi router
	r := chi.NewRouter()

	// Register API routes
	execAPI := api.NewExecutionAPI(repo)
	execAPI.RegisterRoutes(r)

	// Register metrics and health endpoints
	r.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		config.RegisterMetricsHandler(http.DefaultServeMux)
		http.DefaultServeMux.ServeHTTP(w, req)
	}))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := ":" + fmt.Sprint(cfg.HTTPPort)
	logger.Info("HTTP server listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, r); err != nil {
		logger.Fatal("server exited", zap.Error(err))
	}
}
