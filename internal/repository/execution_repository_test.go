package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDBWithContainer(t *testing.T) (*sqlx.DB, func()) {
	ctx := context.Background()
	container, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("failed to get connection string: %v", err)
	}

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
		OrderID:           12345,
		IsOpen:            true,
		ExecutionStatus:   "WORK",
		TradeType:         "BUY",
		Destination:       "DEST",
		SecurityID:        "SECID123",
		Ticker:            "AAPL",
		QuantityOrdered:   100,
		LimitPrice:        sql.NullFloat64{Float64: 150.0, Valid: true},
		ReceivedTimestamp: time.Now().UTC(),
		SentTimestamp:     time.Now().UTC(),
		LastFillTimestamp: sql.NullTime{Valid: false},
		QuantityFilled:    0,
		NextFillTimestamp: sql.NullTime{Valid: false},
		NumberOfFills:     0,
		TotalAmount:       0,
		Version:           1,
	}
	err := repo.Create(ctx, exec)
	assert.NoError(t, err)
	assert.NotZero(t, exec.ID)

	fetched, err := repo.GetByID(ctx, exec.ID)
	assert.NoError(t, err)
	assert.Equal(t, exec.OrderID, fetched.OrderID)
	assert.Equal(t, exec.Ticker, fetched.Ticker)
}
