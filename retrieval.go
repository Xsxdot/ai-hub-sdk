// retrieval.go 实现 IAIHub 契约：Embedding（文本向量化）与 Rerank（文本重排序）。
//
// 职责：
//   - 对齐 /v1/embeddings 与 /v1/rerank 的非流式同步调用
//   - 复用统一 doJSON，剥 result.OK 壳后返回中立 DTO
//
// 边界：
//   - 不做模型选择、重试、计费或厂商协议适配
//   - 不修改调用方传入的 Options 逃生舱
package aihubsdk

import (
	"context"
	"net/http"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

// Embedding 同步文本向量化。
//
// 参数：
//   - ctx: 请求上下文，用于取消 HTTP 调用
//   - req: 中立向量化请求
//
// 返回：
//   - *dto.EmbeddingResponse 或 *APIError / 网络错误
func (c *Client) Embedding(ctx context.Context, req *dto.EmbeddingRequest) (*dto.EmbeddingResponse, error) {
	var resp dto.EmbeddingResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/embeddings", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Rerank 同步文本重排序。
//
// 参数：
//   - ctx: 请求上下文，用于取消 HTTP 调用
//   - req: 中立重排序请求
//
// 返回：
//   - *dto.RerankResponse 或 *APIError / 网络错误
func (c *Client) Rerank(ctx context.Context, req *dto.RerankRequest) (*dto.RerankResponse, error) {
	var resp dto.RerankResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/rerank", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
