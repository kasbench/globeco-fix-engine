# GlobeCo FIX Engine Architecture

## Overview

The GlobeCo FIX Engine is a Go-based microservice designed to process financial trades using the FIX protocol. It consumes trade orders from a Kafka topic, processes them, and produces fill messages to another Kafka topic. The service is built for cloud-native deployment on Kubernetes, with a focus on scalability, robustness, and observability.

---

## High-Level Architecture

```
+-------------------+         +-------------------+         +-------------------+
|                   |         |                   |         |                   |
|   Kafka Orders    |  ---->  |   FIX Engine      |  ---->  |   Kafka Fills     |
|   Topic           |         |   Microservice    |         |   Topic           |
|                   |         |                   |         |                   |
+-------------------+         +-------------------+         +-------------------+
                                      |
                                      v
                             +-------------------+
                             |   PostgreSQL DB   |
                             +-------------------+
```

---

## Key Components

- **Kafka Consumer/Producer**: Listens to the `orders` topic and sends messages to the `fills` topic. Handles delayed responses using a scheduler or job queue. If the Kafka fills topic does not exist, it is created with 20 partitions.
- **Persistent Storage (PostgreSQL)**: Stores all received orders, their processing state, and scheduled/delayed fills. Ensures robustness and enables recovery after failures or restarts.
- **REST API**: Exposes endpoints for CRUD operations on executions, as well as health and readiness endpoints for Kubernetes.
- **Service Layer**: Contains business logic for processing orders, scheduling fills, and managing state transitions.
- **Repository Layer**: Handles data access using `sqlx` for PostgreSQL.
- **Observability**: Implements structured logging (zap), metrics (Prometheus), and tracing (OpenTelemetry).
- **Configuration**: Uses Viper for environment-based configuration.
- **Kubernetes Deployment**: Deploys as a scalable, robust service with readiness/liveness probes and horizontal scaling.
- **Caching**: Implements a 1-minute TTL cache for ticker lookups from the Security Service to reduce external calls and improve performance.

---

## Main Processing Loops

The FIX Engine microservice runs two main processing loops concurrently:

### 1. Kafka Consumer Loop (Order Intake)
- Consumes messages from the `orders` Kafka topic.
- Maps incoming messages to the database schema.
- For fields not present in the message, applies the following defaults:
  - `is_open` is set to `True` (1)
  - `execution_status` is set to `'WORK'`
  - `ticker` is looked up from the Security Service (with 1-minute TTL cache)
  - `last_fill_timestamp` is `NULL`
  - `quantity_filled` is `0`
  - `next_fill_timestamp` is the current timestamp
  - `number_of_fills` is `0`
  - `total_amount` is `0`
  - `version` is `1`

### 2. Database Polling Loop (Fill Processing)
- Polls the database for records where `next_fill_timestamp <= CURRENT_TIMESTAMP` and `is_open = TRUE`.
- For each eligible record, processes a fill according to the business rules:
  1. **Fill Quantity Calculation:**
     - Let `quantity_remaining = quantity_ordered - quantity_filled`.
     - With a 10% probability, set fill quantity to `quantity_remaining` (full fill).
     - With a 5% probability, set fill quantity to `0` (no fill).
     - If `quantity_remaining <= 100`, set fill quantity to `quantity_remaining` (full fill for small orders).
     - If `quantity_remaining > 100`, randomly select one of the following (each with 20% probability):
       - 80% of `quantity_remaining`, rounded to whole units, max 10,000
       - 60% of `quantity_remaining`, rounded to whole units, max 10,000
       - 400% of `quantity_remaining`, rounded to whole units, max 10,000
       - 20% of `quantity_remaining`, rounded to whole units, max 10,000
       - 10% of `quantity_remaining`, rounded to whole units, max 10,000
     - **Note:** Probabilities are mutually exclusive and determined by generating a random number for each decision point in order.
  2. **Price Check:**
     - Calls the Pricing Service for the ticker.
     - If `trade_type` is `'BUY'` or `'COVER'` and the price is greater than `limit_price`, set fill quantity to `0`.
     - If `trade_type` is `'SELL'` or `'SHORT'` and the price is less than `limit_price`, set fill quantity to `0`.
  3. **Update Database:**
     - Increment `quantity_filled` by the fill quantity.
     - Increment `total_amount` by (fill quantity × price).
     - Increment `number_of_fills` by 1.
     - Set `last_fill_timestamp` to the current timestamp.
     - If `quantity_ordered == quantity_filled`, set `is_open` to `False` (0) and `execution_status` to `'FULL'`.
     - If `quantity_filled > 0` and `quantity_filled < quantity_ordered`, set `execution_status` to `'PART'`.
     - If `is_open` is `True`, set `next_fill_timestamp` to the current timestamp plus a random interval between 5 seconds and 2 minutes.
  4. **Publish Fill:**
     - Format the record as `ExecutionDTO` and publish to the `fills` Kafka topic.

---

## Concurrency Control Strategy

### Problem

Since multiple instances of the FIX Engine microservice may be running (for scalability and high availability), it is essential to ensure that only one instance retrieves and processes a given record from the database at a time. This prevents duplicate processing and ensures data consistency.

### Recommended Solution: PostgreSQL Row Locking

**Approach:**

Use PostgreSQL's `SELECT ... FOR UPDATE SKIP LOCKED` within a transaction to safely fetch and lock a record for processing. This pattern is robust, simple, and highly concurrent, making it ideal for distributed microservices.

**How it works:**
- Each service instance attempts to select a record for processing using:
  ```sql
  SELECT * FROM execution
  WHERE next_fill_timestamp <= CURRENT_TIMESTAMP
    AND is_open = TRUE
  FOR UPDATE SKIP LOCKED
  LIMIT 1;
  ```
- The selected row is locked for the current transaction, and any rows already locked by other transactions are skipped.
- Multiple service instances can safely run this query in parallel; each will get a different unlocked row.
- After processing, the service updates the record's status and related fields and commits the transaction.

**Sample Go (sqlx) Usage:**
```go
// Begin a transaction
tx, err := db.Beginx(ctx)
if err != nil { /* handle error */ }
defer tx.Rollback()

var exec Execution
err = tx.GetContext(ctx, &exec, `
    SELECT * FROM execution
    WHERE next_fill_timestamp <= CURRENT_TIMESTAMP
      AND is_open = TRUE
    FOR UPDATE SKIP LOCKED
    LIMIT 1
`)
if err == sql.ErrNoRows {
    // No available work
    return
} else if err != nil {
    // handle error
}

// ... process exec ...

// Update execution fields and commit transaction
_, err = tx.ExecContext(ctx, `
    UPDATE execution SET ... WHERE id = $1
`, exec.ID)
if err != nil { /* handle error */ }

tx.Commit()
```

**Benefits:**
- No external dependencies beyond PostgreSQL.
- Simple and reliable for most workloads.
- Works seamlessly with Kubernetes scaling.

### Alternative Approaches
- **Advisory Locks (PostgreSQL):** For fine-grained, application-level locking.
- **Distributed Locking (Redis):** For cross-database or language-agnostic coordination (usually unnecessary for single-DB setups).

---

## Execution Status Handling

The `execution_status` field tracks the state of each execution. Valid values and transitions include:
- `'WORK'`: Initial state when an order is received and open for fills.
- `'PART'`: Partially filled; some quantity has been filled, but the order is still open.
- `'FULL'`: Fully filled; the order is closed.

State transitions:
- On order intake: set to `'WORK'`.
- On partial fill: set to `'PART'`.
- On full fill: set to `'FULL'` and close the order (`is_open = False`).

---

## Rationale

- **Scalability:** Stateless processing with all state in PostgreSQL allows safe horizontal scaling.
- **Robustness:** Persistent storage and transactional locking ensure no duplicate or lost processing.
- **Observability:** Logging, metrics, and tracing provide full visibility into system behavior.
- **Cloud-Native:** Designed for Kubernetes with health checks, readiness probes, and environment-based configuration.

---

## Summary Table

| Concern                | Solution/Pattern                |
|------------------------|---------------------------------|
| Message Processing     | Kafka consumer/producer         |
| Delayed Fills          | DB-backed scheduler/worker      |
| Persistence            | PostgreSQL                      |
| Scalability            | Stateless pods, DB for state    |
| Robustness             | Idempotency, retries, outbox    |
| Observability          | zap, Prometheus, OpenTelemetry  |
| API                    | RESTful, versioned, validated   |
| Deployment             | Docker, Kubernetes, HPA         |
| Concurrency            | FOR UPDATE SKIP LOCKED          |
| Caching                | 1-min TTL for ticker lookups    |

---

For further details, see the [requirements](./requirements.md) document. 