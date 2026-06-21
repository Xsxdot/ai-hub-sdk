package aihubsdk

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

// newJSONServer 返回固定 result 壳响应的 mock，并记录最后命中的方法+路径。
func newJSONServer(t *testing.T, body string, gotMethod, gotPath *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*gotMethod = r.Method
		*gotPath = r.URL.Path
		w.WriteHeader(200)
		w.Write([]byte(body))
	}))
}

func TestGenerateImage(t *testing.T) {
	var m, p string
	srv := newJSONServer(t, `{"status":200,"data":{"id":"img1","artifacts":[{"ossKey":"ai-hub/public-media/image/x.png","mediaType":"image/png"}]}}`, &m, &p)
	defer srv.Close()
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	res, err := c.GenerateImage(context.Background(), &dto.ImageRequest{Model: "sd", Prompt: "cat"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.ID != "img1" || len(res.Artifacts) != 1 || res.Artifacts[0].OSSKey != "ai-hub/public-media/image/x.png" || p != "/v1/images/generate" || m != http.MethodPost {
		t.Fatalf("res=%+v path=%s method=%s", res, p, m)
	}
}

func TestSubmitVideoJobAndGetJob(t *testing.T) {
	var m, p string
	srvSubmit := newJSONServer(t, `{"status":200,"data":"job-123"}`, &m, &p)
	defer srvSubmit.Close()
	c := New(WithBaseURL(srvSubmit.URL), WithAPIKey("k"))
	jobID, err := c.SubmitVideoJob(context.Background(), &dto.VideoJobRequest{Model: "v", Task: dto.VideoTaskText2Video})
	if err != nil {
		t.Fatalf("submit err: %v", err)
	}
	if jobID != "job-123" || p != "/v1/videos/jobs" {
		t.Fatalf("jobID=%q path=%s", jobID, p)
	}

	srvGet := newJSONServer(t, `{"status":200,"data":{"jobId":"job-123","state":"succeeded"}}`, &m, &p)
	defer srvGet.Close()
	c2 := New(WithBaseURL(srvGet.URL), WithAPIKey("k"))
	res, err := c2.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if res.State != dto.JobStateSucceeded || p != "/v1/videos/jobs/job-123" || m != http.MethodGet {
		t.Fatalf("res=%+v path=%s method=%s", res, p, m)
	}
}

func TestDeleteVoice(t *testing.T) {
	var m, p string
	srv := newJSONServer(t, `{"status":200,"data":null}`, &m, &p)
	defer srv.Close()
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	if err := c.DeleteVoice(context.Background(), 42); err != nil {
		t.Fatalf("err: %v", err)
	}
	if p != "/v1/voices/42" || m != http.MethodDelete {
		t.Fatalf("path=%s method=%s", p, m)
	}
}

func TestGenerateSpeechAndTranscribeAndCreateVoice(t *testing.T) {
	var m, p string
	srv := newJSONServer(t, `{"status":200,"data":{"id":"s1","audioOssKey":"ai-hub/public-media/audio/a.wav"}}`, &m, &p)
	defer srv.Close()
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	speech, err := c.GenerateSpeech(context.Background(), &dto.SpeechRequest{Voice: "v", Text: "hi"})
	if err != nil {
		t.Fatalf("speech err: %v", err)
	}
	if p != "/v1/speech/generate" || speech.AudioOssKey != "ai-hub/public-media/audio/a.wav" {
		t.Fatalf("speech=%+v path=%s", speech, p)
	}

	srv2 := newJSONServer(t, `{"status":200,"data":{"id":"t1","text":"hello"}}`, &m, &p)
	defer srv2.Close()
	c2 := New(WithBaseURL(srv2.URL), WithAPIKey("k"))
	audio := dto.URLMediaRef("http://a", "audio/mpeg")
	if _, err := c2.Transcribe(context.Background(), &dto.TranscribeRequest{Model: "asr", Audio: &audio}); err != nil {
		t.Fatalf("transcribe err: %v", err)
	}
	if p != "/v1/audio/transcriptions" {
		t.Fatalf("transcribe path=%s", p)
	}

	srv3 := newJSONServer(t, `{"status":200,"data":{"logicalVoiceId":7}}`, &m, &p)
	defer srv3.Close()
	c3 := New(WithBaseURL(srv3.URL), WithAPIKey("k"))
	res, err := c3.CreateVoice(context.Background(), &dto.CreateVoiceRequest{Name: "n", Source: dto.VoiceSourceClone})
	if err != nil {
		t.Fatalf("createvoice err: %v", err)
	}
	if res.LogicalVoiceID != 7 || p != "/v1/voices" {
		t.Fatalf("res=%+v path=%s", res, p)
	}
}

func TestOcr(t *testing.T) {
	var m, p string
	srv := newJSONServer(t, `{"status":200,"data":{"model":"ocr","text":"hello","structured":{"kv_result":{"id":"42"}}}}`, &m, &p)
	defer srv.Close()
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	image := dto.URLMediaRef("https://x/y.jpg", "image/jpeg")
	res, err := c.Ocr(context.Background(), &dto.OcrRequest{Model: "ocr", Image: &image, Task: dto.OcrTaskTextRecognition})
	if err != nil {
		t.Fatalf("ocr err: %v", err)
	}
	if res.Text != "hello" || res.Structured == nil || p != "/v1/ocr" || m != http.MethodPost {
		t.Fatalf("res=%+v path=%s method=%s", res, p, m)
	}
}

func TestClient_UploadMedia(t *testing.T) {
	var gotAPIKey string
	var gotKind string
	var gotFilename string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get("X-API-Key")
		gotKind = r.FormValue("kind")
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()
		gotFilename = header.Filename
		raw, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if string(raw) != "image-bytes" {
			t.Fatalf("raw=%q", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":200,"data":{"ossKey":"ai-hub/public-media/image/20260621/a.png","mediaType":"image/png","size":11,"kind":"image"}}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("key-1"))
	res, err := c.UploadMedia(context.Background(), UploadMediaKindImage, "a.png", strings.NewReader("image-bytes"))
	if err != nil {
		t.Fatalf("upload media: %v", err)
	}
	if gotAPIKey != "key-1" || gotKind != "image" || gotFilename != "a.png" {
		t.Fatalf("headers/form apiKey=%q kind=%q filename=%q", gotAPIKey, gotKind, gotFilename)
	}
	if res.OSSKey != "ai-hub/public-media/image/20260621/a.png" || res.MediaType != "image/png" || res.Size != 11 {
		t.Fatalf("res=%+v", res)
	}
}
