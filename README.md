# GlobeCo FIX Engine

This project is a microservice for processing financial trades using the FIX protocol. It is implemented in Go, follows clean architecture principles, and is designed for scalable, cloud-native deployment.

## Features
- Kafka consumer/producer for order and fill topics
- PostgreSQL persistence (schema in `fix-engine.sql`)
- RESTful API (chi router)
- Structured logging (zap)
- Configuration management (viper)
- Testable, modular codebase

## Main Dependencies
- [github.com/go-chi/chi/v5](https://github.com/go-chi/chi) - HTTP router
- [github.com/jmoiron/sqlx](https://github.com/jmoiron/sqlx) - SQL database access
- [go.uber.org/zap](https://github.com/uber-go/zap) - Structured logging
- [github.com/spf13/viper](https://github.com/spf13/viper) - Configuration
- [github.com/stretchr/testify](https://github.com/stretchr/testify) - Testing

## Schema
The database schema is defined in [`fix-engine.sql`](./fix-engine.sql).

## Status
This project is under active development.

## Kubernetes & Docker Deployment

### Docker
- Multi-architecture Dockerfile (Linux/amd64 and Linux/arm64) in project root
- Uses distroless base for minimal image size
- Exposes port 8080

### Kubernetes
- See `k8s/` directory for manifests
- Deployment, Service, and HorizontalPodAutoscaler for `globeco-fix-engine` in the `globeco` namespace
- Health checks: `/healthz` (liveness/startup), `/readyz` (readiness)
- Liveness probe timeout: 240s
- Resource limits: 100m CPU, 200Mi memory per pod
- Scales from 1 to 100 replicas

#### Usage
1. Build and push your multi-arch Docker image
2. Update the image in `k8s/deployment.yaml`
3. Apply manifests:
   ```sh
   kubectl apply -f k8s/deployment.yaml
   kubectl apply -f k8s/service.yaml
   ```

See `k8s/README.md` for more details.
