package config

import (
	"go.uber.org/zap"
)

// NewLogger initializes a zap.Logger based on the environment.
func NewLogger(appEnv string) (*zap.Logger, error) {
	if appEnv == "development" {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
