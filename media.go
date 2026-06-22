// media.go 实现 IAIHubMedia 契约：图片/语音/ASR/OCR/音色/视频异步任务。
//
// 职责：
//   - 对齐 server/system/aihub/router.go 暴露的 /v1/* 多模态 HTTP 路由
//   - 将 result.OK 响应壳交给 transport 层剥离，返回统一 dto 结果
//
// 边界：
//   - 不执行媒体业务编排、轮询、回调处理或重试
//   - 不理解厂商私有参数，仅透传 dto 中的中立请求结构
package aihubsdk

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

type mediaJobSubmitResponse struct {
	JobID string `json:"jobId"`
}

// GenerateImage 同步图片生成。POST /v1/images/generate。
func (c *Client) GenerateImage(ctx context.Context, req *dto.ImageRequest) (*dto.ImageResult, error) {
	var res dto.ImageResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/images/generate", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SubmitImageJob 提交异步图片任务，返回业务 jobID。POST /v1/images/jobs。
func (c *Client) SubmitImageJob(ctx context.Context, req *dto.ImageJobRequest) (string, error) {
	var res mediaJobSubmitResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/images/jobs", req, &res); err != nil {
		return "", err
	}
	return res.JobID, nil
}

// CreateVoice 多渠道容灾创建逻辑音色。POST /v1/voices。
func (c *Client) CreateVoice(ctx context.Context, req *dto.CreateVoiceRequest) (*dto.CreateVoiceResult, error) {
	var res dto.CreateVoiceResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/voices", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// DeleteVoice 删除逻辑音色及其全部厂商绑定。DELETE /v1/voices/:id。
func (c *Client) DeleteVoice(ctx context.Context, logicalVoiceID int64) error {
	path := fmt.Sprintf("/v1/voices/%d", logicalVoiceID)
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// GenerateSpeech 同步 TTS 合成。POST /v1/speech/generate。
func (c *Client) GenerateSpeech(ctx context.Context, req *dto.SpeechRequest) (*dto.SpeechResult, error) {
	var res dto.SpeechResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/speech/generate", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Transcribe 同步 ASR 识别。POST /v1/audio/transcriptions。
func (c *Client) Transcribe(ctx context.Context, req *dto.TranscribeRequest) (*dto.TranscribeResult, error) {
	var res dto.TranscribeResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/audio/transcriptions", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Ocr 同步 OCR 识别。POST /v1/ocr。
func (c *Client) Ocr(ctx context.Context, req *dto.OcrRequest) (*dto.OcrResult, error) {
	var res dto.OcrResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/ocr", req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SubmitVideoJob 提交异步视频任务，返回业务 jobID。POST /v1/videos/jobs。
func (c *Client) SubmitVideoJob(ctx context.Context, req *dto.VideoJobRequest) (string, error) {
	var res mediaJobSubmitResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/videos/jobs", req, &res); err != nil {
		return "", err
	}
	return res.JobID, nil
}

// GetJob 查询异步媒体任务状态与结果。GET /v1/media/jobs/:jobId。
func (c *Client) GetJob(ctx context.Context, jobID string) (*dto.MediaJobResult, error) {
	path := fmt.Sprintf("/v1/media/jobs/%s", jobID)
	var res dto.MediaJobResult
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
