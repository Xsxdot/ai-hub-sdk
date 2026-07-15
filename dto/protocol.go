// Package dto 定义 aihub 的统一中立协议（请求/响应/流事件/计量）。
//
// 职责：
//   - 提供业务方与 ai-hub 之间语言无关、厂商无关的统一数据契约
//   - content blocks 模型天然容纳文本、图片、思维链、工具调用
//
// 边界：
//   - 纯数据类型，无业务方法
//   - 不引用 internal/model，不引用任何厂商 SDK 类型
package dto

// Role 消息角色。
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// BlockType 内容块类型。
type BlockType string

const (
	BlockText       BlockType = "text"
	BlockImage      BlockType = "image"
	BlockVideo      BlockType = "video"
	BlockThinking   BlockType = "thinking"
	BlockToolUse    BlockType = "tool_use"
	BlockToolResult BlockType = "tool_result"
)

// ContentBlock 内容块，统一承载多模态输入与思维链/工具输出。
// 不同 Type 使用不同字段子集（如 text 用 Text；image/video 用 Media）。
type ContentBlock struct {
	Type BlockType `json:"type"`
	// text / thinking
	Text string `json:"text,omitempty"`
	// image/video：使用显式 MediaRef 区分公网 URL 与 ai-hub 发放的 ossKey。
	Media *MediaRef `json:"media,omitempty"`
	// video：视频理解抽帧频率。0 表示调用方未显式指定，由 server codec 使用默认值。
	FPS float64 `json:"fps,omitempty"`
	// tool_use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
	// tool_result
	ToolUseID string `json:"toolUseId,omitempty"`
	// tool_result 的内容（文本化）
	Content string `json:"content,omitempty"`
}

// Message 一条消息。
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ThinkingLevel 统一思考档位。各厂商 Codec 负责映射/降级。
type ThinkingLevel string

const (
	ThinkingLow    ThinkingLevel = "low"
	ThinkingMedium ThinkingLevel = "medium"
	ThinkingHigh   ThinkingLevel = "high"
	ThinkingMax    ThinkingLevel = "max"
)

// ThinkingConfig 思考控制。
type ThinkingConfig struct {
	Enabled bool          `json:"enabled"`
	Level   ThinkingLevel `json:"level,omitempty"`
}

// ResponseFormatType 统一结构化输出格式。各厂商 Codec 负责映射到自己的请求字段。
type ResponseFormatType string

const (
	// ResponseFormatJSONObject 要求模型输出一个 JSON object。
	ResponseFormatJSONObject ResponseFormatType = "json_object"
)

// ResponseFormatConfig 控制模型输出格式。
type ResponseFormatConfig struct {
	Type ResponseFormatType `json:"type"`
}

// Tool 工具定义（function calling）。
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// ChatRequest 统一对话请求。
type ChatRequest struct {
	Model          string                `json:"model"`
	Messages       []Message             `json:"messages"`
	System         string                `json:"system,omitempty"`
	MaxTokens      int                   `json:"maxTokens,omitempty"`
	Temperature    float64               `json:"temperature,omitempty"`
	Thinking       *ThinkingConfig       `json:"thinking,omitempty"`
	ResponseFormat *ResponseFormatConfig `json:"responseFormat,omitempty"`
	Tools          []Tool                `json:"tools,omitempty"`
	Stream         bool                  `json:"stream"`
	Metadata       map[string]string     `json:"metadata,omitempty"`
}

// BillingMetric 计量维度。
type BillingMetric string

const (
	MetricTokenInput      BillingMetric = "token_input"
	MetricTokenCacheRead  BillingMetric = "token_cache_read"
	MetricTokenCacheWrite BillingMetric = "token_cache_write"
	MetricTokenOutput     BillingMetric = "token_output"
	MetricTokenThinking   BillingMetric = "token_thinking"
	MetricRequestCount    BillingMetric = "request_count"
	MetricDurationSecond  BillingMetric = "duration_second"
	MetricCharacterCount  BillingMetric = "character_count"
)

// Usage 统一计量结果：各维度的实际消耗量。
// 各厂商 usage 字段由对应 Codec 归一化进此 map。
type Usage struct {
	Metrics map[BillingMetric]float64 `json:"metrics"`
}

// CostDetail 单个维度的费用明细。
type CostDetail struct {
	Metric   BillingMetric `json:"metric"`
	Quantity float64       `json:"quantity"`
	Amount   float64       `json:"amount"`
}

// Cost 费用。币种按原币种存，不做汇率换算。
type Cost struct {
	Details  []CostDetail `json:"details"`
	Total    float64      `json:"total"`
	Currency string       `json:"currency"`
}

// ChatResponse 统一对话响应（非流式）。
type ChatResponse struct {
	ID string `json:"id"`
	// Model 业务方请求的逻辑模型名。
	Model string `json:"model"`
	// ActualChannelModel Failover 最终命中的候选标识（如 channel_model_id 或可读名）。
	ActualChannelModel string         `json:"actualChannelModel"`
	Content            []ContentBlock `json:"content"`
	StopReason         string         `json:"stopReason"`
	Usage              Usage          `json:"usage"`
	Cost               Cost           `json:"cost"`
}
