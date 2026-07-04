package components

import (
	"strings"
	"testing"
)

func TestHelpPanel_Renders(t *testing.T) {
	hp := NewHelpPanel()
	hp.SetSize(80, 24)
	hp.Show()

	view := hp.View()
	if !strings.Contains(view, "快捷键") {
		t.Error("help panel should show shortcut section")
	}
	if !strings.Contains(view, "命令") {
		t.Error("help panel should show command section")
	}
	if !strings.Contains(view, "Esc") {
		t.Error("help panel should mention Esc")
	}
	if !strings.Contains(view, "Alt+Y") {
		t.Error("help panel should mention Alt+Y")
	}
	if !strings.Contains(view, "/yolo") {
		t.Error("help panel should mention /yolo")
	}
	if !strings.Contains(view, "/trust") {
		t.Error("help panel should mention /trust")
	}
}

func TestHelpPanel_ShowHide(t *testing.T) {
	hp := NewHelpPanel()
	hp.SetSize(80, 24)

	if hp.Visible {
		t.Error("help panel should be hidden by default")
	}

	hp.Show()
	if !hp.Visible {
		t.Error("help panel should be visible after Show()")
	}
	if hp.View() == "" {
		t.Error("help panel should render content when visible")
	}

	hp.Hide()
	if hp.Visible {
		t.Error("help panel should be hidden after Hide()")
	}
	if hp.View() != "" {
		t.Error("help panel should render nothing when hidden")
	}
}

func TestHelpPanel_ContainsAllCommands(t *testing.T) {
	hp := NewHelpPanel()
	hp.SetSize(80, 24)
	hp.Show()

	view := hp.View()

	expectedCmds := []string{
		"/new", "/switch", "/rename", "/export", "/rollback",
		"/pause", "/resume", "/cancel", "/yolo", "/trust",
		"/help", "/quit",
	}

	for _, cmd := range expectedCmds {
		if !strings.Contains(view, cmd) {
			t.Errorf("help panel should contain command: %s", cmd)
		}
	}
}
