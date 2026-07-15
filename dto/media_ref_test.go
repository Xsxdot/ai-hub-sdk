// Package dto tests public media reference validation.
//
// Responsibilities:
//   - Verify public URL refs and ai-hub ossKey refs serialize predictably
//   - Reject ambiguous OSS-like strings before server/provider code sees them
//
// Boundaries:
//   - Does not call ai-hub server or object storage
package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMediaRefValidateURL(t *testing.T) {
	ref := MediaRef{Type: MediaRefTypeURL, URL: "https://public.example.com/a.png", MediaType: "image/png"}
	if err := ref.Validate(); err != nil {
		t.Fatalf("validate url ref: %v", err)
	}
	raw, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) != `{"type":"url","url":"https://public.example.com/a.png","mediaType":"image/png"}` {
		t.Fatalf("json=%s", raw)
	}
}

func TestMediaRefValidateOSSKey(t *testing.T) {
	ref := MediaRef{Type: MediaRefTypeOSSKey, OSSKey: "ai-hub/public-media/image/20260621/a.png", MediaType: "image/png"}
	if err := ref.Validate(); err != nil {
		t.Fatalf("validate oss key ref: %v", err)
	}
}

func TestMediaRefRejectsAmbiguousValues(t *testing.T) {
	cases := []MediaRef{
		{Type: MediaRefTypeURL, URL: "oss://bucket/a.png"},
		{Type: MediaRefTypeURL, URL: "data:image/png;base64,AAAA"},
		{Type: MediaRefTypeOSSKey, OSSKey: "oss://ai-hub/public-media/image/a.png"},
		{Type: MediaRefTypeOSSKey, OSSKey: "https://public.example.com/a.png"},
		{Type: MediaRefTypeURL, URL: "https://public.example.com/a.png", OSSKey: "ai-hub/public-media/image/a.png"},
		{Type: "", URL: "https://public.example.com/a.png"},
	}
	for _, tc := range cases {
		if err := tc.Validate(); err == nil {
			t.Fatalf("expected invalid media ref: %+v", tc)
		}
	}
}

func TestImageAndVoiceContractsUseUnifiedMediaArtifact(t *testing.T) {
	imageReq := ImageRequest{
		Model:     "image-model",
		Prompt:    "cat",
		RefImages: []MediaRef{URLMediaRef("https://public.example.com/ref.png", "image/png")},
	}
	rawImageReq, err := json.Marshal(imageReq)
	if err != nil {
		t.Fatalf("marshal image request: %v", err)
	}
	if string(rawImageReq) != `{"model":"image-model","prompt":"cat","refImages":[{"type":"url","url":"https://public.example.com/ref.png","mediaType":"image/png"}]}` {
		t.Fatalf("image json=%s", rawImageReq)
	}

	imageArtifact := ImageArtifact{
		OSSKey:       "ai-hub/public-media/image/out.png",
		URL:          "https://public.example.com/out.png?token=secret",
		URLExpiresAt: 1784073600000,
		MediaType:    "image/png",
	}
	rawImageArtifact, err := json.Marshal(imageArtifact)
	if err != nil {
		t.Fatalf("marshal image artifact: %v", err)
	}
	if string(rawImageArtifact) != `{"ossKey":"ai-hub/public-media/image/out.png","url":"https://public.example.com/out.png?token=secret","urlExpiresAt":1784073600000,"mediaType":"image/png"}` {
		t.Fatalf("image artifact json=%s", rawImageArtifact)
	}

	audio := OSSKeyMediaRef("ai-hub/public-media/audio/ref.wav", "audio/wav")
	createVoiceReq := CreateVoiceRequest{Name: "clone", Source: VoiceSourceClone, RefAudio: &audio}
	rawCreateVoice, err := json.Marshal(createVoiceReq)
	if err != nil {
		t.Fatalf("marshal create voice: %v", err)
	}
	if string(rawCreateVoice) != `{"name":"clone","source":"clone","refAudio":{"type":"ossKey","ossKey":"ai-hub/public-media/audio/ref.wav","mediaType":"audio/wav"}}` {
		t.Fatalf("create voice json=%s", rawCreateVoice)
	}

	voiceBinding := VoiceBindingResult{
		ChannelModelID:   1,
		OSSKey:           "ai-hub/public-media/audio/preview.wav",
		URL:              "https://public.example.com/preview.wav?token=secret",
		URLExpiresAt:     1784073600000,
		MediaType:        "audio/wav",
		PreviewOssKey:    "ai-hub/public-media/audio/preview.wav",
		PreviewMediaType: "audio/wav",
	}
	rawVoiceBinding, err := json.Marshal(voiceBinding)
	if err != nil {
		t.Fatalf("marshal voice binding: %v", err)
	}
	assertUnifiedArtifactJSON(t, rawVoiceBinding, "ai-hub/public-media/audio/preview.wav", "audio/wav")
	assertJSONField(t, rawVoiceBinding, "previewOssKey", "ai-hub/public-media/audio/preview.wav")
	assertJSONField(t, rawVoiceBinding, "previewMediaType", "audio/wav")

	speech := SpeechResult{
		ID:                 "speech-1",
		Voice:              "v",
		ActualChannelModel: "cm",
		OSSKey:             "ai-hub/public-media/audio/out.wav",
		URL:                "https://public.example.com/out.wav?token=secret",
		URLExpiresAt:       1784073600000,
		AudioOssKey:        "ai-hub/public-media/audio/out.wav",
		MediaType:          "audio/wav",
	}
	rawSpeech, err := json.Marshal(speech)
	if err != nil {
		t.Fatalf("marshal speech: %v", err)
	}
	assertUnifiedArtifactJSON(t, rawSpeech, "ai-hub/public-media/audio/out.wav", "audio/wav")
	assertJSONField(t, rawSpeech, "audioOssKey", "ai-hub/public-media/audio/out.wav")

	transcribe := TranscribeRequest{Model: "asr", Audio: &audio}
	rawTranscribe, err := json.Marshal(transcribe)
	if err != nil {
		t.Fatalf("marshal transcribe: %v", err)
	}
	if string(rawTranscribe) != `{"model":"asr","audio":{"type":"ossKey","ossKey":"ai-hub/public-media/audio/ref.wav","mediaType":"audio/wav"}}` {
		t.Fatalf("transcribe json=%s", rawTranscribe)
	}
}

func TestMediaUploadResultUsesUnifiedMediaArtifact(t *testing.T) {
	result := MediaUploadResult{
		OSSKey:       "ai-hub/public-media/video/out.mp4",
		URL:          "https://public.example.com/out.mp4?token=secret",
		URLExpiresAt: 1784073600000,
		MediaType:    "video/mp4",
		Size:         42,
		Kind:         "video",
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal upload result: %v", err)
	}
	assertUnifiedArtifactJSON(t, raw, "ai-hub/public-media/video/out.mp4", "video/mp4")
	assertJSONField(t, raw, "kind", "video")
}

func assertUnifiedArtifactJSON(t *testing.T, raw []byte, ossKey, mediaType string) {
	t.Helper()
	assertJSONField(t, raw, "ossKey", ossKey)
	assertJSONField(t, raw, "url", "https://public.example.com/"+artifactFilename(ossKey)+"?token=secret")
	assertJSONField(t, raw, "mediaType", mediaType)

	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal artifact json: %v", err)
	}
	if got := fields["urlExpiresAt"]; got != float64(1784073600000) {
		t.Fatalf("urlExpiresAt=%v", got)
	}
	for _, forbidden := range []string{"audioUrl", "previewUrl", "videoUrl"} {
		if _, exists := fields[forbidden]; exists {
			t.Fatalf("forbidden field %q in %s", forbidden, raw)
		}
	}
}

func assertJSONField(t *testing.T, raw []byte, field, want string) {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if got := fields[field]; got != want {
		t.Fatalf("%s=%v, want %q; json=%s", field, got, want, raw)
	}
}

func artifactFilename(ossKey string) string {
	parts := strings.Split(ossKey, "/")
	return parts[len(parts)-1]
}

func mediaRefPtr(ref MediaRef) *MediaRef {
	return &ref
}
