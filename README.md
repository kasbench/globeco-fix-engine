# GlobeCo FIX Engine

A high-performance, cloud-native microservice for processing financial trades using the FIX protocol. Implements clean architecture, domain-driven design, and robust observability. Designed for scalable deployment on Kubernetes.

---

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [API Documentation](#api-documentation)
- [Configuration](#configuration)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)
  - [Docker](#docker)
  - [Kubernetes](#kubernetes)
- [Observability](#observability)
- [Security](#security)
- [Contributing](#contributing)
- [License](#license)

---

## Overview
GlobeCo FIX Engine is a Go microservice that ingests trade orders from Kafka, processes fills, persists executions in PostgreSQL, and exposes a RESTful API for querying executions. It is part of the GlobeCo suite for benchmarking Kubernetes autoscaling.

## Architecture
- **Language:** Go 1.23+
- **Patterns:** Clean Architecture, DDD, modular layering
- **Key Components:**
  - `cmd/` - Main application entrypoint
  - `internal/` - Private application code (API, domain, repository, service, middleware, config, kafka)
  - `config/` - Configuration management
  - `repository/` - Data access layer (PostgreSQL via sqlx)
  - `service/` - Business logic (order intake, fill processing)
  - `middleware/` - HTTP middleware (logging, tracing, CORS)
  - `api/` - REST API handlers
  - `kafka/` - Kafka integration (order/fill topics)
  - `domain/` - Business models and DTOs
  - `migrations/` - Database schema migrations
  - `k8s/` - Kubernetes manifests
  - `documentation/` - OpenAPI specs, diagrams, requirements

## Features
- **Kafka Integration:** Consumes orders, produces fills, auto-creates topics
- **PostgreSQL Persistence:** Robust schema, migrations, and repository pattern
- **REST API:** Query executions, health checks, OpenAPI/Swagger UI
- **Observability:**
  - Structured logging (zap)
  - Prometheus metrics (`/metrics`)
  - OpenTelemetry tracing
- **Health Checks:** Liveness (`/healthz`), readiness (`/readyz`)
- **Config Management:** Viper-based, environment-driven
- **Testing:** Unit, integration, and Kafka tests with testcontainers
- **Security:** Input validation, CORS, environment-based secrets
- **Containerization:** Multi-arch Dockerfile, Kubernetes manifests, CI/CD
- **Timestamps:** All timestamps in API and Kafka messages are encoded as `float64` seconds since epoch (e.g., `1748345329.233793461`), not ISO8601 strings.

## API Documentation
- **OpenAPI schema:** [GET /openapi.json](http://localhost:8080/openapi.json)
- **Swagger UI:** [GET /swagger-ui/](http://localhost:8080/swagger-ui/)

### Main Endpoints
| Method | Path                      | Description                |
|--------|---------------------------|----------------------------|
| GET    | /api/v1/executions        | List all executions        |
| GET    | /api/v1/execution/{id}    | Get execution by ID        |
| GET    | /metrics                  | Prometheus metrics         |
| GET    | /healthz                  | Liveness/health check      |
| GET    | /readyz                   | Readiness check            |
| GET    | /openapi.json             | OpenAPI schema             |
| GET    | /swagger-ui/              | Swagger UI                 |

See Swagger UI for full schema and try-it-out.

### Timestamp Format
All timestamp fields in API and Kafka JSON messages are encoded as `float64` seconds since the Unix epoch (UTC). Example:

```json
{
  "receivedTimestamp": 1748345329.233793461,
  "sentTimestamp": 1748345329.247809878,
  "lastFilledTimestamp": 1748345329.300000000
}
```

This applies to all timestamp fields in both REST API responses and Kafka messages. Do **not** use ISO8601 or RFC3339 strings.

## Configuration
- Uses [Viper](https://github.com/spf13/viper) for config loading
- Supports environment variables and YAML config files
- Example config keys:
  - `APP_ENV` (development/production)
  - `HTTP_PORT` (default: 8080)
  - `POSTGRES_*` (host, port, user, password, dbname, sslmode)
  - `KAFKA_*` (brokers, topics, consumer group)
  - `SECURITY_SVC_*`, `PRICING_SVC_*` (host, port)
- See `config/` and sample config file for details

## Development
1. **Clone the repo:**
   ```sh
   git clone <repo-url>
   cd globeco-fix-engine
   ```
2. **Install Go 1.23+**
3. **Install dependencies:**
   ```sh
   go mod tidy
   ```
4. **Run database migrations:**
   ```sh
   go run cmd/fix-engine/main.go migrate
   ```
5. **Run the service:**
   ```sh
   go run cmd/fix-engine/main.go
   ```
6. **Access API:**
   - [http://localhost:8080/swagger-ui/](http://localhost:8080/swagger-ui/)

## Testing
- **Unit tests:**
  ```sh
  go test ./...
  ```
- **Integration tests:** Use testcontainers for PostgreSQL and Kafka
- **API tests:** See `internal/api/handler_test.go`
- **Coverage:** Aim for high coverage, especially for business logic and repository

## Deployment
### Docker
- Multi-architecture Dockerfile (Linux/amd64, Linux/arm64)
- Minimal distroless image, non-root user
- Exposes port 8080
- Build:
  ```sh
  docker buildx build --platform linux/amd64,linux/arm64 -t <your-image> .
  ```

### Kubernetes
- See `k8s/` for manifests
- **Deployment:**
  - Liveness, readiness, startup probes
  - Resource limits: 100m CPU, 200Mi memory
  - Scales 1-100 replicas (HPA)
- **Service:** ClusterIP on port 8080
- **Apply:**
  ```sh
  kubectl apply -f k8s/deployment.yaml
  kubectl apply -f k8s/service.yaml
  ```
- **Namespace:** `globeco` (must exist)

### CI/CD
- GitHub Actions workflow for multi-arch Docker builds and pushes
- Requires DockerHub credentials as secrets

## Observability
- **Metrics:** `/metrics` (Prometheus)
- **Tracing:** OpenTelemetry (stdout exporter by default)
- **Logging:** zap, structured, environment-aware

## Security
- Input validation on all endpoints
- CORS enabled (adjust for production)
- Secrets/config via environment variables

## Contributing
1. Fork the repo and create a feature branch
2. Write clear, tested, idiomatic Go code
3. Run all tests and linters before PR
4. Document new features in README and OpenAPI
5. Submit a pull request with a clear description

## License
See [LICENSE](LICENSE).
