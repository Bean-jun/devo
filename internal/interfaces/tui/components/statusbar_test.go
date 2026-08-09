package components

import (
	"strings"
	"testing"
)

func TestStatusBar_Render(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Session = "test-session"

	result := sb.Render()
	if !strings.Contains(result, "Devo") {
		t.Error("渲染结果应包含应用名称 Devo")
	}
	if !strings.Contains(result, "test-session") {
		t.Error("渲染结果应包含会话名称")
	}
	if !strings.Contains(result, "idle") {
		t.Error("渲染结果应包含状态 idle")
	}
	if !strings.Contains(result, "─") {
		t.Error("渲染结果应包含分隔线")
	}
}

func TestStatusBar_Processing(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Processing = true

	result := sb.Render()
	if !strings.Contains(result, "Processing") {
		t.Error("处理中状态应显示 Processing")
	}
}

func TestStatusBar_Paused(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Paused = true

	result := sb.Render()
	if !strings.Contains(result, "Paused") {
		t.Error("暂停状态应显示 Paused")
	}
}

func TestStatusBar_Yolo(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Yolo = true

	result := sb.Render()
	if !strings.Contains(result, "YOLO") {
		t.Error("YOLO 模式应显示 YOLO 标记")
	}
}

func TestStatusBar_Disconnected(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Connected = false

	result := sb.Render()
	if !strings.Contains(result, "✗") {
		t.Error("断开连接状态应显示 ✗")
	}
}

func TestStatusBar_MinWidth(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 5

	result := sb.Render()
	if result == "" {
		t.Error("即使宽度很小也应能渲染")
	}
}
