package overlays

import (
	"strings"
	"testing"
)

func TestCommandSheet_New(t *testing.T) {
	cs := NewCommandSheet()
	if len(cs.FlatCommands) == 0 {
		t.Error("CommandSheet 应有命令列表")
	}
	if len(cs.Groups) == 0 {
		t.Error("CommandSheet 应有分组")
	}
}

func TestCommandSheet_CursorNavigation(t *testing.T) {
	cs := NewCommandSheet()
	cs.CursorDown()
	if cs.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	cs.CursorUp()
	if cs.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestCommandSheet_CursorUpBoundary(t *testing.T) {
	cs := NewCommandSheet()
	cs.CursorUp()
	if cs.Selected != 0 {
		t.Error("在顶部 CursorUp() 不应超出边界")
	}
}

func TestCommandSheet_CursorDownBoundary(t *testing.T) {
	cs := NewCommandSheet()
	for i := 0; i < len(cs.FlatCommands)+10; i++ {
		cs.CursorDown()
	}
	if cs.Selected >= len(cs.FlatCommands) {
		t.Error("CursorDown() 不应超出列表范围")
	}
}

func TestCommandSheet_SelectedCommand(t *testing.T) {
	cs := NewCommandSheet()
	cmd := cs.SelectedCommand()
	if cmd.Name != "/new" {
		t.Errorf("首个命令应为 /new, got %s", cmd.Name)
	}
}

func TestCommandSheet_CursorDownSelect(t *testing.T) {
	cs := NewCommandSheet()
	cs.CursorDown()
	cmd := cs.SelectedCommand()
	if cmd.Name != "/switch" {
		t.Errorf("第二个命令应为 /switch, got %s", cmd.Name)
	}
}

func TestCommandSheet_Render(t *testing.T) {
	cs := NewCommandSheet()
	cs.Width = 80
	result := cs.Render()
	if !strings.Contains(result, "命令面板") {
		t.Error("渲染结果应包含标题 '命令面板'")
	}
	if !strings.Contains(result, "▸new") {
		t.Error("渲染结果应包含选中命令 ▸new")
	}
	if !strings.Contains(result, "/help") {
		t.Error("渲染结果应包含 /help 命令")
	}
}
