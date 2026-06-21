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

func TestImageAndVoiceContractsUseMediaRefAndOSSKey(t *testing.T) {
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

	imageArtifact := ImageArtifact{OSSKey: "ai-hub/public-media/image/out.png", MediaType: "image/png"}
	rawImageArtifact, err := json.Marshal(imageArtifact)
	if err != nil {
		t.Fatalf("marshal image artifact: %v", err)
	}
	if string(rawImageArtifact) != `{"ossKey":"ai-hub/public-media/image/out.png","mediaType":"image/png"}` {
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

	voiceBinding := VoiceBindingResult{ChannelModelID: 1, PreviewOssKey: "ai-hub/public-media/audio/preview.wav", PreviewMediaType: "audio/wav"}
	rawVoiceBinding, err := json.Marshal(voiceBinding)
	if err != nil {
		t.Fatalf("marshal voice binding: %v", err)
	}
	if string(rawVoiceBinding) != `{"channelModelId":1,"previewOssKey":"ai-hub/public-media/audio/preview.wav","previewMediaType":"audio/wav"}` {
		t.Fatalf("voice binding json=%s", rawVoiceBinding)
	}

	speech := SpeechResult{ID: "speech-1", Voice: "v", ActualChannelModel: "cm", AudioOssKey: "ai-hub/public-media/audio/out.wav", MediaType: "audio/wav"}
	rawSpeech, err := json.Marshal(speech)
	if err != nil {
		t.Fatalf("marshal speech: %v", err)
	}
	if string(rawSpeech) != `{"id":"speech-1","voice":"v","actualChannelModel":"cm","audioOssKey":"ai-hub/public-media/audio/out.wav","mediaType":"audio/wav","usage":{"metrics":null},"cost":{"details":null,"total":0,"currency":""}}` {
		t.Fatalf("speech json=%s", rawSpeech)
	}

	transcribe := TranscribeRequest{Model: "asr", Audio: &audio}
	rawTranscribe, err := json.Marshal(transcribe)
	if err != nil {
		t.Fatalf("marshal transcribe: %v", err)
	}
	if string(rawTranscribe) != `{"model":"asr","audio":{"type":"ossKey","ossKey":"ai-hub/public-media/audio/ref.wav","mediaType":"audio/wav"}}` {
		t.Fatalf("transcribe json=%s", rawTranscribe)
	}
}

func mediaRefPtr(ref MediaRef) *MediaRef {
	return &ref
}
