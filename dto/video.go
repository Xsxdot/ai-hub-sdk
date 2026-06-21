// Package dto 的视频/异步任务部分：定义视频生成与异步 job 的统一中立契约。
//
// 职责：
//   - 提供业务方与 ai-hub 之间厂商无关的视频生成请求和 job 查询结果契约
//   - 统一异步任务状态与永久 OSS 产物引用
//
// 边界：
//   - 纯数据类型，无业务方法
//   - 不引用 internal/model，不引用任何厂商 SDK 类型
package dto

// VideoTask 视频生成任务类型，由业务方显式指定。
//
// 设计要点：
//   - 各厂商对任务类型的表达高度不一致（火山靠 content[].role 组合、万相靠
//     media[].type 白名单、veo 靠 model 名后缀），ai-hub 把它统一为显式上位概念，
//     由各 Codec 翻译回厂商隐式编码。
type VideoTask string

const (
	VideoTaskText2Video     VideoTask = "text2video"       // 仅 prompt
	VideoTaskImage2Video    VideoTask = "image2video"      // 首帧生视频
	VideoTaskFirstLastFrame VideoTask = "first_last_frame" // 首尾帧生视频
	VideoTaskRefImage2Video VideoTask = "ref_image2video"  // 参考图生视频
	VideoTaskRefVideo2Video VideoTask = "ref_video2video"  // 参考视频生视频
	VideoTaskVideoEdit      VideoTask = "video_edit"       // 视频编辑
)

// AspectRatio 中立宽高比枚举，规范形态为约分后的 "W:H"。
// 各 Codec 据此翻译为厂商参数（万相 ratio 字符串、火山隐式、Sora width/height）。
type AspectRatio string

const (
	AspectRatio16x9 AspectRatio = "16:9"
	AspectRatio9x16 AspectRatio = "9:16"
	AspectRatio1x1  AspectRatio = "1:1"
	AspectRatio4x3  AspectRatio = "4:3"
	AspectRatio3x4  AspectRatio = "3:4"
	AspectRatio21x9 AspectRatio = "21:9"
)

// IsValidAspectRatio 报告 v 是否为 ai-hub 支持的中立宽高比枚举。
//
// 参数：
//   - v: 待校验的宽高比字符串
//
// 返回：
//   - true 表示可以作为 VideoCaps.AspectRatios 的声明值
func IsValidAspectRatio(v string) bool {
	switch AspectRatio(v) {
	case AspectRatio16x9, AspectRatio9x16, AspectRatio1x1, AspectRatio4x3, AspectRatio3x4, AspectRatio21x9:
		return true
	default:
		return false
	}
}

// Resolution 中立分辨率枚举，规范形态为纵向像素档 "{N}p"。
//
// 语义锚定：p = 输出视频的垂直像素数（高）；宽 = 高 × 宽高比，由 Codec 据
// 当前 AspectRatio 推出。厂商若要求尺寸为某倍数或离散档位，由该 Codec snap
// 到最近合法尺寸，但中立锚点（垂直像素 = 档位数值）不变。
type Resolution string

const (
	Resolution480p  Resolution = "480p"
	Resolution720p  Resolution = "720p"
	Resolution1080p Resolution = "1080p"
	Resolution1440p Resolution = "1440p"
	Resolution2160p Resolution = "2160p"
)

// IsValidResolution 报告 v 是否为 ai-hub 支持的中立分辨率档位。
//
// 参数：
//   - v: 待校验的分辨率字符串
//
// 返回：
//   - true 表示可以作为 VideoCaps.Resolutions 的声明值
func IsValidResolution(v string) bool {
	switch Resolution(v) {
	case Resolution480p, Resolution720p, Resolution1080p, Resolution1440p, Resolution2160p:
		return true
	default:
		return false
	}
}

// VideoJobRequest 视频生成提交请求。
//
// 素材按语义分槽，不再共用一个 RefImages：同一字段在不同任务下含义不同会让 Codec
// 无法可靠区分（首帧 vs 参考图 vs 尾帧）。核心调参（ratio/resolution/duration）
// 走强类型字段；Options 仅承载厂商私有 frontier 参数，为明标逃生舱。
type VideoJobRequest struct {
	Model  string    `json:"model"`
	Task   VideoTask `json:"task"`
	Prompt string    `json:"prompt"`

	// 语义素材槽。各 task 只填用到的槽，校验器据此核对数量。
	FirstFrame  *MediaRef  `json:"firstFrame,omitempty"`  // image2video / first_last_frame
	LastFrame   *MediaRef  `json:"lastFrame,omitempty"`   // first_last_frame
	RefImages   []MediaRef `json:"refImages,omitempty"`   // ref_image2video / video_edit 参考图
	RefVideos   []MediaRef `json:"refVideos,omitempty"`   // ref_video2video 参考视频
	SourceVideo *MediaRef  `json:"sourceVideo,omitempty"` // video_edit 待编辑视频
	RefAudios   []MediaRef `json:"refAudios,omitempty"`   // 驱动/参考音频

	// 核心中立调参（强类型枚举，跨厂商契约保证）。
	AspectRatio AspectRatio `json:"aspectRatio,omitempty"` // 约分 W:H，见 AspectRatio
	Resolution  Resolution  `json:"resolution,omitempty"`  // {N}p，p=垂直像素，见 Resolution
	Duration    int         `json:"duration,omitempty"`    // 整数秒

	// 逃生舱（escape hatch，非协议契约）。
	// 注意：使用 Options 即离开中立保证；厂商私有、best-effort、逻辑模型下不推荐、
	// 跨 failover 不保证一致，可能被某 Codec 静默忽略。已知 key 见
	// docs/aihub-public-api.md。核心调参一律走上方强类型字段，不走这里。
	Options  map[string]any    `json:"options,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// CallbackURL 业务方回调地址，空=不回调。任务终态时 ai-hub 主动 POST 结果到此。
	CallbackURL string `json:"callbackUrl,omitempty"`
	// callbackSecret 由 HTTP 层从 API Key 派生写入，不来自客户端 JSON（小写不导出 + json:"-"）。
	// 设为非导出字段，确保业务方无法伪造签名密钥。
	callbackSecret string `json:"-"`
}

// SetCallbackSecret 由 HTTP 层注入回调签名密钥（API Key 派生值）。
//
// 参数：
//   - secret: hex(sha256(apiKey))；置于请求上，随提交透传到 MediaJob。
func (r *VideoJobRequest) SetCallbackSecret(secret string) { r.callbackSecret = secret }

// CallbackSecret 返回注入的回调签名密钥，供 app 层落库使用。
func (r *VideoJobRequest) CallbackSecret() string { return r.callbackSecret }

// JobState 异步任务中性状态。
type JobState string

const (
	JobStateSubmitting JobState = "submitting"
	JobStateQueued     JobState = "queued"
	JobStateRunning    JobState = "running"
	JobStateSucceeded  JobState = "succeeded"
	JobStateFailed     JobState = "failed"
)

// MediaArtifact 永久 OSS 产物引用。
type MediaArtifact struct {
	OSSKey    string `json:"ossKey"`
	MediaType string `json:"mediaType"`
}

// MediaJobResult 业务方查询的 job 结果。
type MediaJobResult struct {
	JobID              string          `json:"jobId"`
	Modality           string          `json:"modality"`
	Model              string          `json:"model"`
	ActualChannelModel string          `json:"actualChannelModel"`
	State              JobState        `json:"state"`
	Artifacts          []MediaArtifact `json:"artifacts"`
	Usage              Usage           `json:"usage"`
	Cost               Cost            `json:"cost"`
	Error              string          `json:"error,omitempty"`
}
