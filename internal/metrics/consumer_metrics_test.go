package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestNewConsumerMetrics_Success(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	defer provider.Shutdown(nil)
	meter := provider.Meter("test")

	cm, err := NewConsumerMetrics(meter, "fix_engine")

	require.NoError(t, err)
	require.NotNil(t, cm)
	assert.NotNil(t, cm.messagesProcessed)
	assert.NotNil(t, cm.messagesFailed)
	assert.NotNil(t, cm.processingSeconds)
	assert.NotNil(t, cm.idleSeconds)
	assert.NotNil(t, cm.recordsPolled)
	assert.NotNil(t, cm.pollSeconds)
	assert.NotNil(t, cm.processingDuration)
	assert.NotNil(t, cm.messageLatency)
}

func TestNewConsumerMetrics_CommonAttrs(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	defer provider.Shutdown(nil)
	meter := provider.Meter("test")

	cm, err := NewConsumerMetrics(meter, "my-group")

	require.NoError(t, err)
	require.NotNil(t, cm)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("service", "globeco-fix-engine"),
		attribute.String("consumer_group", "my-group"),
	}, cm.commonAttrs)
}

func TestNewConsumerMetrics_EmptyConsumerGroup_FallsBackToUnknown(t *testing.T) {
	provider := sdkmetric.NewMeterProvider()
	defer provider.Shutdown(nil)
	meter := provider.Meter("test")

	cm, err := NewConsumerMetrics(meter, "")

	require.NoError(t, err)
	require.NotNil(t, cm)
	assert.Equal(t, []attribute.KeyValue{
		attribute.String("service", "globeco-fix-engine"),
		attribute.String("consumer_group", "unknown"),
	}, cm.commonAttrs)
}
