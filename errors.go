// Package aihubsdk 提供调用 ai-hub HTTP 服务的 Go 客户端。
//
// 职责：
//   - 封装 ai-hub /v1/* 公开接口（鉴权、剥壳、SSE 流式解析）
//   - Client 实现 aihubapi.IAIHub / IAIHubMedia，与进程内调用同契约
//
// 边界：
//   - 不含业务编排（容灾/计费在 server 端）
//   - 零重依赖：仅 stdlib net/http + encoding/json
package aihubsdk

import "fmt"

// APIError 表示 ai-hub 返回的业务级错误（非 2xx 或剥壳后 status != 200）。
//
// 字段：
//   - Status: HTTP 状态码或响应壳中的 status
//   - Message: 服务端返回的错误描述
type APIError struct {
	Status  int
	Message string
}

// Error 实现 error 接口。
func (e *APIError) Error() string {
	return fmt.Sprintf("aihub: status=%d message=%s", e.Status, e.Message)
}
