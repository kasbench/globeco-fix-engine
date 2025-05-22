package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// Config holds all configuration for the FIX Engine microservice.
type Config struct {
	AppEnv      string
	HTTPPort    int
	Kafka       KafkaConfig
	Postgres    PostgresConfig
	SecuritySvc ServiceConfig
	PricingSvc  ServiceConfig
}

type KafkaConfig struct {
	Brokers        []string
	OrdersTopic    string
	FillsTopic     string
	ConsumerGroup  string
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ServiceConfig struct {
	Host string
	Port int
}

// LoadConfig loads configuration from environment variables and config files.
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()

	// Set default values
	viper.SetDefault("AppEnv", "development")
	viper.SetDefault("HTTPPort", 8085)
	viper.SetDefault("Kafka.Brokers", []string{"globeco-execution-service-kafka:9092"})
	viper.SetDefault("Kafka.OrdersTopic", "orders")
	viper.SetDefault("Kafka.FillsTopic", "fills")
	viper.SetDefault("Kafka.ConsumerGroup", "fix_engine")
	viper.SetDefault("Postgres.Host", "globeco-fix-engine-postgresql")
	viper.SetDefault("Postgres.Port", 5437)
	viper.SetDefault("Postgres.User", "postgres")
	viper.SetDefault("Postgres.Password", "")
	viper.SetDefault("Postgres.DBName", "postgres")
	viper.SetDefault("Postgres.SSLMode", "disable")
	viper.SetDefault("SecuritySvc.Host", "globeco-security-service")
	viper.SetDefault("SecuritySvc.Port", 8000)
	viper.SetDefault("PricingSvc.Host", "globeco-pricing-service")
	viper.SetDefault("PricingSvc.Port", 8083)

	// Read config file if present
	err := viper.ReadInConfig()
	if err != nil {
		// Only error if the config file is present but invalid
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return &cfg, nil
} 