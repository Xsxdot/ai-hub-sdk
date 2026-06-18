package aihubsdk

import (
	"context"
	"net/http"
	"testing"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

func TestEmbedding(t *testing.T) {
	var m, p string
	srv := newJSONServer(t, `{"status":200,"data":{"model":"m","data":[{"index":0,"embedding":[0.1]}],"usage":{"metrics":{"token_input":3}}}}`, &m, &p)
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	res, err := c.Embedding(context.Background(), &dto.EmbeddingRequest{Model: "m", Input: []string{"a"}})
	if err != nil {
		t.Fatalf("embedding err: %v", err)
	}
	if res.Model != "m" || len(res.Data) != 1 || res.Data[0].Embedding[0] != 0.1 || p != "/v1/embeddings" || m != http.MethodPost {
		t.Fatalf("res=%+v path=%s method=%s", res, p, m)
	}
}

func TestRerank(t *testing.T) {
	var m, p string
	srv := newJSONServer(t, `{"status":200,"data":{"model":"m","results":[{"index":1,"relevance_score":0.9}],"usage":{"metrics":{"token_input":5}}}}`, &m, &p)
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	res, err := c.Rerank(context.Background(), &dto.RerankRequest{Model: "m", Query: "q", Documents: []string{"a", "b"}})
	if err != nil {
		t.Fatalf("rerank err: %v", err)
	}
	if res.Model != "m" || len(res.Results) != 1 || res.Results[0].Index != 1 || p != "/v1/rerank" || m != http.MethodPost {
		t.Fatalf("res=%+v path=%s method=%s", res, p, m)
	}
}
