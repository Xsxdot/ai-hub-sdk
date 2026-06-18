package aihubsdk

import (
	"strings"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	c := New(WithBaseURL("http://x"), WithAPIKey("k"))
	if c.baseURL != "http://x" {
		t.Fatalf("baseURL = %q", c.baseURL)
	}
	if c.apiKey != "k" {
		t.Fatalf("apiKey = %q", c.apiKey)
	}
	if c.httpClient == nil {
		t.Fatal("httpClient must default to non-nil")
	}
}

func TestNew_TrimsTrailingSlash(t *testing.T) {
	c := New(WithBaseURL("http://x/"))
	if strings.HasSuffix(c.baseURL, "/") {
		t.Fatalf("baseURL should be trimmed, got %q", c.baseURL)
	}
}

func TestWithHTTPClientAndTimeout(t *testing.T) {
	c := New(WithTimeout(3 * time.Second))
	if c.httpClient.Timeout != 3*time.Second {
		t.Fatalf("timeout = %v", c.httpClient.Timeout)
	}
}
