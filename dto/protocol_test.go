package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatRequest_JSONRoundTrip(t *testing.T) {
	req := ChatRequest{
		Model: "smart-multimodal",
		Messages: []Message{
			{
				Role: RoleUser,
				Content: []ContentBlock{
					{Type: BlockText, Text: "你好"},
					{Type: BlockImage, Source: "https://x/y.png", MediaType: "image/png"},
				},
			},
		},
		System:    "你是助手",
		MaxTokens: 1024,
		Thinking:  &ThinkingConfig{Enabled: true, Level: ThinkingHigh},
		Stream:    true,
		Metadata:  map[string]string{"biz_id": "b1", "user_id": "u1"},
	}

	raw, err := json.Marshal(req)
	require.NoError(t, err)

	var got ChatRequest
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "smart-multimodal", got.Model)
	require.Len(t, got.Messages, 1)
	require.Len(t, got.Messages[0].Content, 2)
	assert.Equal(t, BlockText, got.Messages[0].Content[0].Type)
	assert.Equal(t, BlockImage, got.Messages[0].Content[1].Type)
	require.NotNil(t, got.Thinking)
	assert.Equal(t, ThinkingHigh, got.Thinking.Level)
}

func TestThinkingLevel_JSONRoundTripMedium(t *testing.T) {
	raw, err := json.Marshal(&ThinkingConfig{Enabled: true, Level: ThinkingMedium})
	require.NoError(t, err)
	assert.JSONEq(t, `{"enabled":true,"level":"medium"}`, string(raw))

	var got ThinkingConfig
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, ThinkingMedium, got.Level)
}

func TestUsage_Metrics(t *testing.T) {
	u := Usage{Metrics: map[BillingMetric]float64{
		MetricTokenInput:  1200,
		MetricTokenOutput: 800,
	}}
	assert.Equal(t, float64(1200), u.Metrics[MetricTokenInput])
	assert.Equal(t, float64(800), u.Metrics[MetricTokenOutput])
}

func TestChatResponse_JSONRoundTrip(t *testing.T) {
	resp := ChatResponse{
		ID:                 "resp-1",
		Model:              "smart-multimodal",
		ActualChannelModel: "cm-10",
		Content:            []ContentBlock{{Type: BlockText, Text: "答案"}},
		StopReason:         "end_turn",
		Usage:              Usage{Metrics: map[BillingMetric]float64{MetricTokenOutput: 5}},
		Cost: Cost{
			Details:  []CostDetail{{Metric: MetricTokenOutput, Quantity: 5, Amount: 0.01}},
			Total:    0.01,
			Currency: "USD",
		},
	}
	raw, err := json.Marshal(resp)
	require.NoError(t, err)
	var got ChatResponse
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "cm-10", got.ActualChannelModel)
	assert.Equal(t, "USD", got.Cost.Currency)
	require.Len(t, got.Content, 1)
}
