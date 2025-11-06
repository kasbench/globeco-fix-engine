package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// Execution represents a row in the execution table.
type Execution struct {
	ID                      int             `db:"id"`
	ExecutionServiceID      int             `db:"execution_service_id"`
	IsOpen                  bool            `db:"is_open"`
	ExecutionStatus         string          `db:"execution_status"`
	TradeType               string          `db:"trade_type"`
	Destination             string          `db:"destination"`
	SecurityID              string          `db:"security_id"`
	Ticker                  string          `db:"ticker"`
	QuantityOrdered         float64         `db:"quantity_ordered"`
	LimitPrice              sql.NullFloat64 `db:"limit_price"`
	ReceivedTimestamp       time.Time       `db:"received_timestamp"`
	SentTimestamp           time.Time       `db:"sent_timestamp"`
	LastFillTimestamp       sql.NullTime    `db:"last_fill_timestamp"`
	QuantityFilled          float64         `db:"quantity_filled"`
	NextFillTimestamp       sql.NullTime    `db:"next_fill_timestamp"`
	NumberOfFills           int16           `db:"number_of_fills"`
	TotalAmount             float64         `db:"total_amount"`
	TradeServiceExecutionID sql.NullInt64   `db:"trade_service_execution_id"`
	Version                 int             `db:"version"`
}

// ExecutionRepository defines methods for interacting with the execution table.
type ExecutionRepository interface {
	Create(ctx context.Context, exec *Execution) error
	GetByID(ctx context.Context, id int) (*Execution, error)
	List(ctx context.Context) ([]*Execution, error)
	PollNextForFill(ctx context.Context) (*Execution, error)
	Update(ctx context.Context, exec *Execution) error
}

type executionRepository struct {
	db *sqlx.DB
}

func NewExecutionRepository(db *sqlx.DB) ExecutionRepository {
	return &executionRepository{db: db}
}

func (r *executionRepository) Create(ctx context.Context, exec *Execution) error {
	query := `INSERT INTO execution (
		execution_service_id, is_open, execution_status, trade_type, destination, security_id, ticker,
		quantity_ordered, limit_price, received_timestamp, sent_timestamp, last_fill_timestamp,
		quantity_filled, next_fill_timestamp, number_of_fills, total_amount, trade_service_execution_id, version
	) VALUES (
		:execution_service_id, :is_open, :execution_status, :trade_type, :destination, :security_id, :ticker,
		:quantity_ordered, :limit_price, :received_timestamp, :sent_timestamp, :last_fill_timestamp,
		:quantity_filled, :next_fill_timestamp, :number_of_fills, :total_amount, :trade_service_execution_id, :version
	) RETURNING id`
	rows, err := r.db.NamedQueryContext(ctx, query, exec)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return rows.Scan(&exec.ID)
	}
	return sql.ErrNoRows
}

func (r *executionRepository) GetByID(ctx context.Context, id int) (*Execution, error) {
	var exec Execution
	query := `SELECT * FROM execution WHERE id = $1`
	err := r.db.GetContext(ctx, &exec, query, id)
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

func (r *executionRepository) List(ctx context.Context) ([]*Execution, error) {
	var execs []*Execution
	query := `SELECT * FROM execution`
	err := r.db.SelectContext(ctx, &execs, query)
	if err != nil {
		return nil, err
	}
	return execs, nil
}

// PollNextForFill selects the next eligible execution for fill processing using FOR UPDATE SKIP LOCKED.
func (r *executionRepository) PollNextForFill(ctx context.Context) (*Execution, error) {
	var exec Execution
	query := `SELECT * FROM execution
	WHERE next_fill_timestamp <= NOW()
	  AND is_open
	FOR UPDATE SKIP LOCKED
	LIMIT 1`
	err := r.db.GetContext(ctx, &exec, query)
	if err != nil {
		return nil, err
	}
	return &exec, nil
}

func (r *executionRepository) Update(ctx context.Context, exec *Execution) error {
	query := `UPDATE execution SET
		execution_service_id = :execution_service_id,
		is_open = :is_open,
		execution_status = :execution_status,
		trade_type = :trade_type,
		destination = :destination,
		security_id = :security_id,
		ticker = :ticker,
		quantity_ordered = :quantity_ordered,
		limit_price = :limit_price,
		received_timestamp = :received_timestamp,
		sent_timestamp = :sent_timestamp,
		last_fill_timestamp = :last_fill_timestamp,
		quantity_filled = :quantity_filled,
		next_fill_timestamp = :next_fill_timestamp,
		number_of_fills = :number_of_fills,
		total_amount = :total_amount,
		trade_service_execution_id = :trade_service_execution_id,
		version = :version
	WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, exec)
	return err
}
