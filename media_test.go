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
	srv := newJSONServer(t, `{"status":200,"data":{"id":"img1","artifacts":[{"ossKey":"ai-hub/public-media/image/x.png","url":"https://public.example.com/x.png?signature=secret","urlExpiresAt":1784073600000,"mediaType":"image/png"}]}}`, &m, &p)
	defer srv.Close()
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	res, err := c.GenerateImage(context.Background(), &dto.ImageRequest{Model: "sd", Prompt: "cat"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.ID != "img1" || len(res.Artifacts) != 1 ||
		res.Artifacts[0].OSSKey != "ai-hub/public-media/image/x.png" ||
		res.Artifacts[0].URL != "https://public.example.com/x.png?signature=secret" ||
		res.Artifacts[0].URLExpiresAt != 1784073600000 || res.Artifacts[0].MediaType != "image/png" ||
		p != "/v1/images/generate" || m != http.MethodPost {
		t.Fatalf("res=%+v path=%s method=%s", res, p, m)
	}
}

func TestResolveMedia(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAPIKey string
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("X-API-Key")
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read resolve body: %v", err)
		}
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":200,"data":{"ossKey":"ai-hub/public-media/audio/a.wav","url":"https://public.example.com/a.wav?token=secret","urlExpiresAt":1784073600000,"mediaType":"audio/wav"}}`))
	}))
	defer srv.Close()

	c := New(WithBaseURL(srv.URL), WithAPIKey("key-1"))
	artifact, err := c.ResolveMedia(context.Background(), &dto.ResolveMediaRequest{
		OSSKey:    "ai-hub/public-media/audio/a.wav",
		MediaType: "audio/wav",
	})
	if err != nil {
		t.Fatalf("resolve media: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/media/resolve" || gotAPIKey != "key-1" {
		t.Fatalf("method=%q path=%q apiKey=%q", gotMethod, gotPath, gotAPIKey)
	}
	if gotBody != `{"ossKey":"ai-hub/public-media/audio/a.wav","mediaType":"audio/wav"}` {
		t.Fatalf("body=%s", gotBody)
	}
	if artifact.OSSKey != "ai-hub/public-media/audio/a.wav" ||
		artifact.URL != "https://public.example.com/a.wav?token=secret" ||
		artifact.URLExpiresAt != 1784073600000 || artifact.MediaType != "audio/wav" {
		t.Fatalf("artifact=%+v", artifact)
	}
}

func TestSubmitImageJob(t *testing.T) {
	var m, p string
	srv := newJSONServer(t, `{"status":200,"data":{"jobId":"job-image-123"}}`, &m, &p)
	defer srv.Close()
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))

	req := &dto.ImageJobRequest{
		ImageRequest: dto.ImageRequest{Model: "image-pro", Prompt: "poster", N: 1},
		CallbackURL:  "https://biz.example/callback",
	}
	req.SetCallbackSecret("derived-secret")
	jobID, err := c.SubmitImageJob(context.Background(), req)
	if err != nil {
		t.Fatalf("submit image err: %v", err)
	}
	if jobID != "job-image-123" || p != "/v1/images/jobs" || m != http.MethodPost {
		t.Fatalf("jobID=%q path=%s method=%s", jobID, p, m)
	}
	if req.CallbackSecret() != "derived-secret" {
		t.Fatalf("callback secret changed: %q", req.CallbackSecret())
	}
}

func TestSubmitVideoJobAndGetJob(t *testing.T) {
	var m, p string
	srvSubmit := newJSONServer(t, `{"status":200,"data":{"jobId":"job-123","durationAdjustment":{"requestedDuration":1,"actualDuration":4,"reason":"below_model_minimum"}}}`, &m, &p)
	defer srvSubmit.Close()
	c := New(WithBaseURL(srvSubmit.URL), WithAPIKey("k"))
	submit, err := c.SubmitVideoJob(context.Background(), &dto.VideoJobRequest{Model: "v", Task: dto.VideoTaskText2Video})
	if err != nil {
		t.Fatalf("submit err: %v", err)
	}
	if submit.JobID != "job-123" || submit.DurationAdjustment == nil || submit.DurationAdjustment.ActualDuration != 4 || p != "/v1/videos/jobs" {
		t.Fatalf("submit=%+v path=%s", submit, p)
	}

	srvGet := newJSONServer(t, `{"status":200,"data":{"jobId":"job-123","state":"succeeded","artifacts":[{"ossKey":"ai-hub/public-media/video/out.mp4","url":"https://public.example.com/out.mp4?signature=secret","urlExpiresAt":1784073600000,"mediaType":"video/mp4"}]}}`, &m, &p)
	defer srvGet.Close()
	c2 := New(WithBaseURL(srvGet.URL), WithAPIKey("k"))
	res, err := c2.GetJob(context.Background(), "job-123")
	if err != nil {
		t.Fatalf("get err: %v", err)
	}
	if res.State != dto.JobStateSucceeded || len(res.Artifacts) != 1 ||
		res.Artifacts[0].URL != "https://public.example.com/out.mp4?signature=secret" ||
		res.Artifacts[0].URLExpiresAt != 1784073600000 ||
		p != "/v1/media/jobs/job-123" || m != http.MethodGet {
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
	srv := newJSONServer(t, `{"status":200,"data":{"id":"s1","ossKey":"ai-hub/public-media/audio/a.wav","url":"https://public.example.com/a.wav?signature=secret","urlExpiresAt":1784073600000,"audioOssKey":"ai-hub/public-media/audio/a.wav","mediaType":"audio/wav"}}`, &m, &p)
	defer srv.Close()
	c := New(WithBaseURL(srv.URL), WithAPIKey("k"))
	speech, err := c.GenerateSpeech(context.Background(), &dto.SpeechRequest{Voice: "v", Text: "hi"})
	if err != nil {
		t.Fatalf("speech err: %v", err)
	}
	if p != "/v1/speech/generate" || speech.OSSKey != "ai-hub/public-media/audio/a.wav" ||
		speech.AudioOssKey != speech.OSSKey || speech.URL != "https://public.example.com/a.wav?signature=secret" ||
		speech.URLExpiresAt != 1784073600000 || speech.MediaType != "audio/wav" {
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

	srv3 := newJSONServer(t, `{"status":200,"data":{"logicalVoiceId":7,"succeeded":[{"channelModelId":1,"bindingId":2,"vendorVoiceId":"voice-1","ossKey":"ai-hub/public-media/audio/preview.wav","url":"https://public.example.com/preview.wav?signature=secret","urlExpiresAt":1784073600000,"mediaType":"audio/wav","previewOssKey":"ai-hub/public-media/audio/preview.wav","previewMediaType":"audio/wav"}],"failed":[]}}`, &m, &p)
	defer srv3.Close()
	c3 := New(WithBaseURL(srv3.URL), WithAPIKey("k"))
	res, err := c3.CreateVoice(context.Background(), &dto.CreateVoiceRequest{Name: "n", Source: dto.VoiceSourceClone})
	if err != nil {
		t.Fatalf("createvoice err: %v", err)
	}
	if res.LogicalVoiceID != 7 || len(res.Succeeded) != 1 ||
		res.Succeeded[0].URL != "https://public.example.com/preview.wav?signature=secret" ||
		res.Succeeded[0].URLExpiresAt != 1784073600000 ||
		res.Succeeded[0].PreviewOssKey != res.Succeeded[0].OSSKey ||
		res.Succeeded[0].PreviewMediaType != res.Succeeded[0].MediaType || p != "/v1/voices" {
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
		_, _ = w.Write([]byte(`{"status":200,"data":{"ossKey":"ai-hub/public-media/image/20260621/a.png","url":"https://public.example.com/a.png?signature=secret","urlExpiresAt":1784073600000,"mediaType":"image/png","size":11,"kind":"image"}}`))
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
	if res.OSSKey != "ai-hub/public-media/image/20260621/a.png" ||
		res.URL != "https://public.example.com/a.png?signature=secret" ||
		res.URLExpiresAt != 1784073600000 || res.MediaType != "image/png" || res.Size != 11 {
		t.Fatalf("res=%+v", res)
	}
}
