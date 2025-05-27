package domain

import (
	"github.com/kasbench/globeco-fix-engine/internal/repository"
)

// Execution is the domain model (mirrors the DB model)
type Execution = repository.Execution

// ExecutionDTO is used for API responses and Kafka fills topic
// Maps to the execution table and includes all fields
// JSON tags use camelCase for API compatibility
type ExecutionDTO struct {
	ID                      int        `json:"id"`
	ExecutionServiceID      int        `json:"executionServiceId"`
	IsOpen                  bool       `json:"isOpen"`
	ExecutionStatus         string     `json:"executionStatus"`
	TradeType               string     `json:"tradeType"`
	Destination             string     `json:"destination"`
	SecurityID              string     `json:"securityId"`
	Ticker                  string     `json:"ticker"`
	QuantityOrdered         float64    `json:"quantity"`
	LimitPrice              *float64   `json:"limitPrice,omitempty"`
	ReceivedTimestamp       EpochTime  `json:"receivedTimestamp"`
	SentTimestamp           EpochTime  `json:"sentTimestamp"`
	LastFillTimestamp       *EpochTime `json:"lastFilledTimestamp,omitempty"`
	QuantityFilled          float64    `json:"quantityFilled"`
	AveragePrice            *float64   `json:"averagePrice,omitempty"`
	NumberOfFills           int16      `json:"numberOfFills"`
	TotalAmount             float64    `json:"totalAmount"`
	TradeServiceExecutionID *int       `json:"tradeServiceExecutionId,omitempty"`
	Version                 int        `json:"version"`
}

// ExecutionPostDTO is used for creating new executions (API or Kafka orders topic)
// type ExecutionPostDTO struct {
// 	ExecutionStatus         string   `json:"executionStatus"`
// 	TradeType               string   `json:"tradeType"`
// 	Destination             string   `json:"destination"`
// 	SecurityID              string   `json:"securityId"`
// 	Quantity                float64  `json:"quantity"`
// 	LimitPrice              *float64 `json:"limitPrice,omitempty"`
// 	TradeServiceExecutionID *int     `json:"tradeServiceExecutionId,omitempty"`
// 	Version                 int      `json:"version"`
// }

// MapExecutionToDTO maps a DB Execution to an ExecutionDTO
func MapExecutionToDTO(exec *Execution) *ExecutionDTO {
	var limitPrice *float64
	if exec.LimitPrice.Valid {
		limitPrice = &exec.LimitPrice.Float64
	}
	var lastFill *EpochTime
	if exec.LastFillTimestamp.Valid {
		t := EpochTimeFromTime(exec.LastFillTimestamp.Time)
		lastFill = &t
	}
	var avgPrice *float64
	if exec.QuantityFilled > 0 {
		tmp := exec.TotalAmount / exec.QuantityFilled
		val := float64(int64(tmp*10000)) / 10000 // round to 4 decimal places
		avgPrice = &val
	}
	return &ExecutionDTO{
		ID:                 exec.ID,
		ExecutionServiceID: exec.ExecutionServiceID,
		IsOpen:             exec.IsOpen,
		ExecutionStatus:    exec.ExecutionStatus,
		TradeType:          exec.TradeType,
		Destination:        exec.Destination,
		SecurityID:         exec.SecurityID,
		Ticker:             exec.Ticker,
		QuantityOrdered:    exec.QuantityOrdered,
		LimitPrice:         limitPrice,
		ReceivedTimestamp:  EpochTimeFromTime(exec.ReceivedTimestamp),
		SentTimestamp:      EpochTimeFromTime(exec.SentTimestamp),
		LastFillTimestamp:  lastFill,
		QuantityFilled:     exec.QuantityFilled,
		AveragePrice:       avgPrice,
		NumberOfFills:      exec.NumberOfFills,
		TotalAmount:        exec.TotalAmount,
		TradeServiceExecutionID: func() *int {
			if exec.TradeServiceExecutionID.Valid {
				val := int(exec.TradeServiceExecutionID.Int64)
				return &val
			}
			return nil
		}(),
		Version: exec.Version,
	}
}
