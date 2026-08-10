package overlays

import (
	"strings"
	"testing"
)

func TestMCPPanel_Render(t *testing.T) {
	mp := NewMCPPanel()
	mp.Servers = []MCPEntry{
		{Name: "filesystem", URL: "file:///tmp", Status: "connected"},
	}
	mp.Width = 80
	result := mp.Render()
	if !strings.Contains(result, "MCP 服务器") {
		t.Error("渲染结果应包含标题 'MCP 服务器'")
	}
	if !strings.Contains(result, "filesystem") {
		t.Error("渲染结果应包含 filesystem 服务器")
	}
}

func TestMCPPanel_CursorNavigation(t *testing.T) {
	mp := NewMCPPanel()
	mp.Servers = []MCPEntry{{Name: "a"}, {Name: "b"}}
	mp.CursorDown()
	if mp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	mp.CursorUp()
	if mp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestMCPPanel_EditMode(t *testing.T) {
	mp := NewMCPPanel()
	mp.StartEditing()
	if !mp.Editing {
		t.Error("StartEditing() 后 Editing 应为 true")
	}
	if mp.EditBuffer != "" {
		t.Error("StartEditing() 后 EditBuffer 应为空")
	}

	mp.EditBuffer = "myserver http://localhost:8080"
	id, ep := mp.ConfirmEditing()
	if id != "myserver" {
		t.Errorf("ConfirmEditing() server_id 应为 myserver, got %s", id)
	}
	if ep != "http://localhost:8080" {
		t.Errorf("ConfirmEditing() endpoint 应为 http://localhost:8080, got %s", ep)
	}
	if mp.Editing {
		t.Error("ConfirmEditing() 后 Editing 应为 false")
	}
}

func TestMCPPanel_EditModeSingleArg(t *testing.T) {
	mp := NewMCPPanel()
	mp.StartEditing()
	mp.EditBuffer = "onlyid"
	id, ep := mp.ConfirmEditing()
	if id != "onlyid" {
		t.Errorf("server_id 应为 onlyid, got %s", id)
	}
	if ep != "" {
		t.Errorf("endpoint 应为空, got %s", ep)
	}
}

func TestMCPPanel_CancelEditing(t *testing.T) {
	mp := NewMCPPanel()
	mp.StartEditing()
	mp.EditBuffer = "test"
	mp.CancelEditing()
	if mp.Editing {
		t.Error("CancelEditing() 后 Editing 应为 false")
	}
	if mp.EditBuffer != "" {
		t.Error("CancelEditing() 后 EditBuffer 应为空")
	}
}

func TestMCPPanel_RenderEditing(t *testing.T) {
	mp := NewMCPPanel()
	mp.Width = 80
	mp.StartEditing()
	mp.EditBuffer = "srv http://example.com"
	result := mp.Render()
	if !strings.Contains(result, "srv") {
		t.Error("渲染结果应包含编辑缓冲区内容")
	}
	if !strings.Contains(result, "确认添加") {
		t.Error("渲染结果应包含确认提示")
	}
}
