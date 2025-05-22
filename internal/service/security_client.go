package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kasbench/globeco-fix-engine/internal/config"
)

type SecurityServiceClient struct {
	cfg   config.ServiceConfig
	cache map[string]cachedTicker
	mu    sync.Mutex
}

type cachedTicker struct {
	ticker    string
	expiresAt time.Time
}

func NewSecurityServiceClient(cfg config.ServiceConfig) *SecurityServiceClient {
	return &SecurityServiceClient{
		cfg:   cfg,
		cache: make(map[string]cachedTicker),
	}
}

func (c *SecurityServiceClient) GetTickerBySecurityID(ctx context.Context, securityID string) (string, error) {
	c.mu.Lock()
	if entry, ok := c.cache[securityID]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.Unlock()
		return entry.ticker, nil
	}
	c.mu.Unlock()

	url := fmt.Sprintf("http://%s:%d/api/v1/security/%s", c.cfg.Host, c.cfg.Port, securityID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("security service returned status %d", resp.StatusCode)
	}
	var data struct {
		Ticker string `json:"ticker"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	c.mu.Lock()
	c.cache[securityID] = cachedTicker{
		ticker:    data.Ticker,
		expiresAt: time.Now().Add(time.Minute),
	}
	c.mu.Unlock()
	return data.Ticker, nil
}
