// options.go 定义 ai-hub SDK Client 的函数式配置。
//
// 职责：
//   - 配置服务地址、API Key、HTTP 客户端与默认超时
//
// 边界：
//   - 不发起网络请求
//   - 不读取环境变量或配置文件，调用方显式注入
package aihubsdk

import (
	"net/http"
	"time"
)

// Option 配置 Client 的函数式选项。
type Option func(*Client)

// WithBaseURL 设置 ai-hub 服务地址（如 https://aihub.internal）。尾部斜杠会被去除。
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithAPIKey 设置注入到每个请求 X-API-Key 头的业务方凭证。
func WithAPIKey(k string) Option {
	return func(c *Client) { c.apiKey = k }
}

// WithHTTPClient 注入自定义 *http.Client（连接池/超时由调用方控制）。
//
// 注意：流式 ChatStream 不应依赖 http.Client.Timeout 做总超时（会截断长流），
// 流式生命周期由调用方传入的 ctx 控制。
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithTimeout 设置默认 *http.Client 的请求超时。与 WithHTTPClient 互斥时，
// 后调用者覆盖前者；建议二选一。
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}
		c.httpClient.Timeout = d
	}
}
