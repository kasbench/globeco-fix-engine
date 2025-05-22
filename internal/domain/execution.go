package domain

import (
	"time"

	"github.com/kasbench/globeco-fix-engine/internal/repository"
)

// Execution is the domain model (mirrors the DB model)
type Execution = repository.Execution

// ExecutionDTO is used for API responses and Kafka fills topic
// Maps to the execution table and includes all fields
// JSON tags use camelCase for API compatibility
type ExecutionDTO struct {
	ID                int        `json:"id"`
	OrderID           int        `json:"orderId"`
	IsOpen            bool       `json:"isOpen"`
	ExecutionStatus   string     `json:"executionStatus"`
	TradeType         string     `json:"tradeType"`
	Destination       string     `json:"destination"`
	SecurityID        string     `json:"securityId"`
	Ticker            string     `json:"ticker"`
	QuantityOrdered   float64    `json:"quantity"`
	LimitPrice        *float64   `json:"limitPrice,omitempty"`
	ReceivedTimestamp time.Time  `json:"receivedTimestamp"`
	SentTimestamp     time.Time  `json:"sentTimestamp"`
	LastFillTimestamp *time.Time `json:"lastFilledTimestamp,omitempty"`
	QuantityFilled    float64    `json:"quantityFilled"`
	AveragePrice      *float64   `json:"averagePrice,omitempty"`
	NumberOfFills     int16      `json:"numberOfFills"`
	TotalAmount       float64    `json:"totalAmount"`
	Version           int        `json:"version"`
}

// ExecutionPostDTO is used for creating new executions (API or Kafka orders topic)
type ExecutionPostDTO struct {
	ExecutionStatus string   `json:"executionStatus"`
	TradeType       string   `json:"tradeType"`
	Destination     string   `json:"destination"`
	SecurityID      string   `json:"securityId"`
	Quantity        float64  `json:"quantity"`
	LimitPrice      *float64 `json:"limitPrice,omitempty"`
	Version         int      `json:"version"`
}

// MapExecutionToDTO maps a DB Execution to an ExecutionDTO
func MapExecutionToDTO(exec *Execution) *ExecutionDTO {
	var limitPrice *float64
	if exec.LimitPrice.Valid {
		limitPrice = &exec.LimitPrice.Float64
	}
	var lastFill *time.Time
	if exec.LastFillTimestamp.Valid {
		lastFill = &exec.LastFillTimestamp.Time
	}
	var avgPrice *float64
	if exec.QuantityFilled > 0 {
		tmp := exec.TotalAmount / exec.QuantityFilled
		val := float64(int64(tmp*10000)) / 10000 // round to 4 decimal places
		avgPrice = &val
	}
	return &ExecutionDTO{
		ID:                exec.ID,
		OrderID:           exec.OrderID,
		IsOpen:            exec.IsOpen,
		ExecutionStatus:   exec.ExecutionStatus,
		TradeType:         exec.TradeType,
		Destination:       exec.Destination,
		SecurityID:        exec.SecurityID,
		Ticker:            exec.Ticker,
		QuantityOrdered:   exec.QuantityOrdered,
		LimitPrice:        limitPrice,
		ReceivedTimestamp: exec.ReceivedTimestamp,
		SentTimestamp:     exec.SentTimestamp,
		LastFillTimestamp: lastFill,
		QuantityFilled:    exec.QuantityFilled,
		AveragePrice:      avgPrice,
		NumberOfFills:     exec.NumberOfFills,
		TotalAmount:       exec.TotalAmount,
		Version:           exec.Version,
	}
}
