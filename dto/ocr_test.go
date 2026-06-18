// Package dto 的 OCR 契约测试：验证 OCR 中立 DTO 的 JSON 兼容性。
//
// 职责：
//   - 覆盖 OcrRequest/OcrResult 的 JSON 往返序列化
//
// 边界：
//   - 不测试 server 端厂商协议、OSS 解析或计费逻辑
package dto

import (
	"encoding/json"
	"testing"
)

func TestOcrRequestJSON(t *testing.T) {
	req := OcrRequest{
		Model:        "qwen3.5-ocr",
		ImageURL:     "oss://ocr/in/x.jpg",
		Task:         OcrTaskKeyInformation,
		ResultSchema: map[string]any{"发票号码": "提取发票号码"},
		Options:      map[string]any{"enable_rotate": true},
	}
	b, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round OcrRequest
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.Task != OcrTaskKeyInformation || round.ImageURL != "oss://ocr/in/x.jpg" {
		t.Fatalf("round trip mismatch: %+v", round)
	}
}

func TestOcrResultJSON(t *testing.T) {
	res := OcrResult{
		Model:      "qwen3.5-ocr",
		Text:       "hello",
		Structured: map[string]any{"kv_result": map[string]any{"发票号码": "10283819"}},
		Usage:      Usage{Metrics: map[BillingMetric]float64{MetricTokenInput: 606, MetricTokenOutput: 159}},
	}
	b, err := json.Marshal(&res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var round OcrResult
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.Text != "hello" || round.Structured == nil {
		t.Fatalf("round trip mismatch: %+v", round)
	}
}
