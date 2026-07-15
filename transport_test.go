package aihubsdk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type echoData struct {
	Name string `json:"name"`
}

func TestDoJSON_UnwrapsResultShell(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "k" {
			w.WriteHeader(401)
			w.Write([]byte(`{"status":401,"message":"无效的 API Key"}`))
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"status":200,"data":{"name":"hello"}}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	var out echoData
	if err := c.doJSON(context.Background(), http.MethodPost, "/v1/echo", map[string]string{"x": "y"}, &out); err != nil {
		t.Fatalf("doJSON err: %v", err)
	}
	if out.Name != "hello" {
		t.Fatalf("data not unwrapped: %+v", out)
	}
}

func TestNewRequest_PropagatesTraceHeadersFromContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), "traceId", "sdk-trace-1")
	c := New(WithBaseURL("https://aihub.example"), WithAPIKey("k"))

	req, err := c.newRequest(ctx, http.MethodPost, "/v1/echo", map[string]string{"x": "y"})
	if err != nil {
		t.Fatalf("newRequest err: %v", err)
	}
	if got := req.Header.Get("Trace-Head"); got != "sdk-trace-1" {
		t.Fatalf("Trace-Head = %q, want sdk trace", got)
	}
	if got := req.Header.Get("X-Trace-Id"); got != "sdk-trace-1" {
		t.Fatalf("X-Trace-Id = %q, want sdk trace", got)
	}
}

func TestDoJSON_401ReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"status":401,"message":"无效的 API Key"}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("bad"))
	var out echoData
	err := c.doJSON(context.Background(), http.MethodPost, "/v1/echo", nil, &out)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %T (%v)", err, err)
	}
	if apiErr.Status != 401 {
		t.Fatalf("status = %d", apiErr.Status)
	}
}

func TestDoSSE_PropagatesTraceHeadersFromContext(t *testing.T) {
	var gotTraceHead string
	var gotXTraceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceHead = r.Header.Get("Trace-Head")
		gotXTraceID = r.Header.Get("X-Trace-Id")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fl := w.(http.Flusher)
		w.Write([]byte("data: {\"type\":\"message_stop\"}\n\n"))
		fl.Flush()
	}))
	defer srv.Close()

	ctx := context.WithValue(context.Background(), "traceId", "sdk-trace-sse-1")
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	ch, err := c.doSSE(ctx, "/v1/echo-stream", map[string]bool{"stream": true})
	if err != nil {
		t.Fatalf("doSSE err: %v", err)
	}
	for range ch {
	}
	if gotTraceHead != "sdk-trace-sse-1" {
		t.Fatalf("Trace-Head = %q, want sdk trace", gotTraceHead)
	}
	if gotXTraceID != "sdk-trace-sse-1" {
		t.Fatalf("X-Trace-Id = %q, want sdk trace", gotXTraceID)
	}
}

func TestDoSSE_SplitsFramesAndDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	ch, err := c.doSSE(context.Background(), "/v1/echo-stream", map[string]bool{"stream": true})
	if err != nil {
		t.Fatalf("doSSE err: %v", err)
	}
	var types []string
	for ev := range ch {
		types = append(types, string(ev.Type))
	}
	if len(types) != 2 || types[0] != "message_start" || types[1] != "message_stop" {
		t.Fatalf("frames = %v", types)
	}
}

func TestDoSSE_CtxCancelClosesChannel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fl := w.(http.Flusher)
		for i := 0; i < 100; i++ {
			w.Write([]byte("data: {\"type\":\"content_block_delta\"}\n\n"))
			fl.Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	ch, err := c.doSSE(ctx, "/v1/echo-stream", map[string]bool{"stream": true})
	if err != nil {
		t.Fatalf("doSSE err: %v", err)
	}
	<-ch
	cancel()
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after ctx cancel")
	}
}
