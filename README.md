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
