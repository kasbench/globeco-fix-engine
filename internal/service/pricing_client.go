package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/kasbench/globeco-fix-engine/internal/config"
)

type PricingServiceClient struct {
	cfg config.ServiceConfig
}

func NewPricingServiceClient(cfg config.ServiceConfig) *PricingServiceClient {
	return &PricingServiceClient{cfg: cfg}
}

func (c *PricingServiceClient) GetPrice(ctx context.Context, ticker string) (float64, error) {
	url := fmt.Sprintf("http://%s:%d/api/v1/price/%s", c.cfg.Host, c.cfg.Port, ticker)
	log.Printf("PricingServiceClient.GetPrice: GET %s (ticker=%s)", url, ticker)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("pricing service returned status %d", resp.StatusCode)
	}
	var data struct {
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	return data.Price, nil
}
