package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamEvent_JSONRoundTrip(t *testing.T) {
	events := []StreamEvent{
		{Type: EventMessageStart, MessageStart: &MessageStartData{ID: "m1", ActualChannelModel: "cm-10"}},
		{Type: EventContentBlockStart, ContentBlockStart: &ContentBlockStartData{Index: 0, BlockType: BlockThinking}},
		{Type: EventContentBlockDelta, ContentBlockDelta: &ContentBlockDeltaData{Index: 0, Delta: "让我想想"}},
		{Type: EventContentBlockStop, ContentBlockStop: &ContentBlockStopData{Index: 0}},
		{Type: EventUsage, Usage: &Usage{Metrics: map[BillingMetric]float64{MetricTokenOutput: 42}}, Cost: &Cost{Total: 0.01, Currency: "USD"}},
		{Type: EventMessageStop, MessageStop: &MessageStopData{StopReason: "end_turn"}},
		{Type: EventError, Error: &StreamErrorData{Code: "overloaded_error", Message: "上游过载"}},
	}

	for _, e := range events {
		raw, err := json.Marshal(e)
		require.NoError(t, err)
		var got StreamEvent
		require.NoError(t, json.Unmarshal(raw, &got))
		assert.Equal(t, e.Type, got.Type)
	}
}

func TestStreamEvent_ThinkingDeltaCarriesText(t *testing.T) {
	e := StreamEvent{
		Type:              EventContentBlockDelta,
		ContentBlockDelta: &ContentBlockDeltaData{Index: 1, Delta: "thinking text"},
	}
	require.NotNil(t, e.ContentBlockDelta)
	assert.Equal(t, "thinking text", e.ContentBlockDelta.Delta)
}
