package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/go-chi/chi/v5"
	"github.com/kasbench/globeco-fix-engine/internal/api"
	"github.com/kasbench/globeco-fix-engine/internal/config"
	"github.com/kasbench/globeco-fix-engine/internal/kafka"
	"github.com/kasbench/globeco-fix-engine/internal/middleware"
	"github.com/kasbench/globeco-fix-engine/internal/repository"
	"github.com/kasbench/globeco-fix-engine/internal/service"
	"go.opentelemetry.io/otel"
)

func main() {
	// Root context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for SIGINT/SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

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

	// Set up Kafka
	if err := kafka.CreateFillsTopicIfNotExists(ctx, cfg.Kafka); err != nil {
		logger.Fatal("failed to ensure fills topic exists", zap.Error(err))
	}
	ordersConsumer := kafka.NewOrdersConsumer(cfg.Kafka, cfg.Kafka.ConsumerGroup)
	fillsProducer := kafka.NewFillsProducer(cfg.Kafka)
	defer ordersConsumer.Close()
	defer fillsProducer.Close()

	// Set up external service clients
	securityClient := service.NewSecurityServiceClient(cfg.SecuritySvc)
	pricingClient := service.NewPricingServiceClient(cfg.PricingSvc)

	// Set up ExecutionService
	execService := service.NewExecutionService(
		repo,
		db,
		ordersConsumer,
		fillsProducer,
		securityClient,
		pricingClient,
	)

	// Start order intake and fill processing loops in background goroutines
	var wg sync.WaitGroup
	orderIntakeCtx, orderIntakeCancel := context.WithCancel(ctx)
	fillProcessingCtx, fillProcessingCancel := context.WithCancel(ctx)
	wg.Add(2)
	go func() {
		defer wg.Done()
		execService.StartOrderIntakeLoop(orderIntakeCtx)
	}()
	go func() {
		defer wg.Done()
		execService.StartFillProcessingLoop(fillProcessingCtx)
	}()

	// Set up chi router
	r := chi.NewRouter()

	// Add CORS middleware (should be first)
	r.Use(middleware.CORSMiddleware)

	// Add logging middleware
	r.Use(middleware.LoggingMiddleware(logger))

	// Add tracing middleware (using global tracer for now)
	tracer := otel.Tracer("globeco-fix-engine")
	r.Use(middleware.TracingMiddleware(tracer))

	// Register API routes
	execAPI := api.NewExecutionAPI(repo)
	execAPI.RegisterRoutes(r)

	// Serve OpenAPI spec
	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		http.ServeFile(w, r, "documentation/fix-engine-openapi.json")
	})

	// Serve Swagger UI
	r.Get("/swagger-ui/*", func(w http.ResponseWriter, r *http.Request) {
		// Redirect root /swagger-ui/ to index.html
		if r.URL.Path == "/swagger-ui/" || r.URL.Path == "/swagger-ui" {
			http.Redirect(w, r, "/swagger-ui/index.html", http.StatusFound)
			return
		}
		if r.URL.Path == "/swagger-ui/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.17.12/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.17.12/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: '/openapi.json',
        dom_id: '#swagger-ui',
      });
    };
  </script>
</body>
</html>`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

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
	httpServer := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start HTTP server in a goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("HTTP server listening", zap.String("addr", addr))
		serverErr <- httpServer.ListenAndServe()
	}()

	// Wait for signal or server error
	select {
	case sig := <-sigs:
		logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
	case err := <-serverErr:
		logger.Error("server exited", zap.Error(err))
	}

	// Initiate graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}
	// Stop order intake and fill processing loops and wait for them to finish
	orderIntakeCancel()
	fillProcessingCancel()
	wg.Wait()
	logger.Info("Shutdown complete")
}
