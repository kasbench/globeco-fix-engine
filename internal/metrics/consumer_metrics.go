package metrics

import (
	"context"
	"fmt"
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ConsumerMetrics holds all Kafka consumer metric instruments.
type ConsumerMetrics struct {
	messagesProcessed  metric.Float64Counter
	messagesFailed     metric.Float64Counter
	processingSeconds  metric.Float64Counter
	idleSeconds        metric.Float64Counter
	recordsPolled      metric.Float64Counter
	pollSeconds        metric.Float64Counter
	processingDuration metric.Float64Histogram
	messageLatency     metric.Float64Histogram

	// Pre-computed common attributes (service + consumer_group).
	// Topic and partition are added per-observation.
	commonAttrs []attribute.KeyValue
}

// NewConsumerMetrics creates and registers all metric instruments.
// Returns an error if any instrument cannot be created.
func NewConsumerMetrics(meter metric.Meter, consumerGroup string) (*ConsumerMetrics, error) {
	if consumerGroup == "" {
		consumerGroup = "unknown"
	}

	commonAttrs := []attribute.KeyValue{
		attribute.String("service", "globeco-fix-engine"),
		attribute.String("consumer_group", consumerGroup),
	}

	messagesProcessed, err := meter.Float64Counter(
		"kafka_consumer_messages_processed_total",
		metric.WithUnit("{message}"),
		metric.WithDescription("Total number of Kafka messages successfully processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating messages_processed counter: %w", err)
	}

	messagesFailed, err := meter.Float64Counter(
		"kafka_consumer_messages_failed_total",
		metric.WithUnit("{message}"),
		metric.WithDescription("Total number of Kafka messages that permanently failed processing"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating messages_failed counter: %w", err)
	}

	processingSeconds, err := meter.Float64Counter(
		"kafka_consumer_processing_seconds_total",
		metric.WithUnit("s"),
		metric.WithDescription("Total time spent actively processing Kafka messages"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating processing_seconds counter: %w", err)
	}

	idleSeconds, err := meter.Float64Counter(
		"kafka_consumer_idle_seconds_total",
		metric.WithUnit("s"),
		metric.WithDescription("Total time the consumer was idle (blocked in poll or between messages)"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating idle_seconds counter: %w", err)
	}

	recordsPolled, err := meter.Float64Counter(
		"kafka_consumer_records_polled_total",
		metric.WithUnit("{record}"),
		metric.WithDescription("Total number of records returned from poll operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating records_polled counter: %w", err)
	}

	pollSeconds, err := meter.Float64Counter(
		"kafka_consumer_poll_seconds_total",
		metric.WithUnit("s"),
		metric.WithDescription("Total time spent in poll (ReadMessage) calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating poll_seconds counter: %w", err)
	}

	processingDuration, err := meter.Float64Histogram(
		"kafka_consumer_processing_duration_seconds",
		metric.WithUnit("s"),
		metric.WithDescription("Distribution of per-message processing durations"),
		metric.WithExplicitBucketBoundaries(
			0.005, 0.010, 0.025, 0.050, 0.100, 0.250, 0.500,
			1, 2.5, 5, 10, 30, 60,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating processing_duration histogram: %w", err)
	}

	messageLatency, err := meter.Float64Histogram(
		"kafka_consumer_message_latency_seconds",
		metric.WithUnit("s"),
		metric.WithDescription("Distribution of end-to-end message latency from creation to completion"),
		metric.WithExplicitBucketBoundaries(
			0.010, 0.025, 0.050, 0.100, 0.250, 0.500,
			1, 2.5, 5, 10, 30, 60, 120, 300, 600,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating message_latency histogram: %w", err)
	}

	return &ConsumerMetrics{
		messagesProcessed:  messagesProcessed,
		messagesFailed:     messagesFailed,
		processingSeconds:  processingSeconds,
		idleSeconds:        idleSeconds,
		recordsPolled:      recordsPolled,
		pollSeconds:        pollSeconds,
		processingDuration: processingDuration,
		messageLatency:     messageLatency,
		commonAttrs:        commonAttrs,
	}, nil
}

// partitionLabel returns the string representation of a partition number,
// or "unknown" if the partition is invalid (negative).
func partitionLabel(partition int) string {
	if partition < 0 {
		return "unknown"
	}
	return strconv.Itoa(partition)
}

// clampDuration returns 0 if d is non-positive, otherwise returns d unchanged.
func clampDuration(d float64) float64 {
	if d <= 0 {
		return 0
	}
	return d
}

// RecordPollSuccess records metrics after a successful ReadMessage.
// It increments records_polled_total by 1, and adds pollDuration to
// poll_seconds_total and idle_seconds_total, with topic/partition labels.
func (m *ConsumerMetrics) RecordPollSuccess(ctx context.Context, pollDuration float64, topic string, partition int) {
	defer func() { recover() }()

	pollDuration = clampDuration(pollDuration)

	attrs := make([]attribute.KeyValue, 0, len(m.commonAttrs)+2)
	attrs = append(attrs, m.commonAttrs...)
	attrs = append(attrs,
		attribute.String("topic", topic),
		attribute.String("partition", partitionLabel(partition)),
	)
	opt := metric.WithAttributes(attrs...)

	m.recordsPolled.Add(ctx, 1, opt)
	m.pollSeconds.Add(ctx, pollDuration, opt)
	m.idleSeconds.Add(ctx, pollDuration, opt)
}

// RecordPollError records metrics after a failed ReadMessage (non-cancellation).
// It adds pollDuration to poll_seconds_total and idle_seconds_total without
// incrementing records polled. Topic and partition are not included since no
// message was received.
func (m *ConsumerMetrics) RecordPollError(ctx context.Context, pollDuration float64) {
	defer func() { recover() }()

	pollDuration = clampDuration(pollDuration)

	opt := metric.WithAttributes(m.commonAttrs...)

	m.pollSeconds.Add(ctx, pollDuration, opt)
	m.idleSeconds.Add(ctx, pollDuration, opt)
}

// RecordProcessingSuccess records metrics after successful message processing.
// It increments messages_processed_total, adds processingDuration to
// processing_seconds_total, and records histogram observations with result=success.
// If latency is non-nil, it records a message_latency observation.
func (m *ConsumerMetrics) RecordProcessingSuccess(ctx context.Context, processingDuration float64, latency *float64, topic string, partition int) {
	defer func() { recover() }()

	processingDuration = clampDuration(processingDuration)

	attrs := make([]attribute.KeyValue, 0, len(m.commonAttrs)+2)
	attrs = append(attrs, m.commonAttrs...)
	attrs = append(attrs,
		attribute.String("topic", topic),
		attribute.String("partition", partitionLabel(partition)),
	)
	opt := metric.WithAttributes(attrs...)

	m.messagesProcessed.Add(ctx, 1, opt)
	m.processingSeconds.Add(ctx, processingDuration, opt)

	// Histogram observations include the result label.
	histAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	histAttrs = append(histAttrs, attrs...)
	histAttrs = append(histAttrs, attribute.String("result", "success"))
	histOpt := metric.WithAttributes(histAttrs...)

	m.processingDuration.Record(ctx, processingDuration, histOpt)

	if latency != nil {
		m.messageLatency.Record(ctx, *latency, histOpt)
	}
}

// RecordProcessingFailure records metrics after failed message processing.
// It increments messages_failed_total, adds processingDuration to
// processing_seconds_total, and records histogram observations with result=failure.
// If latency is non-nil, it records a message_latency observation.
func (m *ConsumerMetrics) RecordProcessingFailure(ctx context.Context, processingDuration float64, latency *float64, topic string, partition int) {
	defer func() { recover() }()

	processingDuration = clampDuration(processingDuration)

	attrs := make([]attribute.KeyValue, 0, len(m.commonAttrs)+2)
	attrs = append(attrs, m.commonAttrs...)
	attrs = append(attrs,
		attribute.String("topic", topic),
		attribute.String("partition", partitionLabel(partition)),
	)
	opt := metric.WithAttributes(attrs...)

	m.messagesFailed.Add(ctx, 1, opt)
	m.processingSeconds.Add(ctx, processingDuration, opt)

	// Histogram observations include the result label.
	histAttrs := make([]attribute.KeyValue, 0, len(attrs)+1)
	histAttrs = append(histAttrs, attrs...)
	histAttrs = append(histAttrs, attribute.String("result", "failure"))
	histOpt := metric.WithAttributes(histAttrs...)

	m.processingDuration.Record(ctx, processingDuration, histOpt)

	if latency != nil {
		m.messageLatency.Record(ctx, *latency, histOpt)
	}
}
