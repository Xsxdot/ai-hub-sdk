// main.go 提供 ai-hub SDK 的真实环境 smoke 示例命令。
//
// 职责：
//   - 使用 SDK 连接本地 ai-hub HTTP 服务
//   - 顺序测试 chat、chat stream、image、video、asr 四类公开能力
//   - 记录每一步的开始、成功摘要和错误上下文
//
// 边界：
//   - 不保存 API Key，不写入数据库，不管理 ai-hub 服务进程
//   - 不等待视频终态，除非调用方显式设置 AIHUB_WAIT_VIDEO=true
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	aihubsdk "github.com/xsxdot/ai-hub-sdk"
	"github.com/xsxdot/ai-hub-sdk/dto"
)

type smokeStep struct {
	name string
	run  func(context.Context) error
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := LoadConfig(os.Getenv)
	if err != nil {
		log.Error("load config failed", "error", err)
		os.Exit(2)
	}

	log.Info("ai-hub realtest starting",
		"baseURL", cfg.BaseURL,
		"chatModel", cfg.ChatModel,
		"imageModel", cfg.ImageModel,
		"videoModel", cfg.VideoModel,
		"asrModel", cfg.ASRModel,
		"runStream", cfg.RunStream,
		"waitVideo", cfg.WaitVideo,
	)

	if err := runSmoke(context.Background(), cfg, log); err != nil {
		log.Error("ai-hub realtest completed with failures", "error", err)
		os.Exit(1)
	}
	log.Info("ai-hub realtest completed successfully")
}

func runSmoke(ctx context.Context, cfg Config, log *slog.Logger) error {
	client := aihubsdk.New(
		aihubsdk.WithBaseURL(cfg.BaseURL),
		aihubsdk.WithAPIKey(cfg.APIKey),
		aihubsdk.WithTimeout(cfg.HTTPTimeout),
	)

	steps := []smokeStep{
		{name: "chat", run: func(ctx context.Context) error { return runChat(ctx, client, cfg, log) }},
		{name: "chat_stream", run: func(ctx context.Context) error { return runChatStream(ctx, client, cfg, log) }},
		{name: "image", run: func(ctx context.Context) error { return runImage(ctx, client, cfg, log) }},
		{name: "video", run: func(ctx context.Context) error { return runVideo(ctx, client, cfg, log) }},
		{name: "asr", run: func(ctx context.Context) error { return runASR(ctx, client, cfg, log) }},
	}
	return runSteps(ctx, log, steps)
}

func runSteps(ctx context.Context, log *slog.Logger, steps []smokeStep) error {
	var errs []error
	for _, step := range steps {
		log.Info("smoke step started", "step", step.name)
		start := time.Now()
		if err := step.run(ctx); err != nil {
			log.Error("smoke step failed", "step", step.name, "duration", time.Since(start), "error", err)
			errs = append(errs, err)
			continue
		}
		log.Info("smoke step succeeded", "step", step.name, "duration", time.Since(start))
	}
	return errors.Join(errs...)
}

func runChat(ctx context.Context, client *aihubsdk.Client, cfg Config, log *slog.Logger) error {
	resp, err := client.Chat(ctx, &dto.ChatRequest{
		Model:     cfg.ChatModel,
		MaxTokens: 128,
		Messages: []dto.Message{{
			Role: dto.RoleUser,
			Content: []dto.ContentBlock{{
				Type: dto.BlockText,
				Text: cfg.ChatPrompt,
			}},
		}},
	})
	if err != nil {
		return err
	}
	log.Info("chat response received",
		"id", resp.ID,
		"model", resp.Model,
		"actualChannelModel", resp.ActualChannelModel,
		"stopReason", resp.StopReason,
		"contentBlocks", len(resp.Content),
	)
	return nil
}

func runChatStream(ctx context.Context, client *aihubsdk.Client, cfg Config, log *slog.Logger) error {
	if !cfg.RunStream {
		log.Info("chat stream skipped", "reason", "set AIHUB_RUN_STREAM=true to enable")
		return nil
	}
	stream, err := client.ChatStream(ctx, &dto.ChatRequest{
		Model:     cfg.ChatModel,
		MaxTokens: 128,
		Messages: []dto.Message{{
			Role: dto.RoleUser,
			Content: []dto.ContentBlock{{
				Type: dto.BlockText,
				Text: cfg.ChatPrompt,
			}},
		}},
	})
	if err != nil {
		return err
	}

	events := 0
	var deltas []string
	for ev := range stream {
		events++
		if ev.ContentBlockDelta != nil && ev.ContentBlockDelta.Delta != "" {
			deltas = append(deltas, ev.ContentBlockDelta.Delta)
		}
	}
	log.Info("chat stream completed",
		"events", events,
		"deltaPreview", textPreview(strings.Join(deltas, ""), 120),
	)
	return nil
}

func runImage(ctx context.Context, client *aihubsdk.Client, cfg Config, log *slog.Logger) error {
	res, err := client.GenerateImage(ctx, &dto.ImageRequest{
		Model:  cfg.ImageModel,
		Prompt: cfg.ImagePrompt,
		N:      1,
	})
	if err != nil {
		return err
	}
	attrs := []any{
		"id", res.ID,
		"model", res.Model,
		"actualChannelModel", res.ActualChannelModel,
		"artifactCount", len(res.Artifacts),
	}
	for i, artifact := range res.Artifacts {
		if i >= cfg.ImageArtifactMax {
			break
		}
		attrs = append(attrs, "artifactOssKey", artifact.OSSKey, "artifactMediaType", artifact.MediaType)
	}
	log.Info("image generated", attrs...)
	return nil
}

func runVideo(ctx context.Context, client *aihubsdk.Client, cfg Config, log *slog.Logger) error {
	jobID, err := client.SubmitVideoJob(ctx, &dto.VideoJobRequest{
		Model:    cfg.VideoModel,
		Task:     dto.VideoTaskText2Video,
		Prompt:   cfg.VideoPrompt,
		Duration: 5,
	})
	if err != nil {
		return err
	}
	log.Info("video job submitted", "jobID", jobID)

	if !cfg.WaitVideo {
		return pollVideoFixed(ctx, client, cfg, log, jobID)
	}
	return waitVideoTerminal(ctx, client, cfg, log, jobID)
}

func pollVideoFixed(ctx context.Context, client *aihubsdk.Client, cfg Config, log *slog.Logger, jobID string) error {
	for i := 0; i < cfg.VideoInitialPolls; i++ {
		res, err := client.GetJob(ctx, jobID)
		if err != nil {
			return err
		}
		logVideoState(log, "video job polled", res)
	}
	return nil
}

func waitVideoTerminal(ctx context.Context, client *aihubsdk.Client, cfg Config, log *slog.Logger, jobID string) error {
	timeout := cfg.VideoTimeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(cfg.VideoPollEvery)
	defer ticker.Stop()

	for {
		res, err := client.GetJob(ctx, jobID)
		if err != nil {
			return err
		}
		logVideoState(log, "video job polled", res)
		if res.State == dto.JobStateSucceeded {
			return nil
		}
		if res.State == dto.JobStateFailed {
			return errors.New("video job failed: " + res.Error)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func logVideoState(log *slog.Logger, message string, res *dto.MediaJobResult) {
	log.Info(message,
		"jobID", res.JobID,
		"state", res.State,
		"artifactCount", len(res.Artifacts),
		"error", res.Error,
	)
}

func runASR(ctx context.Context, client *aihubsdk.Client, cfg Config, log *slog.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, cfg.TranscribeTimeout)
	defer cancel()

	res, err := client.Transcribe(ctx, &dto.TranscribeRequest{
		Model: cfg.ASRModel,
		Audio: mediaRefPtr(dto.URLMediaRef(cfg.AudioURL, "audio/mpeg")),
	})
	if err != nil {
		return err
	}
	log.Info("asr transcription completed",
		"id", res.ID,
		"model", res.Model,
		"actualChannelModel", res.ActualChannelModel,
		"textChars", utf8.RuneCountInString(res.Text),
		"textPreview", textPreview(res.Text, 120),
	)
	return nil
}

func mediaRefPtr(ref dto.MediaRef) *dto.MediaRef {
	return &ref
}

func textPreview(text string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}
