package kafka

import (
	"context"
	"testing"
	"time"

	segmentio_kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	tc_kafka "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func TestKafkaProduceConsume(t *testing.T) {
	ctx := context.Background()
	container, err := tc_kafka.RunContainer(ctx)
	if err != nil {
		t.Fatalf("failed to start kafka container: %v", err)
	}
	defer container.Terminate(ctx)

	brokers, err := container.Brokers(ctx)
	if err != nil {
		t.Fatalf("failed to get brokers: %v", err)
	}

	topic := "test-topic"
	// Create topic
	conn, err := segmentio_kafka.Dial("tcp", brokers[0])
	if err != nil {
		t.Fatalf("failed to dial kafka: %v", err)
	}
	defer conn.Close()
	if err := conn.CreateTopics(segmentio_kafka.TopicConfig{Topic: topic, NumPartitions: 1, ReplicationFactor: 1}); err != nil {
		t.Fatalf("failed to create topic: %v", err)
	}

	// Produce a message
	writer := segmentio_kafka.NewWriter(segmentio_kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &segmentio_kafka.LeastBytes{},
	})
	defer writer.Close()
	msg := segmentio_kafka.Message{Value: []byte("hello world")}
	err = writer.WriteMessages(ctx, msg)
	assert.NoError(t, err)

	// Consume the message
	reader := segmentio_kafka.NewReader(segmentio_kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  "test-group",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	m, err := reader.ReadMessage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello world"), m.Value)
}
