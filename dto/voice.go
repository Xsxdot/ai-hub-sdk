// Package dto 的语音部分：TTS/ASR/声音设计·复刻的统一中立契约。
//
// 职责：
//   - 提供业务方与 ai-hub 之间厂商无关的语音请求/结果契约
//
// 边界：
//   - 纯数据类型，不引用 internal/model 与厂商 SDK
package dto

// VoiceSource 音色来源。
type VoiceSource string

const (
	VoiceSourceDesign VoiceSource = "design"
	VoiceSourceClone  VoiceSource = "clone"
)

// CreateVoiceRequest 创建逻辑音色（多渠道容灾）请求。
type CreateVoiceRequest struct {
	Name            string      `json:"name"`
	VoicePrompt     string      `json:"voicePrompt,omitempty"`
	PreviewText     string      `json:"previewText,omitempty"`
	Source          VoiceSource `json:"source"`
	ChannelModelIDs []int64     `json:"channelModelIds,omitempty"` // 空则用默认创建渠道集
	RefAudio        *MediaRef   `json:"refAudio,omitempty"`        // clone: 公网 URL 或 ai-hub ossKey
}

// VoiceBindingResult 单个渠道绑定的创建结果。
type VoiceBindingResult struct {
	ChannelModelID   int64  `json:"channelModelId"`
	BindingID        int64  `json:"bindingId,omitempty"`
	VendorVoiceID    string `json:"vendorVoiceId,omitempty"`
	PreviewOssKey    string `json:"previewOssKey,omitempty"`
	PreviewMediaType string `json:"previewMediaType,omitempty"`
	Reason           string `json:"reason,omitempty"` // 失败原因
}

// CreateVoiceResult 创建逻辑音色结果，含逐渠道成败明细。
type CreateVoiceResult struct {
	LogicalVoiceID int64                `json:"logicalVoiceId"`
	Succeeded      []VoiceBindingResult `json:"succeeded"`
	Failed         []VoiceBindingResult `json:"failed"`
}

// SpeechRequest TTS 合成请求。
type SpeechRequest struct {
	Voice          string            `json:"voice"`                    // 逻辑音色名
	VoiceBindingID int64             `json:"voiceBindingId,omitempty"` // 可选：钉死当前逻辑音色下的某个绑定
	Text           string            `json:"text"`
	Format         string            `json:"format,omitempty"`
	SampleRate     int               `json:"sampleRate,omitempty"`
	Options        map[string]any    `json:"options,omitempty"` // volume/rate/pitch
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// SpeechResult TTS 合成结果。
type SpeechResult struct {
	ID                 string `json:"id"`
	Voice              string `json:"voice"`
	VoiceBindingID     int64  `json:"voiceBindingId,omitempty"`
	ActualChannelModel string `json:"actualChannelModel"`
	AudioOssKey        string `json:"audioOssKey"` // 永久 OSS 引用
	MediaType          string `json:"mediaType"`
	Usage              Usage  `json:"usage"`
	Cost               Cost   `json:"cost"`
}

// TranscribeRequest ASR 识别请求。
type TranscribeRequest struct {
	Model    string            `json:"model"`
	Audio    *MediaRef         `json:"audio"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TranscribeResult ASR 识别结果。
type TranscribeResult struct {
	ID                 string `json:"id"`
	Model              string `json:"model"`
	ActualChannelModel string `json:"actualChannelModel"`
	Text               string `json:"text"`
	Usage              Usage  `json:"usage"`
	Cost               Cost   `json:"cost"`
}
