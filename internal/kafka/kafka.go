package kafka

import (
	"context"
	"fmt"
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
// Uses extended session and rebalance timeouts to tolerate slow broker stabilization at startup.
func NewOrdersConsumer(cfg config.KafkaConfig, groupID string, logger *zap.Logger) *kafka.Reader {
	logger.Info("Creating Kafka orders consumer",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.OrdersTopic),
		zap.String("groupID", groupID),
		zap.Duration("sessionTimeout", 60*time.Second),
		zap.Duration("rebalanceTimeout", 90*time.Second),
		zap.Duration("heartbeatInterval", 10*time.Second),
	)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:           cfg.Brokers,
		GroupID:           groupID,
		Topic:             cfg.OrdersTopic,
		MinBytes:          1,    // Was 10e3 (10KB)
		MaxBytes:          10e6, // 10MB
		MaxWait:           500 * time.Millisecond,
		StartOffset:       kafka.FirstOffset, // consume from beginning if no committed offset exists
		SessionTimeout:    60 * time.Second,  // extended from default 30s to tolerate slow startup
		RebalanceTimeout:  90 * time.Second,  // give broker extra time to complete rebalances
		HeartbeatInterval: 10 * time.Second,  // more frequent heartbeats to maintain session
	})
	logger.Info("Kafka orders consumer created successfully",
		zap.String("topic", cfg.OrdersTopic),
		zap.String("groupID", groupID),
	)
	return reader
}

// WaitForPartitionAssignment polls reader stats to confirm partition assignment.
// Uses a short per-attempt timeout with retries to handle the race condition where
// this is called before Kafka is fully operational.
// Returns nil once partitions are confirmed, or an error if the context is cancelled.
func WaitForPartitionAssignment(ctx context.Context, reader *kafka.Reader, logger *zap.Logger) error {
	const (
		attemptTimeout = 15 * time.Second // short timeout per attempt
		maxAttempts    = 10               // total attempts before giving up
		pollInterval   = 2 * time.Second  // how often to check stats within an attempt
	)

	logger.Info("Waiting for Kafka partition assignment",
		zap.String("topic", reader.Config().Topic),
		zap.String("groupID", reader.Config().GroupID),
		zap.Duration("attemptTimeout", attemptTimeout),
		zap.Int("maxAttempts", maxAttempts),
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		logger.Info("Partition assignment attempt",
			zap.Int("attempt", attempt),
			zap.Int("maxAttempts", maxAttempts),
		)

		assigned, err := waitForAssignmentAttempt(ctx, reader, attemptTimeout, pollInterval)
		if err != nil {
			return err // context cancelled
		}
		if assigned {
			stats := reader.Stats()
			logger.Info("Kafka partition assignment confirmed",
				zap.String("topic", stats.Topic),
				zap.Int64("dials", stats.Dials),
				zap.Int64("rebalances", stats.Rebalances),
				zap.Int("attempt", attempt),
			)
			return nil
		}

		logger.Warn("Partition assignment not confirmed, retrying",
			zap.Int("attempt", attempt),
			zap.Int("maxAttempts", maxAttempts),
			zap.Duration("nextRetryIn", pollInterval),
		)

		// Brief pause before retrying to let Kafka stabilize
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	// Exhausted all retries — fail so the pod crashes and gets restarted by K8s
	stats := reader.Stats()
	logger.Error("Exhausted all partition assignment attempts, failing startup",
		zap.String("topic", stats.Topic),
		zap.String("partition", stats.Partition),
		zap.Int("attempts", maxAttempts),
	)
	return fmt.Errorf("failed to confirm Kafka partition assignment after %d attempts for topic %s", maxAttempts, stats.Topic)
}

// waitForAssignmentAttempt polls reader stats for a single attempt window.
// Returns (true, nil) if assignment is confirmed, (false, nil) on timeout,
// or (false, err) if the context is cancelled.
func waitForAssignmentAttempt(ctx context.Context, reader *kafka.Reader, timeout, pollInterval time.Duration) (bool, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-deadline:
			return false, nil
		case <-ticker.C:
			stats := reader.Stats()
			if stats.Dials > 0 && stats.Rebalances > 0 {
				return true, nil
			}
		}
	}
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
