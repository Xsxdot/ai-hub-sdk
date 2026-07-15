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
	RefImages   []MediaRef        `json:"refImages,omitempty"`   // 参考图，支持公网 URL 或 ai-hub ossKey
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ImageJobRequest 统一图片异步任务提交请求。
//
// 参数：
//   - ImageRequest: 同步图片生成的中立请求字段
//   - CallbackURL: 业务方回调地址，空表示只轮询
//
// 注意：
//   - callbackSecret 由 server HTTP 层从 API Key 派生，业务方 JSON 不能直接写入。
//   - 同步 ImageRequest 不包含 callback 字段，避免同步接口出现无效参数。
type ImageJobRequest struct {
	ImageRequest
	CallbackURL string `json:"callbackUrl,omitempty"`

	callbackSecret string `json:"-"`
}

// SetCallbackSecret 注入回调签名密钥。
//
// 参数：
//   - secret: hex(sha256(apiKey)) 派生值
func (r *ImageJobRequest) SetCallbackSecret(secret string) { r.callbackSecret = secret }

// CallbackSecret 返回注入的回调签名密钥。
//
// 返回：
//   - HTTP 层注入的回调签名密钥；未注入时为空字符串
func (r *ImageJobRequest) CallbackSecret() string { return r.callbackSecret }

// ImageArtifact 是统一 MediaArtifact 的兼容别名。
//
// Deprecated: 新代码统一使用 MediaArtifact；别名仅保留现有调用方源码兼容性。
type ImageArtifact = MediaArtifact

// ImageResult 统一图片生成结果（非流式）。
type ImageResult struct {
	ID                 string          `json:"id"`
	Model              string          `json:"model"`              // 业务方请求的逻辑模型名
	ActualChannelModel string          `json:"actualChannelModel"` // 命中候选标识
	Artifacts          []ImageArtifact `json:"artifacts"`
	Usage              Usage           `json:"usage"`
	Cost               Cost            `json:"cost"`
}
