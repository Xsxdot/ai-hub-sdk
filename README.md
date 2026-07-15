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

- `UploadMedia` / `GenerateImage` / `GenerateSpeech`：返回永久 OSS 身份和临时公网 URL。
- `CreateVoice` / `DeleteVoice`：逻辑音色管理。
- `SubmitImageJob` / `SubmitVideoJob` 返回 jobID；`GetJob(jobID)` 通过统一 `/v1/media/jobs/{jobId}` 轮询直到 `state == succeeded/failed`。

### 统一媒体产物

图片、视频和音频统一使用 `dto.MediaArtifact` 的四个字段：

```json
{
  "ossKey": "media/image/20260617/example.png",
  "url": "https://public.example.com/media/image/20260617/example.png?signature=...",
  "urlExpiresAt": 1784347200000,
  "mediaType": "image/png"
}
```

- `OSSKey` 是永久身份和 URL 刷新依据，应长期保存。
- `URL` 是可轮换的临时公网地址，默认约 7 天有效，不可作为身份或缓存键。
- `URLExpiresAt` 是 Unix 毫秒时间戳；到期前或到期后使用 `ResolveMedia` 刷新。
- 图片、视频、音频的访问地址都只叫 `URL`。

```go
fresh, err := c.ResolveMedia(ctx, &dto.ResolveMediaRequest{
	OSSKey:    savedOSSKey,
	MediaType: "image/png",
})
if err != nil {
	return err
}
fmt.Println(fresh.URL, fresh.URLExpiresAt)
```

生成、上传、轮询或回调中的单项 URL 签名失败不会重跑模型、重复上传、重复计费或改变任务终态；结果仍保留 `OSSKey` 和 `MediaType`，`URL` 为空、`URLExpiresAt` 为 `0`，稍后可调用 `ResolveMedia` 恢复。专门的刷新调用签名失败时会直接返回错误。

兼容窗口内，`SpeechResult.AudioOssKey` 已 Deprecated 且与 `SpeechResult.OSSKey` 同值；`VoiceBindingResult.PreviewOssKey`、`PreviewMediaType` 已 Deprecated，且分别与规范 `OSSKey`、`MediaType` 同值。新代码只读取规范字段。

### 上传、同步图片、TTS 与音色预览

```go
uploaded, err := c.UploadMedia(ctx, aihubsdk.UploadMediaKindImage, "poster.png", imageReader)
if err != nil {
	return err
}
fmt.Println(uploaded.OSSKey, uploaded.URL, uploaded.URLExpiresAt)

image, err := c.GenerateImage(ctx, &dto.ImageRequest{Model: "image-pro", Prompt: "poster"})
if err != nil {
	return err
}
fmt.Println(image.Artifacts[0].OSSKey, image.Artifacts[0].URL)

speech, err := c.GenerateSpeech(ctx, &dto.SpeechRequest{Voice: "narrator", Text: "hello"})
if err != nil {
	return err
}
fmt.Println(speech.OSSKey, speech.URL, speech.URLExpiresAt)

voice, err := c.CreateVoice(ctx, &dto.CreateVoiceRequest{
	Name:        "warm-narrator",
	Source:      dto.VoiceSourceDesign,
	VoicePrompt: "warm and natural",
	PreviewText: "hello",
})
if err != nil {
	return err
}
if len(voice.Succeeded) > 0 {
	preview := voice.Succeeded[0]
	fmt.Println(preview.OSSKey, preview.URL, preview.URLExpiresAt)
}
```

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
if err != nil {
	return err
}
if job.State == dto.JobStateSucceeded && len(job.Artifacts) > 0 {
	fmt.Println(job.Artifacts[0].OSSKey, job.Artifacts[0].URL, job.Artifacts[0].URLExpiresAt)
}
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
	if result.State == dto.JobStateSucceeded && len(result.Artifacts) > 0 {
		artifact := result.Artifacts[0]
		fmt.Println(artifact.OSSKey, artifact.URL, artifact.URLExpiresAt)
	}
	w.WriteHeader(http.StatusOK)
}
```

轮询和回调使用同一媒体产物字段。每次回调尝试都会重新投影 URL，因此重试时 URL 字符串可能不同；业务身份始终以 `OSSKey` 为准。

## 错误处理

业务错误为 `*aihubsdk.APIError`（含 `Status` / `Message`）；网络错误原样透出。

```go
var apiErr *aihubsdk.APIError
if errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized {
	// 鉴权失败
}
```

## 发布顺序

依赖发布顺序为：gokit 显式公网 signer → ai-hub-sdk 统一契约 → AI-HUB server。正式发布、构建和部署不得依赖指向本机目录的绝对 `replace`；本地 workspace/临时 `replace` 只用于联调。
