// Package dto 的检索部分：定义文本向量化与重排序的中立请求/响应契约。
//
// 职责：
//   - 定义 embedding/rerank 的统一请求与响应形状
//   - 通用核心字段强类型化，厂商特有参数走 Options 逃生舱透传
//
// 边界：
//   - 不含任何厂商协议细节（DashScope/OpenAI 形状差异在 server 端 Codec 处理）
//   - 不含网络与计费逻辑
package dto

// EmbeddingRequest 向量化请求（中立契约）。
type EmbeddingRequest struct {
	Model          string         `json:"model"`                     // 逻辑模型名
	Input          []string       `json:"input"`                     // 待向量化文本
	Dimensions     int            `json:"dimensions,omitempty"`      // 向量维度
	EncodingFormat string         `json:"encoding_format,omitempty"` // float / base64
	Options        map[string]any `json:"options,omitempty"`         // text_type/instruct/output_type 逃生舱
}

// EmbeddingItem 单条文本的向量结果。
type EmbeddingItem struct {
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

// EmbeddingResponse 向量化响应（中立契约）。
type EmbeddingResponse struct {
	Model string          `json:"model"`
	Data  []EmbeddingItem `json:"data"`
	Usage Usage           `json:"usage"`
}

// RerankRequest 重排序请求（中立契约）。
type RerankRequest struct {
	Model           string         `json:"model"`
	Query           string         `json:"query"`
	Documents       []string       `json:"documents"`
	TopN            int            `json:"top_n,omitempty"`
	ReturnDocuments bool           `json:"return_documents,omitempty"`
	Options         map[string]any `json:"options,omitempty"` // instruct 逃生舱
}

// RerankItem 单条重排序结果。
type RerankItem struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
	Document       string  `json:"document,omitempty"` // ReturnDocuments=true 时填充
}

// RerankResponse 重排序响应（中立契约）。
type RerankResponse struct {
	Model   string       `json:"model"`
	Results []RerankItem `json:"results"`
	Usage   Usage        `json:"usage"`
}
