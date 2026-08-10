package overlays

import (
	"strings"
	"testing"
)

func TestNewSessionModal_Render(t *testing.T) {
	nsm := NewSessionModal{Width: 80}
	result := nsm.Render()
	if !strings.Contains(result, "新建会话") {
		t.Error("渲染结果应包含标题 '新建会话'")
	}
}
