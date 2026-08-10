package overlays

import (
	"strings"
	"testing"
)

func TestRollbackPicker_Render(t *testing.T) {
	items := []RollbackItem{
		{Content: "第一条消息", Role: "用户", Time: "12:00"},
		{Content: "第二条消息", Role: "助手", Time: "12:01"},
	}
	rp := NewRollbackPicker(items)
	rp.Width = 80
	result := rp.Render()
	if !strings.Contains(result, "回滚") {
		t.Error("渲染结果应包含标题 '回滚'")
	}
	if !strings.Contains(result, "第一条消息") {
		t.Error("渲染结果应包含消息内容")
	}
}

func TestRollbackPicker_CursorNavigation(t *testing.T) {
	items := []RollbackItem{
		{Content: "a", Role: "用户", Time: "12:00"},
		{Content: "b", Role: "助手", Time: "12:01"},
	}
	rp := NewRollbackPicker(items)
	rp.CursorUp()
	if rp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}
