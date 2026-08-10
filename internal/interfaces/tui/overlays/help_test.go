package overlays

import (
	"strings"
	"testing"
)

func TestHelpPanel_Render(t *testing.T) {
	hp := HelpPanel{Width: 80}
	result := hp.Render()
	if !strings.Contains(result, "Help") {
		t.Error("渲染结果应包含标题 Help")
	}
	if !strings.Contains(result, "Ctrl+C") {
		t.Error("渲染结果应包含 Ctrl+C 快捷键")
	}
	if !strings.Contains(result, "Enter") {
		t.Error("渲染结果应包含 Enter 快捷键")
	}
}

func TestHelpPanel_MinWidth(t *testing.T) {
	hp := HelpPanel{Width: 10}
	result := hp.Render()
	if result == "" {
		t.Error("即使宽度很小也应能渲染")
	}
}
