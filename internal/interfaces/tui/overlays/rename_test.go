package overlays

import (
	"strings"
	"testing"
)

func TestRenameModal_Render(t *testing.T) {
	rm := RenameModal{Current: "demo-session", NewName: "新名称", Width: 80}
	result := rm.Render()
	if !strings.Contains(result, "重命名") {
		t.Error("渲染结果应包含标题 '重命名'")
	}
	if !strings.Contains(result, "demo-session") {
		t.Error("渲染结果应包含当前名称")
	}
	if !strings.Contains(result, "新名称") {
		t.Error("渲染结果应包含新名称")
	}
}
