// Package dto 测试公开媒体响应统一使用 url 的 JSON 契约。
//
// 职责：
//   - 集中守卫图片、视频、音频、上传、异步任务和 callback body 的媒体字段
//   - 验证临时地址过期时间使用 Unix 毫秒数，兼容字段与规范字段保持同值
//   - 验证新旧 JSON 消费者在兼容窗口内可双向解码
//
// 边界：
//   - 仅验证 SDK 公开 DTO 的 JSON 契约，不访问 AI-HUB 服务或 OSS
//   - 使用代表性 fixture 检查协议，不反射 Go 结构体实现细节
package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const (
	publicMediaContractURL       = "https://public.example.com/media?signature=secret"
	publicMediaContractExpiresAt = int64(1784073600000)
)

// TestPublicMediaContractGuardUsesOnlyUnifiedURL 验证所有代表性媒体响应递归只暴露 url。
func TestPublicMediaContractGuardUsesOnlyUnifiedURL(t *testing.T) {
	imageKey := "ai-hub/public-media/image/out.png"
	audioKey := "ai-hub/public-media/audio/out.mp3"
	videoKey := "ai-hub/public-media/video/out.mp4"
	responses := []struct {
		name  string
		value any
	}{
		{
			name: "media artifact",
			value: MediaArtifact{OSSKey: imageKey, URL: publicMediaContractURL,
				URLExpiresAt: publicMediaContractExpiresAt, MediaType: "image/png"},
		},
		{
			name: "image result",
			value: ImageResult{Artifacts: []ImageArtifact{{OSSKey: imageKey, URL: publicMediaContractURL,
				URLExpiresAt: publicMediaContractExpiresAt, MediaType: "image/png"}}},
		},
		{
			name: "media upload result",
			value: MediaUploadResult{OSSKey: videoKey, URL: publicMediaContractURL,
				URLExpiresAt: publicMediaContractExpiresAt, MediaType: "video/mp4", Size: 42, Kind: "video"},
		},
		{
			name: "speech result",
			value: SpeechResult{OSSKey: audioKey, URL: publicMediaContractURL,
				URLExpiresAt: publicMediaContractExpiresAt, AudioOssKey: audioKey, MediaType: "audio/mpeg"},
		},
		{
			name: "voice preview result",
			value: CreateVoiceResult{Succeeded: []VoiceBindingResult{{OSSKey: audioKey, URL: publicMediaContractURL,
				URLExpiresAt: publicMediaContractExpiresAt, MediaType: "audio/mpeg",
				PreviewOssKey: audioKey, PreviewMediaType: "audio/mpeg"}}},
		},
		{
			// 查询响应与 callback body 共用 MediaJobResult，必须由同一守卫覆盖。
			name: "media job query and callback body",
			value: MediaJobResult{Artifacts: []MediaArtifact{{OSSKey: videoKey, URL: publicMediaContractURL,
				URLExpiresAt: publicMediaContractExpiresAt, MediaType: "video/mp4"}}},
		},
	}

	for _, response := range responses {
		t.Run(response.name, func(t *testing.T) {
			raw, err := json.Marshal(response.value)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			decoded := decodePublicMediaContractJSON(t, raw)
			if count := assertOnlyUnifiedPublicMediaURLs(t, decoded, "$"); count == 0 {
				t.Fatalf("response has no unified url field: %s", raw)
			}
		})
	}
}

// TestPublicMediaContractGuardLegacyFieldsMirrorCanonicalFields 验证兼容字段与规范字段同值。
func TestPublicMediaContractGuardLegacyFieldsMirrorCanonicalFields(t *testing.T) {
	const audioKey = "ai-hub/public-media/audio/out.mp3"
	speechRaw, err := json.Marshal(SpeechResult{OSSKey: audioKey, URL: publicMediaContractURL,
		URLExpiresAt: publicMediaContractExpiresAt, AudioOssKey: audioKey, MediaType: "audio/mpeg"})
	if err != nil {
		t.Fatalf("marshal speech result: %v", err)
	}
	speech := decodePublicMediaContractJSON(t, speechRaw).(map[string]any)
	assertPublicMediaStringFieldsEqual(t, speech, "ossKey", "audioOssKey")

	voiceRaw, err := json.Marshal(VoiceBindingResult{OSSKey: audioKey, URL: publicMediaContractURL,
		URLExpiresAt: publicMediaContractExpiresAt, MediaType: "audio/mpeg",
		PreviewOssKey: audioKey, PreviewMediaType: "audio/mpeg"})
	if err != nil {
		t.Fatalf("marshal voice binding result: %v", err)
	}
	voice := decodePublicMediaContractJSON(t, voiceRaw).(map[string]any)
	assertPublicMediaStringFieldsEqual(t, voice, "ossKey", "previewOssKey")
	assertPublicMediaStringFieldsEqual(t, voice, "mediaType", "previewMediaType")
}

// TestPublicMediaContractGuardOldConsumersIgnoreNewFields 验证旧消费者可忽略新增交付字段。
func TestPublicMediaContractGuardOldConsumersIgnoreNewFields(t *testing.T) {
	t.Run("artifact", func(t *testing.T) {
		var legacy struct {
			OSSKey    string `json:"ossKey"`
			MediaType string `json:"mediaType"`
		}
		fixture := `{"ossKey":"ai-hub/public-media/image/out.png","url":"https://public.example.com/out.png?signature=secret","urlExpiresAt":1784073600000,"mediaType":"image/png"}`
		if err := json.Unmarshal([]byte(fixture), &legacy); err != nil {
			t.Fatalf("legacy artifact consumer: %v", err)
		}
		if legacy.OSSKey != "ai-hub/public-media/image/out.png" || legacy.MediaType != "image/png" {
			t.Fatalf("legacy artifact=%+v", legacy)
		}
	})

	t.Run("speech", func(t *testing.T) {
		var legacy struct {
			AudioOssKey string `json:"audioOssKey"`
			MediaType   string `json:"mediaType"`
		}
		fixture := `{"ossKey":"ai-hub/public-media/audio/out.mp3","url":"https://public.example.com/out.mp3?signature=secret","urlExpiresAt":1784073600000,"audioOssKey":"ai-hub/public-media/audio/out.mp3","mediaType":"audio/mpeg"}`
		if err := json.Unmarshal([]byte(fixture), &legacy); err != nil {
			t.Fatalf("legacy speech consumer: %v", err)
		}
		if legacy.AudioOssKey != "ai-hub/public-media/audio/out.mp3" || legacy.MediaType != "audio/mpeg" {
			t.Fatalf("legacy speech=%+v", legacy)
		}
	})

	t.Run("voice preview", func(t *testing.T) {
		var legacy struct {
			PreviewOssKey    string `json:"previewOssKey"`
			PreviewMediaType string `json:"previewMediaType"`
		}
		fixture := `{"ossKey":"ai-hub/public-media/audio/preview.mp3","url":"https://public.example.com/preview.mp3?signature=secret","urlExpiresAt":1784073600000,"mediaType":"audio/mpeg","previewOssKey":"ai-hub/public-media/audio/preview.mp3","previewMediaType":"audio/mpeg"}`
		if err := json.Unmarshal([]byte(fixture), &legacy); err != nil {
			t.Fatalf("legacy voice consumer: %v", err)
		}
		if legacy.PreviewOssKey != "ai-hub/public-media/audio/preview.mp3" || legacy.PreviewMediaType != "audio/mpeg" {
			t.Fatalf("legacy voice=%+v", legacy)
		}
	})
}

// TestPublicMediaContractGuardNewSDKDecodesLegacyKeyOnlyResponses 验证新 SDK 可读取旧 key-only 响应。
func TestPublicMediaContractGuardNewSDKDecodesLegacyKeyOnlyResponses(t *testing.T) {
	t.Run("media artifact", func(t *testing.T) {
		var got MediaArtifact
		decodeLegacyPublicMediaFixture(t, `{"ossKey":"ai-hub/public-media/image/out.png","mediaType":"image/png"}`, &got)
		if got.OSSKey == "" || got.MediaType != "image/png" || got.URL != "" || got.URLExpiresAt != 0 {
			t.Fatalf("artifact=%+v", got)
		}
	})

	t.Run("image result", func(t *testing.T) {
		var got ImageResult
		decodeLegacyPublicMediaFixture(t, `{"artifacts":[{"ossKey":"ai-hub/public-media/image/out.png","mediaType":"image/png"}]}`, &got)
		if len(got.Artifacts) != 1 || got.Artifacts[0].OSSKey == "" || got.Artifacts[0].URL != "" {
			t.Fatalf("image result=%+v", got)
		}
	})

	t.Run("media upload result", func(t *testing.T) {
		var got MediaUploadResult
		decodeLegacyPublicMediaFixture(t, `{"ossKey":"ai-hub/public-media/video/out.mp4","mediaType":"video/mp4","size":42,"kind":"video"}`, &got)
		if got.OSSKey == "" || got.Kind != "video" || got.URL != "" || got.URLExpiresAt != 0 {
			t.Fatalf("upload result=%+v", got)
		}
	})

	t.Run("speech result", func(t *testing.T) {
		var got SpeechResult
		decodeLegacyPublicMediaFixture(t, `{"audioOssKey":"ai-hub/public-media/audio/out.mp3","mediaType":"audio/mpeg"}`, &got)
		if got.AudioOssKey == "" || got.MediaType != "audio/mpeg" || got.URL != "" || got.URLExpiresAt != 0 {
			t.Fatalf("speech result=%+v", got)
		}
	})

	t.Run("voice preview result", func(t *testing.T) {
		var got CreateVoiceResult
		decodeLegacyPublicMediaFixture(t, `{"succeeded":[{"channelModelId":1,"previewOssKey":"ai-hub/public-media/audio/preview.mp3","previewMediaType":"audio/mpeg"}],"failed":[]}`, &got)
		if len(got.Succeeded) != 1 || got.Succeeded[0].PreviewOssKey == "" ||
			got.Succeeded[0].PreviewMediaType != "audio/mpeg" || got.Succeeded[0].URL != "" {
			t.Fatalf("voice result=%+v", got)
		}
	})

	t.Run("media job query and callback body", func(t *testing.T) {
		var got MediaJobResult
		decodeLegacyPublicMediaFixture(t, `{"jobId":"job-1","state":"succeeded","artifacts":[{"ossKey":"ai-hub/public-media/video/out.mp4","mediaType":"video/mp4"}]}`, &got)
		if len(got.Artifacts) != 1 || got.Artifacts[0].OSSKey == "" || got.Artifacts[0].URL != "" {
			t.Fatalf("job result=%+v", got)
		}
	})
}

func decodePublicMediaContractJSON(t *testing.T, raw []byte) any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("decode public media response: %v", err)
	}
	return decoded
}

func assertOnlyUnifiedPublicMediaURLs(t *testing.T, value any, path string) int {
	t.Helper()
	switch node := value.(type) {
	case map[string]any:
		urlCount := 0
		if rawURL, exists := node["url"]; exists {
			url, ok := rawURL.(string)
			if !ok || url == "" {
				t.Fatalf("%s.url=%v, want non-empty string", path, rawURL)
			}
			expiresRaw, exists := node["urlExpiresAt"]
			if !exists {
				t.Fatalf("%s.url has no urlExpiresAt", path)
			}
			expiresNumber, ok := expiresRaw.(json.Number)
			if !ok {
				t.Fatalf("%s.urlExpiresAt=%T(%v), want JSON number", path, expiresRaw, expiresRaw)
			}
			expiresAt, err := expiresNumber.Int64()
			if err != nil {
				t.Fatalf("%s.urlExpiresAt=%q is not an integer: %v", path, expiresNumber, err)
			}
			// 临时 URL 的过期时间处于当前时代；十位秒级值会被该范围明确拒绝。
			if expiresAt < 1_000_000_000_000 || expiresAt >= 10_000_000_000_000 {
				t.Fatalf("%s.urlExpiresAt=%d, want Unix milliseconds", path, expiresAt)
			}
			urlCount++
		}
		if _, exists := node["urlExpiresAt"]; exists {
			if _, hasURL := node["url"]; !hasURL {
				t.Fatalf("%s.urlExpiresAt has no matching url", path)
			}
		}
		for key, child := range node {
			lowerKey := strings.ToLower(key)
			// 响应中的媒体地址只有 url 一个名字，阻止 audioUrl/previewUrl/videoUrl 等分叉回流。
			if strings.HasSuffix(lowerKey, "url") && key != "url" {
				t.Fatalf("%s contains forbidden media URL field %q", path, key)
			}
			urlCount += assertOnlyUnifiedPublicMediaURLs(t, child, path+"."+key)
		}
		return urlCount
	case []any:
		urlCount := 0
		for index, child := range node {
			urlCount += assertOnlyUnifiedPublicMediaURLs(t, child, fmt.Sprintf("%s[%d]", path, index))
		}
		return urlCount
	default:
		return 0
	}
}

func assertPublicMediaStringFieldsEqual(t *testing.T, object map[string]any, canonical, legacy string) {
	t.Helper()
	canonicalValue, canonicalOK := object[canonical].(string)
	legacyValue, legacyOK := object[legacy].(string)
	if !canonicalOK || !legacyOK || canonicalValue == "" || canonicalValue != legacyValue {
		t.Fatalf("%s=%v %s=%v, want non-empty equal strings", canonical, object[canonical], legacy, object[legacy])
	}
}

func decodeLegacyPublicMediaFixture(t *testing.T, fixture string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(fixture), target); err != nil {
		t.Fatalf("decode legacy public media fixture: %v", err)
	}
}
