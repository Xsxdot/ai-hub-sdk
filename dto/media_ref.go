// Package dto 的媒体引用部分：定义公开 API/SDK 中的媒体输入引用。
//
// 职责：
//   - 区分公网 URL 与 ai-hub 发放的 ossKey，消除业务方自有 OSS key 的歧义
//   - 提供轻量校验，帮助 SDK 和 server 在边界处给出一致错误
//
// 边界：
//   - 不访问网络或 OSS
//   - 不判断 ossKey 是否存在；存在性由 server 的 PayloadStore/预览 URL 处理
package dto

import (
	"fmt"
	"net/url"
	"strings"
)

// MediaRefType 媒体引用类型。
type MediaRefType string

const (
	// MediaRefTypeURL 表示公网 HTTP(S) URL。
	MediaRefTypeURL MediaRefType = "url"
	// MediaRefTypeOSSKey 表示 ai-hub 发放的对象 key。
	MediaRefTypeOSSKey MediaRefType = "ossKey"
)

// MediaRef 是公开媒体输入的显式引用。
type MediaRef struct {
	Type      MediaRefType `json:"type"`
	URL       string       `json:"url,omitempty"`
	OSSKey    string       `json:"ossKey,omitempty"`
	MediaType string       `json:"mediaType,omitempty"`
}

// URLMediaRef 构造公网 URL 媒体引用。
func URLMediaRef(rawURL string, mediaType string) MediaRef {
	return MediaRef{Type: MediaRefTypeURL, URL: rawURL, MediaType: mediaType}
}

// OSSKeyMediaRef 构造 ai-hub ossKey 媒体引用。
func OSSKeyMediaRef(ossKey string, mediaType string) MediaRef {
	return MediaRef{Type: MediaRefTypeOSSKey, OSSKey: ossKey, MediaType: mediaType}
}

// Validate 校验 MediaRef 的形状，不校验 ossKey 是否存在。
func (r MediaRef) Validate() error {
	switch r.Type {
	case MediaRefTypeURL:
		if strings.TrimSpace(r.URL) == "" {
			return fmt.Errorf("media ref url is empty")
		}
		if strings.TrimSpace(r.OSSKey) != "" {
			return fmt.Errorf("media ref url cannot include ossKey")
		}
		parsed, err := url.Parse(strings.TrimSpace(r.URL))
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("media ref url must be http or https")
		}
		return nil
	case MediaRefTypeOSSKey:
		if strings.TrimSpace(r.OSSKey) == "" {
			return fmt.Errorf("media ref ossKey is empty")
		}
		if strings.TrimSpace(r.URL) != "" {
			return fmt.Errorf("media ref ossKey cannot include url")
		}
		key := strings.TrimSpace(r.OSSKey)
		if strings.HasPrefix(key, "oss://") || strings.Contains(key, "://") {
			return fmt.Errorf("media ref ossKey must be ai-hub object key without scheme")
		}
		return nil
	default:
		return fmt.Errorf("media ref type must be url or ossKey")
	}
}

// MediaUploadResult 是 POST /v1/media 返回的上传结果。
type MediaUploadResult struct {
	OSSKey       string `json:"ossKey"`
	URL          string `json:"url"`
	URLExpiresAt int64  `json:"urlExpiresAt"`
	MediaType    string `json:"mediaType"`
	Size         int    `json:"size"`
	Kind         string `json:"kind"`
}
