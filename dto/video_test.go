// Package dto 测试视频任务 DTO 的中立协议字段。
//
// 职责：
//   - 验证 VideoJobRequest 的任务类型与语义素材槽 JSON 往返
//   - 验证 VideoTask 枚举值非空，避免协议常量被误删
//
// 边界：
//   - 不测试 provider codec 的厂商字段映射
//   - 不引用 internal/model
package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestVideoJobRequestJSONRoundTrip(t *testing.T) {
	in := VideoJobRequest{
		Model:       "smart-video",
		Task:        VideoTaskFirstLastFrame,
		Prompt:      "cat yawns",
		FirstFrame:  mediaRefPtr(URLMediaRef("https://x/first.png", "image/png")),
		LastFrame:   mediaRefPtr(URLMediaRef("https://x/last.png", "image/png")),
		RefImages:   []MediaRef{URLMediaRef("https://x/r1.png", "image/png")},
		RefVideos:   []MediaRef{URLMediaRef("https://x/v1.mp4", "video/mp4")},
		SourceVideo: mediaRefPtr(OSSKeyMediaRef("ai-hub/public-media/video/src.mp4", "video/mp4")),
		RefAudios:   []MediaRef{URLMediaRef("https://x/a1.mp3", "audio/mpeg")},
		Options:     map[string]any{"duration": float64(10)},
		Metadata:    map[string]string{"tier": "pro"},
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out VideoJobRequest
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Task != VideoTaskFirstLastFrame {
		t.Fatalf("task = %q, want first_last_frame", out.Task)
	}
	if out.FirstFrame == nil || out.FirstFrame.URL != "https://x/first.png" || out.LastFrame == nil || out.LastFrame.URL != "https://x/last.png" {
		t.Fatalf("frame slots not preserved: %+v", out)
	}
	if len(out.RefImages) != 1 || len(out.RefVideos) != 1 || len(out.RefAudios) != 1 {
		t.Fatalf("ref slots not preserved: %+v", out)
	}
	if out.SourceVideo == nil || out.SourceVideo.OSSKey != "ai-hub/public-media/video/src.mp4" {
		t.Fatalf("source video slot not preserved: %+v", out)
	}
}

func TestVideoJobRequestCoreParamsJSON(t *testing.T) {
	req := VideoJobRequest{
		Model:       "text2video-standard",
		Task:        VideoTaskText2Video,
		Prompt:      "a cat",
		AspectRatio: AspectRatio16x9,
		Resolution:  Resolution1080p,
		Duration:    5,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back VideoJobRequest
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.AspectRatio != AspectRatio16x9 || back.Resolution != Resolution1080p || back.Duration != 5 {
		t.Fatalf("core params not round-tripped: %+v", back)
	}
}

func TestVideoTaskValues(t *testing.T) {
	want := []VideoTask{
		VideoTaskText2Video, VideoTaskImage2Video, VideoTaskFirstLastFrame,
		VideoTaskRefImage2Video, VideoTaskRefVideo2Video, VideoTaskVideoEdit,
	}
	for _, task := range want {
		if string(task) == "" {
			t.Fatalf("task value empty")
		}
	}
}

func TestVideoCoreParamValidators(t *testing.T) {
	if !IsValidAspectRatio(string(AspectRatio16x9)) {
		t.Fatal("expected 16:9 aspect ratio valid")
	}
	if IsValidAspectRatio("4:5") {
		t.Fatal("expected 4:5 aspect ratio invalid")
	}
	if !IsValidResolution(string(Resolution1080p)) {
		t.Fatal("expected 1080p resolution valid")
	}
	if IsValidResolution("99p") {
		t.Fatal("expected 99p resolution invalid")
	}
}

func TestMediaArtifactJSONUsesUnifiedDeliveryFields(t *testing.T) {
	artifact := MediaArtifact{
		OSSKey:       "ai-hub/public-media/video/out.mp4",
		URL:          "https://public.example.com/out.mp4?token=secret",
		URLExpiresAt: 1784073600000,
		MediaType:    "video/mp4",
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) != `{"ossKey":"ai-hub/public-media/video/out.mp4","url":"https://public.example.com/out.mp4?token=secret","urlExpiresAt":1784073600000,"mediaType":"video/mp4"}` {
		t.Fatalf("json=%s", raw)
	}

	// 图片产物必须是同一契约的别名，避免两份定义后续再次漂移。
	var imageArtifact MediaArtifact = ImageArtifact(artifact)
	if imageArtifact.URL != artifact.URL {
		t.Fatalf("image artifact url=%q, want %q", imageArtifact.URL, artifact.URL)
	}
}

func TestVideoJobRequestAutoAdjustRoundTrip(t *testing.T) {
	req := VideoJobRequest{Model: "video", Task: VideoTaskFirstLastFrame, Duration: 1, AutoAdjust: true}
	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"auto_adjust":true`) {
		t.Fatalf("auto_adjust 未序列化: %s", raw)
	}
	var back VideoJobRequest
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !back.AutoAdjust {
		t.Fatalf("AutoAdjust 回环丢失")
	}
}

func TestVideoSubmitResultAdjustment(t *testing.T) {
	res := VideoSubmitResult{
		JobID:              "job-1",
		DurationAdjustment: &DurationAdjustment{RequestedDuration: 1, ActualDuration: 4, Reason: "below_model_minimum"},
	}
	raw, _ := json.Marshal(res)
	if !strings.Contains(string(raw), `"durationAdjustment"`) {
		t.Fatalf("durationAdjustment 缺失: %s", raw)
	}
}
