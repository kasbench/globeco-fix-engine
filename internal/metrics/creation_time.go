package metrics

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// ResolveMessageCreationTime extracts the message creation timestamp using
// priority: (a) created_at Kafka header, (b) createdAt/created_at payload field,
// (c) Kafka record timestamp.
// Returns the resolved time and whether a valid time was found.
func ResolveMessageCreationTime(msg kafka.Message, payloadJSON []byte) (time.Time, bool) {
	// Priority (a): created_at Kafka header
	if t, ok := resolveFromHeader(msg.Headers); ok {
		return t, true
	}

	// Priority (b): createdAt or created_at payload field
	if t, ok := resolveFromPayload(payloadJSON); ok {
		return t, true
	}

	// Priority (c): Kafka record timestamp
	if !msg.Time.IsZero() && msg.Time.Unix() > 0 {
		return msg.Time, true
	}

	return time.Time{}, false
}

// CalculateLatency computes end-to-end latency and applies clamping rules.
// Returns the latency in seconds and whether it should be recorded.
// - latency >= 0: return (latency, true)
// - -1s < latency < 0: return (0, true) — clamp to 0
// - latency <= -1s: return (0, false) — reject
func CalculateLatency(creationTime, completionTime time.Time) (float64, bool) {
	latency := completionTime.Sub(creationTime).Seconds()
	if latency >= 0 {
		return latency, true
	}
	if latency > -1.0 {
		return 0, true
	}
	return 0, false
}

// resolveFromHeader looks for a "created_at" header and attempts to parse it.
// Parse order: integer string (Unix epoch millis), decimal string (Unix epoch seconds), RFC 3339.
func resolveFromHeader(headers []kafka.Header) (time.Time, bool) {
	for _, h := range headers {
		if h.Key != "created_at" {
			continue
		}
		val := strings.TrimSpace(string(h.Value))
		if val == "" {
			return time.Time{}, false
		}
		return parseHeaderValue(val)
	}
	return time.Time{}, false
}

// parseHeaderValue attempts to parse a header value string in priority order:
// 1. Integer string → Unix epoch milliseconds
// 2. Decimal string → Unix epoch seconds
// 3. RFC 3339 string
func parseHeaderValue(val string) (time.Time, bool) {
	// Try integer (Unix epoch milliseconds)
	if millis, err := strconv.ParseInt(val, 10, 64); err == nil {
		t := time.UnixMilli(millis)
		if t.Unix() > 0 {
			return t, true
		}
		return time.Time{}, false
	}

	// Try decimal (Unix epoch seconds)
	if secs, err := strconv.ParseFloat(val, 64); err == nil {
		sec := int64(secs)
		nsec := int64((secs - float64(sec)) * 1e9)
		t := time.Unix(sec, nsec)
		if t.Unix() > 0 {
			return t, true
		}
		return time.Time{}, false
	}

	// Try RFC 3339
	if t, err := time.Parse(time.RFC3339, val); err == nil {
		if t.Unix() > 0 {
			return t, true
		}
		return time.Time{}, false
	}

	return time.Time{}, false
}

// resolveFromPayload looks for "createdAt" or "created_at" fields in a JSON payload.
// Numeric values are interpreted as Unix epoch milliseconds.
// String values are parsed as RFC 3339.
func resolveFromPayload(payloadJSON []byte) (time.Time, bool) {
	if len(payloadJSON) == 0 {
		return time.Time{}, false
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payloadJSON, &raw); err != nil {
		return time.Time{}, false
	}

	// Check "createdAt" first, then "created_at"
	for _, key := range []string{"createdAt", "created_at"} {
		fieldVal, exists := raw[key]
		if !exists {
			continue
		}
		if t, ok := parsePayloadField(fieldVal); ok {
			return t, true
		}
	}

	return time.Time{}, false
}

// parsePayloadField parses a JSON field value as either a numeric Unix epoch millis
// or a string RFC 3339 timestamp.
func parsePayloadField(fieldVal json.RawMessage) (time.Time, bool) {
	// Try numeric (Unix epoch milliseconds)
	var numVal float64
	if err := json.Unmarshal(fieldVal, &numVal); err == nil {
		millis := int64(numVal)
		t := time.UnixMilli(millis)
		if t.Unix() > 0 {
			return t, true
		}
		return time.Time{}, false
	}

	// Try string (RFC 3339)
	var strVal string
	if err := json.Unmarshal(fieldVal, &strVal); err == nil {
		strVal = strings.TrimSpace(strVal)
		if strVal == "" {
			return time.Time{}, false
		}
		if t, err := time.Parse(time.RFC3339, strVal); err == nil {
			if t.Unix() > 0 {
				return t, true
			}
		}
		return time.Time{}, false
	}

	return time.Time{}, false
}
