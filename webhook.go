// webhook.go 提供 ai-hub 视频任务回调的接收侧验签 helper。
//
// 职责：
//   - VerifyCallback：用提交任务时的同一把 apiKey 复算 HMAC 签名并常量时间比对，
//     同时校验 timestamp 时效（防重放）
//
// 边界：
//   - 不解析 body 业务字段（验签通过后由调用方自行反序列化为 dto.MediaJobResult）
//   - 算法必须与 server callback_notifier.go 完全一致
package aihubsdk

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// VerifyCallback 校验 ai-hub 视频回调签名与时效。
//
// 参数：
//   - apiKey: 提交该视频任务时使用的同一把 API Key
//   - signatureHeader: 回调请求头 X-AIHub-Signature（形如 "sha256=<hex>"）
//   - timestampHeader: 回调请求头 X-AIHub-Timestamp（unix 秒）
//   - body: 回调原始请求体字节（验签前不要改动/重序列化）
//   - maxAge: 允许的最大时钟偏移；timestamp 距今超过此值视为重放，<=0 时跳过时效校验
//
// 返回：
//   - nil 表示验签通过；否则返回描述性错误（签名不符 / 时间戳非法 / 已过期）
//
// 注意：
//   - 算法与 server 一致：expected = "sha256=" + hex(HMAC-SHA256(secret, ts + "." + body))，
//     secret = hex(sha256(apiKey))。用 hmac.Equal 常量时间比对防时序攻击。
func VerifyCallback(apiKey, signatureHeader, timestampHeader string, body []byte, maxAge time.Duration) error {
	tsUnix, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return fmt.Errorf("aihub: invalid callback timestamp %q: %w", timestampHeader, err)
	}
	if maxAge > 0 {
		age := time.Since(time.Unix(tsUnix, 0))
		if age < 0 {
			// 容忍业务方时钟略快于 ai-hub，使用绝对偏移判断是否超窗。
			age = -age
		}
		if age > maxAge {
			return fmt.Errorf("aihub: callback timestamp expired (age=%s > maxAge=%s)", age, maxAge)
		}
	}

	sum := sha256.Sum256([]byte(apiKey))
	secret := hex.EncodeToString(sum[:])
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestampHeader + "." + string(body)))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signatureHeader)) {
		return fmt.Errorf("aihub: callback signature mismatch")
	}
	return nil
}
