// config.go 负责 realtest 示例命令的环境变量配置解析。
//
// 职责：
//   - 从环境变量读取 ai-hub 地址、API Key、逻辑模型名和超时配置
//   - 提供默认模型名，便于一条命令覆盖 chat/image/video/asr smoke
//
// 边界：
//   - 不发起网络请求，不保存 API Key
//   - 不做业务调用编排，真实调用在 main.go 中执行
package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL            = "http://localhost:10100"
	defaultChatModel          = "normal-chat"
	defaultImageModel         = "image"
	defaultVideoModel         = "video"
	defaultASRModel           = "asr"
	defaultAudioURL           = "https://raw.githubusercontent.com/openai/whisper/main/tests/jfk.flac"
	defaultVideoPollEvery     = 3 * time.Second
	defaultHTTPTimeout        = 2 * time.Minute
	defaultTranscribeTimeout  = 2 * time.Minute
	defaultImagePrompt        = "A small red cube on a white table, product photo style"
	defaultVideoPrompt        = "A red cube slowly rotating on a white table"
	defaultChatPrompt         = "Reply with exactly one short sentence: ai-hub sdk smoke test ok"
	defaultVideoInitialPoll   = 1
	defaultImageArtifactLimit = 3
)

// Config 是 realtest 示例命令的全部运行配置。
//
// 字段：
//   - APIKey: 从 AIHUB_API_KEY 读取，必须显式提供
//   - BaseURL: ai-hub 服务地址，默认 http://localhost:10100
//   - *Model: 各模态逻辑模型名，默认使用本地测试环境中的 normal-chat/image/video/asr
//   - RunStream/WaitVideo: 控制是否额外测试流式对话、是否等待视频终态
type Config struct {
	BaseURL           string
	APIKey            string
	ChatModel         string
	ImageModel        string
	VideoModel        string
	ASRModel          string
	AudioURL          string
	ChatPrompt        string
	ImagePrompt       string
	VideoPrompt       string
	RunStream         bool
	WaitVideo         bool
	VideoTimeout      time.Duration
	VideoPollEvery    time.Duration
	HTTPTimeout       time.Duration
	TranscribeTimeout time.Duration
	VideoInitialPolls int
	ImageArtifactMax  int
}

// LoadConfig 从 lookup 读取环境变量并返回 realtest 配置。
//
// 参数：
//   - lookup: 环境变量读取函数，生产环境传 os.Getenv，测试传 map lookup
//
// 返回：
//   - Config: 已填充默认值的配置
//   - error: 缺少 AIHUB_API_KEY 或 duration/int/bool 解析失败
func LoadConfig(lookup func(string) string) (Config, error) {
	cfg := Config{
		BaseURL:           valueOrDefault(lookup("AIHUB_BASE_URL"), defaultBaseURL),
		APIKey:            strings.TrimSpace(lookup("AIHUB_API_KEY")),
		ChatModel:         valueOrDefault(lookup("AIHUB_CHAT_MODEL"), defaultChatModel),
		ImageModel:        valueOrDefault(lookup("AIHUB_IMAGE_MODEL"), defaultImageModel),
		VideoModel:        valueOrDefault(lookup("AIHUB_VIDEO_MODEL"), defaultVideoModel),
		ASRModel:          valueOrDefault(lookup("AIHUB_ASR_MODEL"), defaultASRModel),
		AudioURL:          valueOrDefault(lookup("AIHUB_AUDIO_URL"), defaultAudioURL),
		ChatPrompt:        valueOrDefault(lookup("AIHUB_CHAT_PROMPT"), defaultChatPrompt),
		ImagePrompt:       valueOrDefault(lookup("AIHUB_IMAGE_PROMPT"), defaultImagePrompt),
		VideoPrompt:       valueOrDefault(lookup("AIHUB_VIDEO_PROMPT"), defaultVideoPrompt),
		VideoPollEvery:    defaultVideoPollEvery,
		HTTPTimeout:       defaultHTTPTimeout,
		TranscribeTimeout: defaultTranscribeTimeout,
		VideoInitialPolls: defaultVideoInitialPoll,
		ImageArtifactMax:  defaultImageArtifactLimit,
	}
	if cfg.APIKey == "" {
		return Config{}, fmt.Errorf("AIHUB_API_KEY is required")
	}

	var err error
	if cfg.RunStream, err = parseBoolEnv(lookup, "AIHUB_RUN_STREAM", false); err != nil {
		return Config{}, err
	}
	if cfg.WaitVideo, err = parseBoolEnv(lookup, "AIHUB_WAIT_VIDEO", false); err != nil {
		return Config{}, err
	}
	if cfg.VideoTimeout, err = parseDurationEnv(lookup, "AIHUB_VIDEO_TIMEOUT", 0); err != nil {
		return Config{}, err
	}
	if cfg.VideoPollEvery, err = parseDurationEnv(lookup, "AIHUB_VIDEO_POLL_EVERY", cfg.VideoPollEvery); err != nil {
		return Config{}, err
	}
	if cfg.HTTPTimeout, err = parseDurationEnv(lookup, "AIHUB_HTTP_TIMEOUT", cfg.HTTPTimeout); err != nil {
		return Config{}, err
	}
	if cfg.TranscribeTimeout, err = parseDurationEnv(lookup, "AIHUB_TRANSCRIBE_TIMEOUT", cfg.TranscribeTimeout); err != nil {
		return Config{}, err
	}
	if cfg.VideoInitialPolls, err = parseIntEnv(lookup, "AIHUB_VIDEO_INITIAL_POLLS", cfg.VideoInitialPolls); err != nil {
		return Config{}, err
	}
	if cfg.ImageArtifactMax, err = parseIntEnv(lookup, "AIHUB_IMAGE_ARTIFACT_MAX", cfg.ImageArtifactMax); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func valueOrDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func parseBoolEnv(lookup func(string) string, key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(lookup(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a bool: %w", key, err)
	}
	return v, nil
}

func parseDurationEnv(lookup func(string) string, key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(lookup(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be a Go duration like 30s or 2m: %w", key, err)
	}
	return v, nil
}

func parseIntEnv(lookup func(string) string, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(lookup(key))
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return v, nil
}
