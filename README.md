# ai-hub-sdk

ai-hub 的 Go 客户端：跨进程 HTTP 调用，封装鉴权、SSE 流式、结果剥壳。

## 安装

```bash
go get github.com/xsxdot/ai-hub-sdk
```

## 快速上手

```go
import (
	"context"

	aihubsdk "github.com/xsxdot/ai-hub-sdk"
	"github.com/xsxdot/ai-hub-sdk/dto"
)

c := aihubsdk.New(
	aihubsdk.WithBaseURL("https://aihub.internal"),
	aihubsdk.WithAPIKey("YOUR_API_KEY"),
)

resp, err := c.Chat(context.Background(), &dto.ChatRequest{
	Model: "your-logical-model",
	Messages: []dto.Message{{
		Role: dto.RoleUser,
		Content: []dto.ContentBlock{{
			Type: dto.BlockText,
			Text: "你好",
		}},
	}},
})
```

## 流式

```go
ch, err := c.ChatStream(ctx, &dto.ChatRequest{Model: "m", Messages: msgs})
if err != nil {
	// handle error
}
for ev := range ch {
	switch ev.Type {
	case dto.EventContentBlockDelta:
		fmt.Print(ev.ContentBlockDelta.Delta)
	case dto.EventMessageStop:
		// 结束
	}
}
// ctx 取消即断流；channel 关闭表示流结束。
```

## 多模态

- `GenerateImage` / `GenerateSpeech` / `Transcribe`：同步返回（产物为永久 OSS 引用）。
- `CreateVoice` / `DeleteVoice`：逻辑音色管理。
- `SubmitImageJob` / `SubmitVideoJob` 返回 jobID；`GetJob(jobID)` 通过统一 `/v1/media/jobs/{jobId}` 轮询直到 `state == succeeded/failed`。

### 异步图片

```go
jobID, err := c.SubmitImageJob(ctx, &dto.ImageJobRequest{
	ImageRequest: dto.ImageRequest{
		Model:  "image-pro",
		Prompt: "poster",
		N:      1,
	},
	CallbackURL: "https://your.app/aihub/callback",
})
if err != nil {
	return err
}
job, err := c.GetJob(ctx, jobID)
```

## 异步任务回调

图片或视频提交时填 `CallbackURL` 即启用回调（不填则用 `GetJob` 轮询）。签名密钥由 ai-hub 从你的 API Key 派生，你无需传任何密钥：

```go
jobID, err := c.SubmitVideoJob(ctx, &dto.VideoJobRequest{
	Model:       "v",
	Task:        dto.VideoTaskText2Video,
	Prompt:      "...",
	CallbackURL: "https://your.app/aihub/callback",
})
```

任务终态时 ai-hub 向该地址 POST `MediaJobResult` JSON，并带头 `X-AIHub-Signature` / `X-AIHub-Timestamp` / `X-AIHub-Job-Id`。接收端验签：

```go
func handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	err := aihubsdk.VerifyCallback(
		"YOUR_API_KEY",
		r.Header.Get("X-AIHub-Signature"),
		r.Header.Get("X-AIHub-Timestamp"),
		body,
		5*time.Minute,
	)
	if err != nil {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}

	var result dto.MediaJobResult
	_ = json.Unmarshal(body, &result)
	// 处理 result.State / result.Artifacts ...
	w.WriteHeader(http.StatusOK)
}
```

## 错误处理

业务错误为 `*aihubsdk.APIError`（含 `Status` / `Message`）；网络错误原样透出。

```go
var apiErr *aihubsdk.APIError
if errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized {
	// 鉴权失败
}
```
