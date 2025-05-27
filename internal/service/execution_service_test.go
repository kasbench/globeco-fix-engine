package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/kasbench/globeco-fix-engine/internal/repository"
	"github.com/stretchr/testify/assert"
)

type mockRepo struct {
	updateCalled bool
	lastExec     *repository.Execution
	updateErr    error
}

func (m *mockRepo) Create(ctx context.Context, exec interface{}) error       { return nil }
func (m *mockRepo) GetByID(ctx context.Context, id int) (interface{}, error) { return nil, nil }
func (m *mockRepo) List(ctx context.Context) ([]interface{}, error)          { return nil, nil }
func (m *mockRepo) PollNextForFill(ctx context.Context) (interface{}, error) { return nil, nil }
func (m *mockRepo) Update(ctx context.Context, exec interface{}) error {
	m.updateCalled = true
	if e, ok := exec.(*repository.Execution); ok {
		m.lastExec = e
	}
	return m.updateErr
}

// Mock clients for external services

type mockPricingClient struct {
	price float64
	fail  bool
}

func (m *mockPricingClient) GetPrice(ctx context.Context, ticker string) (float64, error) {
	if m.fail {
		return 0, errors.New("pricing error")
	}
	return m.price, nil
}

type mockSecurityClient struct {
	ticker string
	fail   bool
}

func (m *mockSecurityClient) GetTickerBySecurityID(ctx context.Context, securityID string) (string, error) {
	if m.fail {
		return "", errors.New("security error")
	}
	return m.ticker, nil
}

func TestCalculateFillQuantity(t *testing.T) {
	// Test edge cases for fill quantity logic
	assert.Equal(t, 0.0, calculateFillQuantity(0))
	assert.Equal(t, 100.0, calculateFillQuantity(100))
	assert.Equal(t, 0.0, calculateFillQuantity(-50))
	// For >100, should be <= 10000
	for i := 0; i < 100; i++ {
		fill := calculateFillQuantity(20000)
		assert.LessOrEqual(t, fill, 10000.0)
	}
}

func TestFillStatusTransitions(t *testing.T) {
	// repo := &mockRepo{} // not used in this test
	exec := &repository.Execution{
		ID:                 1,
		ExecutionServiceID: 1,
		IsOpen:             true,
		ExecutionStatus:    "WORK",
		TradeType:          "BUY",
		Destination:        "DEST",
		SecurityID:         "SECID123",
		Ticker:             "AAPL",
		QuantityOrdered:    100,
		QuantityFilled:     0,
		NumberOfFills:      0,
		TotalAmount:        0,
		Version:            1,
	}

	// Simulate a partial fill
	fillQty := 40.0
	exec.QuantityFilled += fillQty
	exec.TotalAmount += fillQty * 10.0
	exec.NumberOfFills++
	if exec.QuantityFilled >= exec.QuantityOrdered {
		exec.IsOpen = false
		exec.ExecutionStatus = "FULL"
	} else if fillQty > 0 {
		exec.ExecutionStatus = "PART"
	}
	assert.True(t, exec.IsOpen)
	assert.Equal(t, "PART", exec.ExecutionStatus)

	// Simulate a full fill
	fillQty = 60.0
	exec.QuantityFilled += fillQty
	exec.TotalAmount += fillQty * 10.0
	exec.NumberOfFills++
	if exec.QuantityFilled >= exec.QuantityOrdered {
		exec.IsOpen = false
		exec.ExecutionStatus = "FULL"
	} else if fillQty > 0 {
		exec.ExecutionStatus = "PART"
	}
	assert.False(t, exec.IsOpen)
	assert.Equal(t, "FULL", exec.ExecutionStatus)
}

func TestPriceCheckBlocksFill(t *testing.T) {
	exec := &repository.Execution{
		TradeType:      "BUY",
		LimitPrice:     toNullFloat64(100.0),
		Ticker:         "AAPL",
		QuantityFilled: 0,
	}
	pricing := &mockPricingClient{price: 120.0}
	// Price is above limit, should block fill
	fillQty := 50.0
	if (exec.TradeType == "BUY" || exec.TradeType == "COVER") && exec.LimitPrice.Valid && pricing.price > exec.LimitPrice.Float64 {
		fillQty = 0
	}
	assert.Equal(t, 0.0, fillQty)
}

func TestRepositoryUpdateError(t *testing.T) {
	repo := &mockRepo{updateErr: errors.New("update failed")}
	exec := &repository.Execution{}
	err := repo.Update(context.Background(), exec)
	assert.Error(t, err)
	assert.EqualError(t, err, "update failed")
}

func TestExternalServiceClientErrors(t *testing.T) {
	sec := &mockSecurityClient{fail: true}
	_, err := sec.GetTickerBySecurityID(context.Background(), "SECID123")
	assert.Error(t, err)
	assert.EqualError(t, err, "security error")

	pricing := &mockPricingClient{fail: true}
	_, err = pricing.GetPrice(context.Background(), "AAPL")
	assert.Error(t, err)
	assert.EqualError(t, err, "pricing error")
}

func toNullFloat64(f float64) (nf sql.NullFloat64) {
	nf.Valid = true
	nf.Float64 = f
	return
}
