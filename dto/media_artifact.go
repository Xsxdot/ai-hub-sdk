// Package dto 的媒体交付部分定义公开 API 统一的媒体产物契约。
//
// 职责：
//   - 统一图片、视频和音频产物的稳定 OSS 身份与临时公网访问地址
//   - 定义单个媒体产物 URL 刷新请求
//
// 边界：
//   - 仅描述公开 JSON 契约，不生成、校验或持久化临时 URL
//   - 不访问 OSS，不允许调用方指定 URL 有效期
package dto

// MediaArtifact 是图片、视频和音频共用的公开媒体产物。
//
// 注意：
//   - OSSKey 是稳定身份，调用方应长期保存它
//   - URL 是可轮换的临时公网地址，不得作为产物身份或缓存键
//   - URLExpiresAt 使用 Unix 毫秒时间戳
type MediaArtifact struct {
	OSSKey       string `json:"ossKey"`
	URL          string `json:"url"`
	URLExpiresAt int64  `json:"urlExpiresAt"`
	MediaType    string `json:"mediaType"`
}

// ResolveMediaRequest 请求为一个 AI-HUB 媒体对象签发新的临时公网地址。
//
// 注意：
//   - OSSKey 必须是 AI-HUB 发放的对象 key，不能包含 URL scheme
//   - MediaType 可选；调用方不能通过该请求指定有效期
type ResolveMediaRequest struct {
	OSSKey    string `json:"ossKey"`
	MediaType string `json:"mediaType,omitempty"`
}
