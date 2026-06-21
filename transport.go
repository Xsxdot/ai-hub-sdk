// transport.go 实现 SDK 的 HTTP 编解码与 SSE 帧解析。
//
// 职责：
//   - doJSON：非流式请求；自动注入 X-API-Key；剥 gokit result.OK 壳取 data
//   - doSSE：流式请求；按 SSE "data: {json}\n\n" 逐帧解析为 dto.StreamEvent
//
// 边界：
//   - 不含业务语义，仅做协议级编解码
//   - 网络/ctx 错误原样透出；业务级错误包成 *APIError
package aihubsdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xsxdot/ai-hub-sdk/dto"
)

// resultShell 对应 gokit result.OK 的统一响应壳：{"status":200,"data":...}。
// 错误时形如 {"status":401,"message":"..."}。
type resultShell struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

const (
	traceContextKey     = "traceId"
	traceHeaderName     = "Trace-Head"
	traceHeaderXTraceID = "X-Trace-Id"
)

// newRequest 构造带鉴权头的请求。body 为 nil 时不带 body（用于 GET/DELETE）。
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	setTraceHeaders(req, ctx)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func setTraceHeaders(req *http.Request, ctx context.Context) {
	if req == nil || ctx == nil {
		return
	}
	traceID, ok := ctx.Value(traceContextKey).(string)
	traceID = strings.TrimSpace(traceID)
	if !ok || traceID == "" {
		return
	}
	if req.Header.Get(traceHeaderName) == "" {
		req.Header.Set(traceHeaderName, traceID)
	}
	if req.Header.Get(traceHeaderXTraceID) == "" {
		req.Header.Set(traceHeaderXTraceID, traceID)
	}
}

// doJSON 执行非流式请求，剥 result 壳后把 data 反序列化到 out。
//
// 参数：
//   - out: 目标 dto 指针；为 nil 时只校验成功不解码（如 DeleteVoice）
//
// 返回：
//   - status != 200（壳内）或 HTTP 非 2xx → *APIError；网络错误原样透出
func (c *Client) doJSON(ctx context.Context, method, path string, body, out any) error {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	return c.doRequest(req, out)
}

// doRequest 执行已构造好的 HTTP 请求，剥 result 壳后把 data 反序列化到 out。
//
// 参数：
//   - req: 已设置鉴权、Content-Type 等头的 HTTP 请求
//   - out: 目标 dto 指针；为 nil 时只校验成功不解码
//
// 返回：
//   - status != 200（壳内）或 HTTP 非 2xx → *APIError；网络错误原样透出
func (c *Client) doRequest(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var shell resultShell
	if uErr := json.Unmarshal(raw, &shell); uErr != nil {
		// 非预期壳（如网关返回纯文本错误页）：用 HTTP 状态码兜底为 APIError。
		if resp.StatusCode/100 != 2 {
			return &APIError{Status: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
		}
		return fmt.Errorf("decode result shell: %w", uErr)
	}

	if shell.Status != 200 {
		status := shell.Status
		if status == 0 {
			status = resp.StatusCode
		}
		return &APIError{Status: status, Message: shell.Message}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(shell.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

// doSSE 执行流式请求，返回逐帧投递的 StreamEvent channel。
//
// 行为：
//   - HTTP 非 2xx → 不创建事件流，直接返回 *APIError
//   - 正常：goroutine 按 "data: " 行切帧，json 反序列化为 StreamEvent 投递
//   - ctx 取消会断开请求、停止推送并关闭 channel
//   - 解析失败的帧跳过，不中断整流
func (c *Client) doSSE(ctx context.Context, path string, body any) (<-chan dto.StreamEvent, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var shell resultShell
		if json.Unmarshal(raw, &shell) == nil && shell.Status != 0 {
			return nil, &APIError{Status: shell.Status, Message: shell.Message}
		}
		return nil, &APIError{Status: resp.StatusCode, Message: strings.TrimSpace(string(raw))}
	}

	out := make(chan dto.StreamEvent)
	go func() {
		defer close(out)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		// SSE 单帧可能较大（含 base64），放宽行缓冲上限到 1MB。
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			payload := strings.TrimPrefix(line, "data: ")
			var ev dto.StreamEvent
			if json.Unmarshal([]byte(payload), &ev) != nil {
				continue
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
