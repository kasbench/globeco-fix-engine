# GlobeCo FIX Engine Microservice: Execution Plan

## Step-by-Step Build Plan

1. **Project Initialization and Repository Setup**
   - Initialize a new Go module and set up the project directory structure according to clean architecture principles.
   - Configure version control (Git) and add a .gitignore file.
   - Set up Go module dependencies (chi, sqlx, zap, viper, testify, etc.).

2. **Configuration Management**
   - Implement configuration loading using Viper (supporting environment variables and config files).
   - Define configuration structs for Kafka, PostgreSQL, external services, and app settings.

3. **Logging and Observability Foundation**
   - Integrate zap for structured logging.
   - Set up Prometheus metrics endpoint and basic application metrics.
   - Integrate OpenTelemetry for distributed tracing (initial setup).

4. **Database Schema and Repository Layer**
   - Implement PostgreSQL connection management using sqlx.
   - Create repository interfaces and implementations for the execution table.
   - Add database migrations for the execution table and required indexes.

5. **Domain Models and DTOs**
   - Define Go structs for domain models and DTOs (Execution, ExecutionDTO, ExecutionPostDTO, etc.).
   - Implement mapping functions between database models and DTOs.

6. **Kafka Integration**
   - Set up Kafka consumer for the `orders` topic and producer for the `fills` topic.
   - Implement logic to create the `fills` topic with 20 partitions if it does not exist.
   - Configure consumer group and error handling.

7. **External Service Integration**
   - Implement client for the Security Service with a 1-minute TTL cache for ticker lookups.
   - Implement client for the Pricing Service.

8. **Business Logic: Service Layer**
   - Implement the order intake logic (Kafka consumer loop): map and persist incoming orders.
   - Implement the fill processing logic (DB polling loop): select eligible executions, calculate fill quantities, perform price checks, update state, and publish fills.
   - Apply concurrency control using `SELECT ... FOR UPDATE SKIP LOCKED`.
   - Handle execution_status transitions and all business rules.

9. **REST API Implementation**
   - Set up chi router and define RESTful endpoints for executions (list, get by ID).
   - Implement request validation and consistent error response formatting.
   - Add health and readiness endpoints for Kubernetes.
   
10. **Middleware and Utilities**
    - Implement HTTP middleware for logging, request tracing, and CORS.
    - Add utility functions as needed (e.g., random fill logic, time calculations).

11. **Testing**
    - Write unit tests for repository, service, and API layers using testify.
    - Add integration tests for Kafka and database interactions.
    - Ensure high test coverage and test for edge cases.

12. **Graceful Shutdown and Robustness**
    - Implement context-based cancellation and graceful shutdown for all components.
    - Ensure proper error handling and retries for transient failures.

13. **Containerization and Deployment**
    - Write a multi-stage Dockerfile for minimal image size.
    - Add Docker Compose for local development and integration testing.
    - Prepare Kubernetes manifests for deployment (Deployment, Service, ConfigMap, Secret, etc.).
    - Configure readiness and liveness probes.

14. **CI/CD Integration**
    - Set up CI pipeline for linting, testing, and building Docker images.
    - Add CD steps for deployment to Kubernetes (if applicable).

15. **Documentation**
    - Document API endpoints, configuration, and operational procedures.
    - Update architecture and requirements documentation as needed.

---

**Next Step:**
Review step 1: Project Initialization and Repository Setup. 