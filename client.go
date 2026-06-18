// client.go 定义 ai-hub SDK 的 Client 入口。
//
// 职责：
//   - 持有 baseURL / apiKey / httpClient
//   - 作为 Chat、流式 Chat、媒体能力方法的接收者
//
// 边界：
//   - 仅做 HTTP 编解码与转发，不含业务逻辑
//   - 不隐式读取全局配置，不管理重试、熔断或计费
package aihubsdk

import (
	"net/http"
	"strings"

	"github.com/xsxdot/ai-hub-sdk/aihubapi"
)

// Client 是 ai-hub 的跨进程 HTTP 客户端。
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// 编译期断言：Client 必须实现两个对外契约。
var _ aihubapi.IAIHub = (*Client)(nil)
var _ aihubapi.IAIHubMedia = (*Client)(nil)

// New 构造 Client。
//
// 参数：
//   - opts: 函数式选项（WithBaseURL/WithAPIKey/WithHTTPClient/WithTimeout）
//
// 返回：
//   - 已配置好的 *Client；未指定 httpClient 时使用默认值
func New(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	// baseURL 去尾斜杠，避免拼接出 //v1/chat。
	c.baseURL = strings.TrimRight(c.baseURL, "/")
	return c
}
