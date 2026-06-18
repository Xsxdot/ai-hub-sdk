package dto

import (
	"encoding/json"
	"testing"
)

func TestEmbeddingRequestJSON(t *testing.T) {
	req := EmbeddingRequest{Model: "text-embedding-v4", Input: []string{"a", "b"}, Dimensions: 1024}
	b, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(b)
	if got != `{"model":"text-embedding-v4","input":["a","b"],"dimensions":1024}` {
		t.Fatalf("unexpected json: %s", got)
	}
}

func TestRerankResponseJSON(t *testing.T) {
	resp := RerankResponse{
		Model:   "qwen3-rerank",
		Results: []RerankItem{{Index: 2, RelevanceScore: 0.9}},
		Usage:   Usage{Metrics: map[BillingMetric]float64{MetricTokenInput: 12}},
	}
	b, err := json.Marshal(&resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round RerankResponse
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.Results[0].Index != 2 || round.Results[0].RelevanceScore != 0.9 {
		t.Fatalf("round trip mismatch: %+v", round)
	}
}
