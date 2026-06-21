// Package dto 的 OCR 部分：定义 OCR 的中立请求/响应契约。
//
// 职责：
//   - 定义 OCR 的统一请求与响应形状
//   - 内置任务、提示词、抽取模板为强类型中立线内字段；厂商特有像素/旋转参数走 Options 逃生舱
//
// 边界：
//   - 不含任何厂商协议细节（DashScope multimodal-generation 形状在 server 端 Codec 处理）
//   - 不含网络、OSS 解析与计费逻辑
package dto

// OcrTask OCR 内置任务的强类型枚举（中立线内）。
type OcrTask string

const (
	OcrTaskTextRecognition     OcrTask = "text_recognition"           // 通用文字识别 -> 纯文本
	OcrTaskAdvancedRecognition OcrTask = "advanced_recognition"       // 高精识别（文字+坐标定位）
	OcrTaskKeyInformation      OcrTask = "key_information_extraction" // 信息抽取 -> kv JSON
	OcrTaskTableParsing        OcrTask = "table_parsing"              // 表格解析 -> HTML
	OcrTaskDocumentParsing     OcrTask = "document_parsing"           // 文档解析 -> LaTeX
	OcrTaskFormulaRecognition  OcrTask = "formula_recognition"        // 公式识别 -> LaTeX
	OcrTaskMultiLanguage       OcrTask = "multi_lan"                  // 多语言识别 -> 纯文本
)

// OcrRequest OCR 请求（中立契约）。
type OcrRequest struct {
	Model        string         `json:"model"`                  // 逻辑模型名
	Image        *MediaRef      `json:"image"`                  // 公网 URL 或 ai-hub 发放的 ossKey
	Task         OcrTask        `json:"task,omitempty"`         // 内置任务；为空且无 Prompt 时厂商用默认提示词
	Prompt       string         `json:"prompt,omitempty"`       // 自定义提示词（与 Task 可并存）
	ResultSchema map[string]any `json:"resultSchema,omitempty"` // 仅 key_information_extraction：自定义抽取字段模板
	Options      map[string]any `json:"options,omitempty"`      // min_pixels/max_pixels/enable_rotate 等逃生舱
}

// OcrResult OCR 结果（中立契约）。
type OcrResult struct {
	ID                 string         `json:"id"`
	Model              string         `json:"model"`
	ActualChannelModel string         `json:"actualChannelModel"`
	Text               string         `json:"text"`                 // 主输出，厂商 content.text 原样（含 markdown 围栏）
	Structured         map[string]any `json:"structured,omitempty"` // 结构化透传：厂商 ocr_result 原样（words_info / kv_result）
	Usage              Usage          `json:"usage"`
	Cost               Cost           `json:"cost"`
}
