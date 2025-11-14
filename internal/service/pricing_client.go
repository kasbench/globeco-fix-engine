package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/kasbench/globeco-fix-engine/internal/config"
	"go.uber.org/zap"
)

type PricingServiceClient struct {
	cfg    config.ServiceConfig
	logger *zap.Logger
}

func NewPricingServiceClient(cfg config.ServiceConfig, logger *zap.Logger) *PricingServiceClient {
	return &PricingServiceClient{cfg: cfg, logger: logger}
}

func (c *PricingServiceClient) GetPrice(ctx context.Context, ticker string) (float64, error) {
	url := fmt.Sprintf("http://%s:%d/api/v1/price/%s", c.cfg.Host, c.cfg.Port, ticker)
	c.logger.Debug("PricingServiceClient.GetPrice", zap.String("url", url), zap.String("ticker", ticker))
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
		var bodyBytes []byte
		bodyBytes, _ = io.ReadAll(resp.Body)
		log.Printf("PricingServiceClient.GetPrice: non-200 response %d, body: %s", resp.StatusCode, string(bodyBytes))
		return 0, fmt.Errorf("pricing service returned status %d", resp.StatusCode)
	}
	var data struct {
		ID     int     `json:"id"`
		Ticker string  `json:"ticker"`
		Date   string  `json:"date"`
		Open   float64 `json:"open"`
		Close  float64 `json:"close"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Volume int     `json:"volume"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	return data.Close, nil
}
