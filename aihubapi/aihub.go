// Package aihubapi 定义 aihub 模块对外的统一调用契约。
//
// 职责：
//   - IAIHub：业务方（进程内或其他模块）调用 ai-hub 的唯一接口
//
// 边界：
//   - 只定义契约，不含实现（实现在 api/client）
//   - 依赖 api/dto 的统一协议类型，不引用 internal/
package aihubapi

import (
	"context"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

// IAIHub AI 调用中心对外契约。
type IAIHub interface {
	// Chat 非流式对话。
	Chat(ctx context.Context, req *dto.ChatRequest) (*dto.ChatResponse, error)
	// ChatStream 流式对话，返回统一事件流（业务方消费转发）。
	ChatStream(ctx context.Context, req *dto.ChatRequest) (<-chan dto.StreamEvent, error)
	// Embedding 同步文本向量化。
	Embedding(ctx context.Context, req *dto.EmbeddingRequest) (*dto.EmbeddingResponse, error)
	// Rerank 同步文本重排序。
	Rerank(ctx context.Context, req *dto.RerankRequest) (*dto.RerankResponse, error)
}

// IAIHubMedia AI 多模态调用对外契约（图片/视频/语音）。
//
// 边界：
//   - 只定义契约，不含实现（实现在 api/client）
//   - 同步生成（图片等）直接返回结果；异步生成（视频等）返回 jobId 由业务方轮询（后续 plan）
type IAIHubMedia interface {
	// GenerateImage 同步图片生成，返回已转存为永久 OSS 引用的产物。
	GenerateImage(ctx context.Context, req *dto.ImageRequest) (*dto.ImageResult, error)
	// CreateVoice 多渠道容灾创建逻辑音色，返回逐渠道成败明细。
	CreateVoice(ctx context.Context, req *dto.CreateVoiceRequest) (*dto.CreateVoiceResult, error)
	// DeleteVoice 删除逻辑音色及其全部厂商绑定。
	DeleteVoice(ctx context.Context, logicalVoiceID int64) error
	// GenerateSpeech 同步 TTS 合成，返回永久 OSS 音频引用。
	GenerateSpeech(ctx context.Context, req *dto.SpeechRequest) (*dto.SpeechResult, error)
	// Transcribe 同步 ASR 识别，返回文本。
	Transcribe(ctx context.Context, req *dto.TranscribeRequest) (*dto.TranscribeResult, error)
	// SubmitVideoJob 提交异步视频生成任务，返回业务 jobID。
	SubmitVideoJob(ctx context.Context, req *dto.VideoJobRequest) (jobID string, err error)
	// GetJob 查询异步任务状态与结果（业务方轮询）。
	GetJob(ctx context.Context, jobID string) (*dto.MediaJobResult, error)
}
