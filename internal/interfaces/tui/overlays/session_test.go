package overlays

import (
	"strings"
	"testing"

	"devo/internal/interfaces/tui/types"
)

func TestSessionPicker_Render(t *testing.T) {
	sp := SessionPicker{
		Width:    80,
		Selected: 0,
		Sessions: []types.SessionInfo{
			{ID: "1", Title: "test-session", MessageCount: 5},
		},
	}
	result := sp.Render()
	if !strings.Contains(result, "切换会话") {
		t.Error("渲染结果应包含标题 '切换会话'")
	}
	if !strings.Contains(result, "test-session") {
		t.Error("渲染结果应包含会话名称")
	}
}

func TestSessionPicker_CursorNavigation(t *testing.T) {
	sp := SessionPicker{
		Sessions: []types.SessionInfo{
			{ID: "1", Title: "a"},
			{ID: "2", Title: "b"},
			{ID: "3", Title: "c"},
		},
	}
	sp.CursorDown()
	if sp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	sp.CursorUp()
	if sp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}
