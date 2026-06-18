// runner_test.go 验证 realtest 示例命令的步骤编排 helper。
//
// 职责：
//   - 确认单个模态失败不会阻止后续模态继续测试
//   - 覆盖日志摘要中使用的文本截断逻辑
//
// 边界：
//   - 不创建真实 SDK Client，不访问 ai-hub 服务
//   - 不输出测试日志到标准输出
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
)

// TestRunStepsContinuesAfterFailure 验证失败聚合时仍会继续执行后续步骤。
func TestRunStepsContinuesAfterFailure(t *testing.T) {
	var calls []string
	steps := []smokeStep{
		{name: "first", run: func(context.Context) error {
			calls = append(calls, "first")
			return errors.New("boom")
		}},
		{name: "second", run: func(context.Context) error {
			calls = append(calls, "second")
			return nil
		}},
	}

	err := runSteps(context.Background(), discardLogger(), steps)
	if err == nil {
		t.Fatal("want aggregate error, got nil")
	}
	if !reflect.DeepEqual(calls, []string{"first", "second"}) {
		t.Fatalf("calls = %v", calls)
	}
}

// TestTextPreviewTruncatesLongText 验证长文本预览会按 rune 数截断。
func TestTextPreviewTruncatesLongText(t *testing.T) {
	got := textPreview("abcdefghijklmnopqrstuvwxyz", 10)
	if got != "abcdefghij..." {
		t.Fatalf("preview = %q", got)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
