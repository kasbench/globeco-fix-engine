package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kasbench/globeco-fix-engine/internal/domain"
	"github.com/kasbench/globeco-fix-engine/internal/repository"
	"github.com/segmentio/kafka-go"
)

// ExecutionService wires together the repository, Kafka, and external service clients.
type ExecutionService struct {
	Repo           repository.ExecutionRepository
	DB             *sqlx.DB
	OrdersConsumer *kafka.Reader
	FillsProducer  *kafka.Writer
	SecurityClient *SecurityServiceClient
	PricingClient  *PricingServiceClient
}

// NewExecutionService constructs a new ExecutionService.
func NewExecutionService(
	repo repository.ExecutionRepository,
	db *sqlx.DB,
	ordersConsumer *kafka.Reader,
	fillsProducer *kafka.Writer,
	securityClient *SecurityServiceClient,
	pricingClient *PricingServiceClient,
) *ExecutionService {
	return &ExecutionService{
		Repo:           repo,
		DB:             db,
		OrdersConsumer: ordersConsumer,
		FillsProducer:  fillsProducer,
		SecurityClient: securityClient,
		PricingClient:  pricingClient,
	}
}

// StartOrderIntakeLoop consumes messages from the orders topic, maps and persists them to the database.
// Uses the Security Service client to look up tickers and applies all default field rules.
func (s *ExecutionService) StartOrderIntakeLoop(ctx context.Context) {
	for {
		m, err := s.OrdersConsumer.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // context cancelled
			}
			log.Printf("error reading Kafka message: %v", err)
			continue
		}

		var postDTO domain.ExecutionDTO
		if err := json.Unmarshal(m.Value, &postDTO); err != nil {
			log.Printf("error unmarshalling order: %v", err)
			continue
		}

		ticker, err := s.SecurityClient.GetTickerBySecurityID(ctx, postDTO.SecurityID)
		if err != nil {
			log.Printf("error looking up ticker: %v", err)
			continue
		}

		now := time.Now().UTC()
		var limitPricePtr *float64
		if postDTO.LimitPrice != nil {
			if *postDTO.LimitPrice > -0.0001 && *postDTO.LimitPrice < 0.0001 {
				limitPricePtr = nil
			} else {
				limitPricePtr = postDTO.LimitPrice
			}
		}
		exec := &repository.Execution{
			ExecutionServiceID: postDTO.ID, // This should be the order ID from the message if present
			IsOpen:             true,
			ExecutionStatus:    "WORK",
			TradeType:          postDTO.TradeType,
			Destination:        postDTO.Destination,
			SecurityID:         postDTO.SecurityID,
			Ticker:             ticker,
			QuantityOrdered:    postDTO.QuantityOrdered,
			LimitPrice:         sqlNullFloat64(limitPricePtr),
			ReceivedTimestamp:  postDTO.ReceivedTimestamp.Time(),
			SentTimestamp:      postDTO.SentTimestamp.Time(),
			LastFillTimestamp:  sqlNullTime(nil),
			QuantityFilled:     0,
			NextFillTimestamp:  sqlNullTime(&now),
			NumberOfFills:      0,
			TotalAmount:        0,
			Version:            postDTO.Version,
		}

		if err := s.Repo.Create(ctx, exec); err != nil {
			log.Printf("error saving execution: %v", err)
			continue
		}
		// Kafka-go commits automatically when using ReadMessage
		log.Printf("order ingested: order_id=%d ticker=%s", exec.ExecutionServiceID, exec.Ticker)
	}
}

func sqlNullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

func sqlNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// StartFillProcessingLoop polls the database for eligible executions and processes fills.
// Uses FOR UPDATE SKIP LOCKED for concurrency control. Publishes fills to the fills topic.
func (s *ExecutionService) StartFillProcessingLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Poll for eligible execution
			exec, err := s.Repo.PollNextForFill(ctx)
			if err != nil {
				if err == sql.ErrNoRows {
					continue // nothing to process
				}
				log.Printf("error polling for fill: %v", err)
				continue
			}

			quantityRemaining := exec.QuantityOrdered - exec.QuantityFilled
			fillQty := calculateFillQuantity(quantityRemaining)

			// Price check
			price, err := s.PricingClient.GetPrice(ctx, exec.Ticker)
			log.Printf("price received: price=%.4f", price)
			if err != nil {
				log.Printf("error getting price: %v", err)
				continue
			}
			if (exec.TradeType == "BUY" || exec.TradeType == "COVER") && exec.LimitPrice.Valid && price > exec.LimitPrice.Float64 {
				fillQty = 0
			}
			if (exec.TradeType == "SELL" || exec.TradeType == "SHORT") && exec.LimitPrice.Valid && price < exec.LimitPrice.Float64 {
				fillQty = 0
			}

			// Cap fillQty to quantityRemaining
			if fillQty > quantityRemaining {
				fillQty = quantityRemaining
			}

			// Update execution
			exec.QuantityFilled += fillQty
			exec.TotalAmount += fillQty * price
			exec.NumberOfFills += 1
			now := time.Now().UTC()
			exec.LastFillTimestamp = sqlNullTime(&now)
			if exec.QuantityFilled >= exec.QuantityOrdered {
				exec.IsOpen = false
				exec.ExecutionStatus = "FULL"
			} else if fillQty > 0 {
				exec.ExecutionStatus = "PART"
			}
			if exec.IsOpen {
				delta := time.Duration(rand.Intn(115)+5) * time.Second // 5s to 2m
				next := now.Add(delta)
				exec.NextFillTimestamp = sqlNullTime(&next)
			}

			if err := s.Repo.Update(ctx, exec); err != nil {
				log.Printf("error updating execution: %v", err)
				continue
			}

			// Publish fill to Kafka
			dto := domain.MapExecutionToDTO(exec)
			msg, err := json.Marshal(dto)
			if err != nil {
				log.Printf("error marshalling fill DTO: %v", err)
				continue
			}
			err = s.FillsProducer.WriteMessages(ctx, kafka.Message{Value: msg})
			if err != nil {
				log.Printf("error publishing fill: %v", err)
				continue
			}
			log.Printf("fill published: execution_service_id=%d fill_qty=%.2f price=%.4f", exec.ExecutionServiceID, fillQty, price)
		}
	}
}

func calculateFillQuantity(quantityRemaining float64) float64 {
	if quantityRemaining <= 0 {
		return 0
	}
	p := rand.Float64()
	if p < 0.10 {
		fill := quantityRemaining
		fill = float64(int64(fill)) // round to whole units
		if fill > 10000 {
			fill = 10000
		}
		return fill
	}
	if p < 0.15 {
		return 0 // 5% probability: no fill
	}
	if quantityRemaining <= 100 {
		return quantityRemaining
	}
	// For >100, pick one of 5 possibilities, each 20%
	choices := []float64{0.8, 0.6, 4.0, 0.2, 0.1}
	idx := rand.Intn(5)
	fill := quantityRemaining * choices[idx]
	fill = float64(int64(fill)) // round to whole units
	if fill > 10000 {
		fill = 10000
	}
	return fill
}
