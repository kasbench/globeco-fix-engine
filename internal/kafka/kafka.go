package kafka

import (
	"context"
	"time"

	"github.com/kasbench/globeco-fix-engine/internal/config"
	"github.com/segmentio/kafka-go"
)

// CreateFillsTopicIfNotExists creates the fills topic with 20 partitions if it does not exist.
func CreateFillsTopicIfNotExists(ctx context.Context, cfg config.KafkaConfig) error {
	conn, err := kafka.DialContext(ctx, "tcp", cfg.Brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
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
		return nil // already exists
	}

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             cfg.FillsTopic,
		NumPartitions:     1,
		ReplicationFactor: 1, // adjust for production
	})
	return err
}

// NewOrdersConsumer creates a Kafka reader for the orders topic.
func NewOrdersConsumer(cfg config.KafkaConfig, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Brokers,
		GroupID:  groupID,
		Topic:    cfg.OrdersTopic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
		MaxWait:  500 * time.Millisecond,
	})
}

// NewFillsProducer creates a Kafka writer for the fills topic.
func NewFillsProducer(cfg config.KafkaConfig) *kafka.Writer {
	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  cfg.Brokers,
		Topic:    cfg.FillsTopic,
		Balancer: &kafka.Hash{},
	})
}
