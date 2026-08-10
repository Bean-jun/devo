package overlays

import (
	"strings"
	"testing"
)

func TestWorkspacePanel_Render(t *testing.T) {
	wp := NewWorkspacePanel()
	wp.Workspaces = []WorkspaceEntry{
		{Name: "my-project", Path: "/tmp", Active: true},
	}
	wp.Width = 80
	result := wp.Render()
	if !strings.Contains(result, "工作区") {
		t.Error("渲染结果应包含标题 '工作区'")
	}
	if !strings.Contains(result, "my-project") {
		t.Error("渲染结果应包含工作区 my-project")
	}
}

func TestWorkspacePanel_CursorNavigation(t *testing.T) {
	wp := NewWorkspacePanel()
	wp.Workspaces = []WorkspaceEntry{{Name: "a"}, {Name: "b"}}
	wp.CursorDown()
	if wp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	wp.CursorUp()
	if wp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}
