package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDBWithContainer(t *testing.T) (*sqlx.DB, func()) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("failed to get mapped port: %v", err)
	}
	dsn := fmt.Sprintf("host=%s port=%s user=testuser password=testpass dbname=testdb sslmode=disable", host, port.Port())

	// Retry loop for connecting to DB
	var db *sqlx.DB
	for i := 0; i < 10; i++ {
		db, err = sqlx.Connect("postgres", dsn)
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("failed to connect to test db: %v", err)
	}

	// Apply schema
	schema, err := os.ReadFile("../../migrations/0001_init.up.sql")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("failed to read schema: %v", err)
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("failed to apply schema: %v", err)
	}

	cleanup := func() {
		db.Close()
		container.Terminate(ctx)
	}

	return db, cleanup
}

func TestExecutionRepository_CreateAndGetByID(t *testing.T) {
	db, cleanup := setupTestDBWithContainer(t)
	defer cleanup()
	repo := NewExecutionRepository(db)
	ctx := context.Background()
	exec := &Execution{
		ExecutionServiceID: 12345,
		IsOpen:             true,
		ExecutionStatus:    "WORK",
		TradeType:          "BUY",
		Destination:        "DEST",
		SecurityID:         "SECID123",
		Ticker:             "AAPL",
		QuantityOrdered:    100,
		LimitPrice:         sql.NullFloat64{Float64: 150.0, Valid: true},
		ReceivedTimestamp:  time.Now().UTC(),
		SentTimestamp:      time.Now().UTC(),
		LastFillTimestamp:  sql.NullTime{Valid: false},
		QuantityFilled:     0,
		NextFillTimestamp:  sql.NullTime{Valid: false},
		NumberOfFills:      0,
		TotalAmount:        0,
		Version:            1,
	}
	err := repo.Create(ctx, exec)
	assert.NoError(t, err)
	assert.NotZero(t, exec.ID)

	fetched, err := repo.GetByID(ctx, exec.ID)
	assert.NoError(t, err)
	assert.Equal(t, exec.ExecutionServiceID, fetched.ExecutionServiceID)
	assert.Equal(t, exec.Ticker, fetched.Ticker)
}
