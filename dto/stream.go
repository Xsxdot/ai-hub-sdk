// Package dto 的流式事件部分：定义 ai-hub SSE 输出的统一事件契约。
//
// 职责：
//   - 提供 ChatStream 与 server 流式转发共用的事件类型和负载结构
//   - 让业务方只依赖中立 StreamEvent，不感知厂商原始流帧
//
// 边界：
//   - 纯数据类型，无网络读取和业务编排
//   - 不引用 internal/model，不引用任何厂商 SDK 类型
package dto

// EventType 流事件类型。Codec 把各厂商五花八门的流帧归一化为这些统一事件。
// 上层（转发/聚合/记录）只认 StreamEvent，不感知厂商。
type EventType string

const (
	EventMessageStart      EventType = "message_start"
	EventContentBlockStart EventType = "content_block_start"
	EventContentBlockDelta EventType = "content_block_delta"
	EventContentBlockStop  EventType = "content_block_stop"
	EventUsage             EventType = "usage"
	EventMessageStop       EventType = "message_stop"
	EventError             EventType = "error"
)

// MessageStartData message_start 负载。
type MessageStartData struct {
	ID                 string `json:"id"`
	ActualChannelModel string `json:"actualChannelModel"`
}

// ContentBlockStartData content_block_start 负载。
type ContentBlockStartData struct {
	Index     int       `json:"index"`
	BlockType BlockType `json:"blockType"`
}

// ContentBlockDeltaData content_block_delta 负载（文本/思维链/工具入参增量）。
type ContentBlockDeltaData struct {
	Index int    `json:"index"`
	Delta string `json:"delta"`
}

// ContentBlockStopData content_block_stop 负载。
type ContentBlockStopData struct {
	Index int `json:"index"`
}

// MessageStopData message_stop 负载。
type MessageStopData struct {
	StopReason string `json:"stopReason"`
}

// StreamErrorData error 负载。
type StreamErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// StreamEvent 统一流事件。按 Type 取用对应的可空负载字段。
// 这是 Codec.DecodeStreamChunk 的输出契约，也是聚合/转发/记录的统一输入。
type StreamEvent struct {
	Type EventType `json:"type"`

	MessageStart      *MessageStartData      `json:"messageStart,omitempty"`
	ContentBlockStart *ContentBlockStartData `json:"contentBlockStart,omitempty"`
	ContentBlockDelta *ContentBlockDeltaData `json:"contentBlockDelta,omitempty"`
	ContentBlockStop  *ContentBlockStopData  `json:"contentBlockStop,omitempty"`
	Usage             *Usage                 `json:"usage,omitempty"`
	Cost              *Cost                  `json:"cost,omitempty"`
	MessageStop       *MessageStopData       `json:"messageStop,omitempty"`
	Error             *StreamErrorData       `json:"error,omitempty"`
}
