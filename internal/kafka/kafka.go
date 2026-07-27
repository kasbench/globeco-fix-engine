package kafka

import (
	"context"
	"time"

	"github.com/kasbench/globeco-fix-engine/internal/config"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// CreateFillsTopicIfNotExists creates the fills topic with 20 partitions if it does not exist.
func CreateFillsTopicIfNotExists(ctx context.Context, cfg config.KafkaConfig, logger *zap.Logger) error {
	logger.Info("Connecting to Kafka broker for topic management", zap.String("broker", cfg.Brokers[0]))
	conn, err := kafka.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		logger.Error("Failed to connect to Kafka broker", zap.String("broker", cfg.Brokers[0]), zap.Error(err))
		return err
	}
	defer conn.Close()
	logger.Info("Successfully connected to Kafka broker", zap.String("broker", cfg.Brokers[0]))

	partitions, err := conn.ReadPartitions()
	if err != nil {
		logger.Error("Failed to read Kafka partitions", zap.Error(err))
		return err
	}
	topicExists := false
	for _, p := range partitions {
		if p.Topic == cfg.FillsTopic {
			topicExists = true
			break
		}
	}
	if topicExists {
		logger.Info("Fills topic already exists", zap.String("topic", cfg.FillsTopic))
		return nil // already exists
	}

	logger.Info("Creating fills topic", zap.String("topic", cfg.FillsTopic), zap.Int("partitions", 20))
	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             cfg.FillsTopic,
		NumPartitions:     20,
		ReplicationFactor: 1, // adjust for production
	})
	if err != nil {
		logger.Error("Failed to create fills topic", zap.String("topic", cfg.FillsTopic), zap.Error(err))
	} else {
		logger.Info("Successfully created fills topic", zap.String("topic", cfg.FillsTopic))
	}
	return err
}

// NewOrdersConsumer creates a Kafka reader for the orders topic.
func NewOrdersConsumer(cfg config.KafkaConfig, groupID string, logger *zap.Logger) *kafka.Reader {
	logger.Info("Creating Kafka orders consumer",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.OrdersTopic),
		zap.String("groupID", groupID),
	)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     cfg.Brokers,
		GroupID:     groupID,
		Topic:       cfg.OrdersTopic,
		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     500 * time.Millisecond,
		StartOffset: kafka.FirstOffset, // consume from beginning if no committed offset exists
	})
	logger.Info("Kafka orders consumer created successfully",
		zap.String("topic", cfg.OrdersTopic),
		zap.String("groupID", groupID),
	)
	return reader
}

// NewFillsProducer creates a Kafka writer for the fills topic.
func NewFillsProducer(cfg config.KafkaConfig, logger *zap.Logger) *kafka.Writer {
	logger.Info("Creating Kafka fills producer",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.FillsTopic),
	)
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.FillsTopic,
		Balancer: &kafka.Hash{},
	})
	logger.Info("Kafka fills producer created successfully", zap.String("topic", cfg.FillsTopic))
	return writer
}
