package config

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger initializes a zap.Logger based on the environment and log level.
func NewLogger(appEnv, logLevel string) (*zap.Logger, error) {
	var config zap.Config

	if appEnv == "development" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Set log level based on configuration
	level, err := parseLogLevel(logLevel)
	if err != nil {
		// Default to info level if parsing fails
		level = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(level)

	return config.Build()
}

// parseLogLevel converts string log level to zapcore.Level
func parseLogLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	case "panic":
		return zapcore.PanicLevel, nil
	default:
		return zapcore.InfoLevel, nil
	}
}
