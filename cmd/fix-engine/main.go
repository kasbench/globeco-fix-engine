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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
	logger, err := config.NewLogger(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()
	logger.Info("FIX Engine starting up", zap.String("env", cfg.AppEnv), zap.String("logLevel", cfg.LogLevel))

	// Initialize OpenTelemetry (tracing and metrics)
	logger.Info("Initializing OpenTelemetry",
		zap.String("trace_endpoint", cfg.OTEL.TraceEndpoint),
		zap.String("metric_endpoint", cfg.OTEL.MetricEndpoint),
		zap.String("service_name", cfg.OTEL.ServiceName),
		zap.String("service_namespace", cfg.OTEL.ServiceNamespace),
		zap.String("resource_attributes", cfg.OTEL.ResourceAttributes),
		zap.Bool("insecure", cfg.OTEL.Insecure),
	)
	otelShutdown, err := config.InitOTel(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize OpenTelemetry: %v", err)
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			log.Printf("error shutting down OpenTelemetry: %v", err)
		}
	}()

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
	pricingClient := service.NewPricingServiceClient(cfg.PricingSvc, logger)

	// Set up ExecutionService
	execService := service.NewExecutionService(
		repo,
		db,
		ordersConsumer,
		fillsProducer,
		securityClient,
		pricingClient,
		logger,
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

	// Add OpenTelemetry HTTP middleware first for automatic tracing and metrics
	// Skip instrumentation for health and metrics endpoints to avoid noise
	r.Use(func(next http.Handler) http.Handler {
		otelHandler := otelhttp.NewMiddleware("globeco-fix-engine")(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			otelHandler.ServeHTTP(w, r)
		})
	})

	// Add CORS middleware
	r.Use(middleware.CORSMiddleware)

	// Add logging middleware
	r.Use(middleware.LoggingMiddleware(logger))

	// Register API routes
	execAPI := api.NewExecutionAPI(repo)
	execAPI.RegisterRoutes(r)

	// Serve OpenAPI spec
	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
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
			w.WriteHeader(http.StatusOK)
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

	// Register metrics endpoint
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		http.DefaultServeMux.ServeHTTP(w, r)
	})
	config.RegisterMetricsHandler(http.DefaultServeMux)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
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
