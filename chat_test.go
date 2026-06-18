package aihubsdk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

func TestChat_NonStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req dto.ChatRequest
		_ = json.Unmarshal(raw, &req)
		if req.Model != "gpt" {
			t.Errorf("model not forwarded: %q", req.Model)
		}
		if req.Stream {
			t.Errorf("Chat must force stream=false")
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"status":200,"data":{"id":"resp1","model":"gpt","stopReason":"end_turn"}}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	resp, err := c.Chat(context.Background(), &dto.ChatRequest{Model: "gpt", Stream: true})
	if err != nil {
		t.Fatalf("Chat err: %v", err)
	}
	if resp.ID != "resp1" || resp.StopReason != "end_turn" {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestChatStream_ForcesStreamTrueAndYields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req dto.ChatRequest
		_ = json.Unmarshal(raw, &req)
		if !req.Stream {
			t.Errorf("ChatStream must force stream=true")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fl := w.(http.Flusher)
		w.Write([]byte("data: {\"type\":\"message_start\"}\n\n"))
		fl.Flush()
		w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		fl.Flush()
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	ch, err := c.ChatStream(context.Background(), &dto.ChatRequest{Model: "gpt"})
	if err != nil {
		t.Fatalf("ChatStream err: %v", err)
	}
	var n int
	for range ch {
		n++
	}
	if n != 2 {
		t.Fatalf("events = %d, want 2", n)
	}
}
