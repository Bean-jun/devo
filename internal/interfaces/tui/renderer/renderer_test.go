package renderer

import (
	"strings"
	"testing"

	"devo/internal/interfaces/tui/types"
)

func TestWrapCJK_English(t *testing.T) {
	text := "Hello World this is a test"
	wrapped := WrapCJK(text, 10)
	lines := strings.Split(wrapped, "\n")
	if len(lines) < 3 {
		t.Errorf("长英文文本应换行, got %d lines", len(lines))
	}
}

func TestWrapCJK_Chinese(t *testing.T) {
	text := "这是一段很长的中文文本需要测试换行功能"
	wrapped := WrapCJK(text, 10)
	lines := strings.Split(wrapped, "\n")
	if len(lines) < 3 {
		t.Errorf("长中文文本应换行, got %d lines", len(lines))
	}
}

func TestWrapCJK_Mixed(t *testing.T) {
	text := "Hello世界 这是一个ABC测试"
	wrapped := WrapCJK(text, 10)
	if wrapped == "" {
		t.Error("换行结果不应为空")
	}
}

func TestWrapCJK_Newline(t *testing.T) {
	text := "line1\nline2\nline3"
	wrapped := WrapCJK(text, 80)
	lines := strings.Split(wrapped, "\n")
	if len(lines) != 3 {
		t.Errorf("应保留换行符, 预期 3 行, got %d", len(lines))
	}
}

func TestWrapCJK_ShortText(t *testing.T) {
	text := "short"
	wrapped := WrapCJK(text, 80)
	if wrapped != "short" {
		t.Errorf("短文本不应换行, got %s", wrapped)
	}
}

func TestWrapCJK_ZeroWidth(t *testing.T) {
	text := "some text"
	wrapped := WrapCJK(text, 0)
	if wrapped != text {
		t.Error("width 为 0 时应返回原文本")
	}
}

func TestNewRenderer(t *testing.T) {
	r := New(80)
	if r == nil {
		t.Error("New() 不应返回 nil")
	}
	if r.width != 80 {
		t.Errorf("width 应为 80, got %d", r.width)
	}
}

func TestRenderer_SetWidth(t *testing.T) {
	r := New(80)
	r.SetWidth(120)
	if r.width != 120 {
		t.Errorf("SetWidth(120) 后 width 应为 120, got %d", r.width)
	}
}

func TestRenderer_Render(t *testing.T) {
	r := New(80)
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "World"},
	}
	result := r.Render(messages)
	if !strings.Contains(result, "Hello") {
		t.Error("渲染结果应包含用户消息")
	}
	if !strings.Contains(result, "World") {
		t.Error("渲染结果应包含助手消息")
	}
}

func TestRenderer_RenderSystem(t *testing.T) {
	r := New(80)
	messages := []types.Message{
		{Role: "system", Content: "System notice"},
	}
	result := r.Render(messages)
	if !strings.Contains(result, "System notice") {
		t.Error("渲染结果应包含系统消息")
	}
}

func TestRenderer_MessageCountInCache(t *testing.T) {
	r := New(80)
	if r.MessageCountInCache() != 0 {
		t.Error("初始缓存应为 0")
	}
	r.Render([]types.Message{
		{Role: "user", Content: "msg1"},
		{Role: "user", Content: "msg2"},
	})
	if r.MessageCountInCache() != 2 {
		t.Errorf("缓存应有 2 条消息, got %d", r.MessageCountInCache())
	}
}

func TestRenderer_GetCacheLineCount(t *testing.T) {
	r := New(80)
	r.Render([]types.Message{
		{Role: "user", Content: "test"},
	})
	if r.GetCacheLineCount(0) <= 0 {
		t.Error("渲染后的消息应有行数")
	}
}

func TestRenderer_FindUserMessageYOffsets(t *testing.T) {
	r := New(80)
	messages := []types.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "How are you?"},
	}
	r.Render(messages)
	offsets := r.FindUserMessageYOffsets(messages)
	if len(offsets) != 2 {
		t.Errorf("应有 2 个用户消息偏移量, got %d", len(offsets))
	}
}

func TestRenderer_CacheReuse(t *testing.T) {
	r := New(80)
	messages := []types.Message{
		{Role: "user", Content: "msg1"},
	}
	r.Render(messages)
	first := r.Render(messages)
	second := r.Render(messages)
	if first != second {
		t.Error("相同消息应使用缓存，两次渲染结果应相同")
	}
}
