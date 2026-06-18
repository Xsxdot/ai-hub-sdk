// chat.go 实现 IAIHub 契约：Chat（非流式）与 ChatStream（SSE 流式）。
//
// 职责：
//   - 对齐 /v1/chat 路由的非流式与流式调用
//   - 强制设置 stream 标志，避免调用方误用路由形态
//
// 边界：
//   - 不拼接 prompt、不做模型选择或重试
//   - 不消费流事件，业务方自行读取返回 channel
package aihubsdk

import (
	"context"
	"net/http"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

// Chat 非流式对话。强制 stream=false，剥壳返回 ChatResponse。
//
// 参数：
//   - req: 统一对话请求；本方法会把 Stream 置 false
//
// 返回：
//   - *dto.ChatResponse 或 *APIError / 网络错误
func (c *Client) Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, error) {
	req.Stream = false
	var resp dto.ChatResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/chat", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ChatStream 流式对话。强制 stream=true，返回逐帧 StreamEvent channel。
//
// 注意：
//   - channel 关闭表示流结束；ctx 取消会断流并关闭 channel
//   - 业务方需消费完 channel 或取消 ctx，避免 goroutine/连接泄漏
func (c *Client) ChatStream(ctx context.Context, req *dto.ChatRequest) (<-chan dto.StreamEvent, error) {
	req.Stream = true
	return c.doSSE(ctx, "/v1/chat", req)
}
