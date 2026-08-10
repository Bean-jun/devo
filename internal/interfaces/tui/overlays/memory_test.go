package overlays

import (
	"strings"
	"testing"
)

func TestMemoryPanel_Render(t *testing.T) {
	mp := NewMemoryPanel()
	mp.Memories = []MemoryEntry{
		{Key: "user_pref", Content: "偏好设置", Type: "user"},
	}
	mp.Width = 80
	result := mp.Render()
	if !strings.Contains(result, "记忆管理") {
		t.Error("渲染结果应包含标题 '记忆管理'")
	}
	if !strings.Contains(result, "user_pref") {
		t.Error("渲染结果应包含 user_pref")
	}
	if !strings.Contains(result, "[user]") {
		t.Error("渲染结果应包含类型标签 [user]")
	}
}

func TestMemoryPanel_CursorNavigation(t *testing.T) {
	mp := NewMemoryPanel()
	mp.Memories = []MemoryEntry{{Key: "a", Type: "user"}, {Key: "b", Type: "project"}}
	mp.CursorDown()
	if mp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	mp.CursorUp()
	if mp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestMemoryPanel_EditMode(t *testing.T) {
	mp := NewMemoryPanel()
	mp.StartEditing()
	if !mp.Editing {
		t.Error("StartEditing() 后 Editing 应为 true")
	}
	if mp.EditBuffer != "" {
		t.Error("StartEditing() 后 EditBuffer 应为空")
	}

	mp.EditBuffer = "mykey my content here"
	key, content := mp.ConfirmEditing()
	if key != "mykey" {
		t.Errorf("key 应为 mykey, got %s", key)
	}
	if content != "my content here" {
		t.Errorf("content 应为 my content here, got %s", content)
	}
	if mp.Editing {
		t.Error("ConfirmEditing() 后 Editing 应为 false")
	}
}

func TestMemoryPanel_EditModeSingleArg(t *testing.T) {
	mp := NewMemoryPanel()
	mp.StartEditing()
	mp.EditBuffer = "onlykey"
	key, content := mp.ConfirmEditing()
	if key != "onlykey" {
		t.Errorf("key 应为 onlykey, got %s", key)
	}
	if content != "" {
		t.Errorf("content 应为空, got %s", content)
	}
}

func TestMemoryPanel_CancelEditing(t *testing.T) {
	mp := NewMemoryPanel()
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

func TestMemoryPanel_RenderEditing(t *testing.T) {
	mp := NewMemoryPanel()
	mp.Width = 80
	mp.StartEditing()
	mp.EditBuffer = "mykey my content"
	result := mp.Render()
	if !strings.Contains(result, "mykey") {
		t.Error("渲染结果应包含编辑缓冲区内容")
	}
	if !strings.Contains(result, "确认添加") {
		t.Error("渲染结果应包含确认提示")
	}
}
