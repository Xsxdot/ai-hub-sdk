// config_test.go 验证 realtest 示例命令的环境变量配置解析。
//
// 职责：
//   - 覆盖默认模型名、必填 API Key、duration/bool/int 环境变量解析
//   - 防止真实 smoke 示例在配置层静默跳过关键模态
//
// 边界：
//   - 不发起真实网络请求
//   - 不包含任何真实 API Key
package main

import (
	"testing"
	"time"
)

// TestLoadConfigUsesDefaultsAndEnv 验证默认模型名与显式环境变量会一起生效。
func TestLoadConfigUsesDefaultsAndEnv(t *testing.T) {
	env := map[string]string{
		"AIHUB_API_KEY":   "test-key",
		"AIHUB_BASE_URL":  "http://127.0.0.1:10100/",
		"AIHUB_AUDIO_URL": "https://example.com/audio.wav",
	}

	cfg, err := LoadConfig(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("LoadConfig err: %v", err)
	}

	if cfg.BaseURL != "http://127.0.0.1:10100/" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.APIKey != "test-key" {
		t.Fatalf("APIKey = %q", cfg.APIKey)
	}
	if cfg.ChatModel != "normal-chat" || cfg.ImageModel != "image" || cfg.VideoModel != "video" || cfg.ASRModel != "asr" {
		t.Fatalf("models = chat:%q image:%q video:%q asr:%q", cfg.ChatModel, cfg.ImageModel, cfg.VideoModel, cfg.ASRModel)
	}
	if cfg.VideoPollEvery != 3*time.Second || cfg.VideoTimeout != 0 {
		t.Fatalf("video polling = every %s timeout %s", cfg.VideoPollEvery, cfg.VideoTimeout)
	}
}

// TestLoadConfigRequiresAPIKey 验证示例命令必须从环境变量读取 API Key。
func TestLoadConfigRequiresAPIKey(t *testing.T) {
	_, err := LoadConfig(func(key string) string {
		if key == "AIHUB_BASE_URL" {
			return "http://localhost:10100"
		}
		return ""
	})
	if err == nil {
		t.Fatal("want missing API key error, got nil")
	}
}

// TestLoadConfigParsesDurations 验证超时和开关类环境变量按 Go duration/bool 解析。
func TestLoadConfigParsesDurations(t *testing.T) {
	env := map[string]string{
		"AIHUB_API_KEY":            "test-key",
		"AIHUB_VIDEO_TIMEOUT":      "1m",
		"AIHUB_VIDEO_POLL_EVERY":   "5s",
		"AIHUB_HTTP_TIMEOUT":       "15s",
		"AIHUB_RUN_STREAM":         "true",
		"AIHUB_WAIT_VIDEO":         "true",
		"AIHUB_TRANSCRIBE_TIMEOUT": "45s",
	}

	cfg, err := LoadConfig(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("LoadConfig err: %v", err)
	}

	if cfg.VideoTimeout != time.Minute || cfg.VideoPollEvery != 5*time.Second || cfg.HTTPTimeout != 15*time.Second {
		t.Fatalf("durations = video:%s poll:%s http:%s", cfg.VideoTimeout, cfg.VideoPollEvery, cfg.HTTPTimeout)
	}
	if !cfg.RunStream || !cfg.WaitVideo {
		t.Fatalf("flags = stream:%v waitVideo:%v", cfg.RunStream, cfg.WaitVideo)
	}
	if cfg.TranscribeTimeout != 45*time.Second {
		t.Fatalf("TranscribeTimeout = %s", cfg.TranscribeTimeout)
	}
}
