// Package dto 的图片生成部分：定义图片生成统一中立协议。
//
// 职责：
//   - 提供业务方与 ai-hub 之间厂商无关的图片生成请求/结果契约
//
// 边界：
//   - 纯数据类型，无业务方法
//   - 不引用 internal/model，不引用任何厂商 SDK 类型
package dto

// ImageRequest 统一图片生成请求。
type ImageRequest struct {
	Model       string            `json:"model"`                 // 逻辑模型名
	Prompt      string            `json:"prompt"`                // 文本提示词
	AspectRatio string            `json:"aspectRatio,omitempty"` // 中立比例，如 1:1 / 2:3 / 3:2
	Resolution  string            `json:"resolution,omitempty"`  // 中立分辨率档位，如 standard / high
	Size        string            `json:"size,omitempty"`        // Deprecated: 厂商侧尺寸值，仅兼容旧调用
	N           int               `json:"n,omitempty"`           // 生成张数；<=0 视为 1
	RefImages   []string          `json:"refImages,omitempty"`   // 参考图（url 或 base64），部分厂商支持
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ImageArtifact 一张生成产物，已转存为永久 OSS 引用。
type ImageArtifact struct {
	Ref       string `json:"ref"`       // OSS objectKey/ref
	MediaType string `json:"mediaType"` // 如 image/png
}

// ImageResult 统一图片生成结果（非流式）。
type ImageResult struct {
	ID                 string          `json:"id"`
	Model              string          `json:"model"`              // 业务方请求的逻辑模型名
	ActualChannelModel string          `json:"actualChannelModel"` // 命中候选标识
	Artifacts          []ImageArtifact `json:"artifacts"`
	Usage              Usage           `json:"usage"`
	Cost               Cost            `json:"cost"`
}
