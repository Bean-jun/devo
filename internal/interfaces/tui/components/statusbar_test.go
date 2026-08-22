package components

import (
	"strings"
	"testing"
)

func TestStatusBar_Render(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Session = "test-session"
	sb.ServerPort = "8080"

	result := sb.Render()
	if !strings.Contains(result, "test-session") {
		t.Error("渲染结果应包含会话名称")
	}
	if !strings.Contains(result, "idle") {
		t.Error("渲染结果应包含状态 idle")
	}
	if !strings.Contains(result, "─") {
		t.Error("渲染结果应包含分隔线")
	}
	if !strings.Contains(result, ":8080") {
		t.Error("渲染结果应包含端口号 :8080")
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

func TestStatusBar_TeamMode(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.TeamMode = true

	result := sb.Render()
	if !strings.Contains(result, "TEAM") {
		t.Error("Team Mode 应显示 TEAM 标记")
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

func TestStatusBar_ReasoningEnabled(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ReasoningEnabled = true
	sb.ReasoningEffort = "medium"

	result := sb.Render()
	if !strings.Contains(result, "medium") {
		t.Error("启用思维链时应显示 effort 等级")
	}
}

func TestStatusBar_ReasoningDisabled(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ReasoningEnabled = false

	result := sb.Render()
	if !strings.Contains(result, "off") {
		t.Error("关闭思维链时应显示 off")
	}
}

func TestStatusBar_ReasoningEffortLow(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ReasoningEnabled = true
	sb.ReasoningEffort = "low"

	result := sb.Render()
	if !strings.Contains(result, "low") {
		t.Error("启用思维链低强度时应显示 low")
	}
}

func TestStatusBar_ReasoningEffortHigh(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ReasoningEnabled = true
	sb.ReasoningEffort = "high"

	result := sb.Render()
	if !strings.Contains(result, "high") {
		t.Error("启用思维链高强度时应显示 high")
	}
}

func TestStatusBar_ActivityShowsStreamingContent(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ActivityActive = true
	sb.Activity = "hello world"

	result := sb.Render()
	if !strings.Contains(result, "hello world") {
		t.Error("ActivityActive 时应显示流式内容")
	}
}

func TestStatusBar_ActivityActiveOverridesProcessing(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Processing = true
	sb.ActivityActive = true
	sb.Activity = "streaming token"

	result := sb.Render()
	if strings.Contains(result, "Processing...") {
		t.Error("ActivityActive 时不应显示 Processing...")
	}
	if !strings.Contains(result, "streaming token") {
		t.Error("ActivityActive 时应显示流式内容而非 Processing...")
	}
}

func TestStatusBar_ActivityInactiveShowsProcessing(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.Processing = true
	sb.ActivityActive = false

	result := sb.Render()
	if !strings.Contains(result, "Processing...") {
		t.Error("非 ActivityActive 且 Processing 时应显示 Processing...")
	}
}

func TestStatusBar_ActivityEmpty(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ActivityActive = true
	sb.Activity = ""

	result := sb.Render()
	if strings.Contains(result, "⏳") {
		t.Error("Activity 为空且 ActivityActive 时不应显示 spinner")
	}
}

func TestStatusBar_ActivityWithReasoning(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ActivityActive = true
	sb.Activity = "思考中: 我们来分析一下这个问题"

	result := sb.Render()
	if !strings.Contains(result, "思考中: 我们来分析一下这个问题") {
		t.Error("推理活动应显示推理前缀")
	}
}

func TestStatusBar_ActivityCleared(t *testing.T) {
	sb := NewStatusBar()
	sb.Width = 80
	sb.ActivityActive = true
	sb.Activity = "streaming content"

	sb.ActivityActive = false
	sb.Activity = ""

	result := sb.Render()
	if strings.Contains(result, "streaming content") {
		t.Error("清除 Activity 后不应显示流式内容")
	}
}
