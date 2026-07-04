package components

import (
	"strings"
	"testing"
)

func TestInputArea_New(t *testing.T) {
	ia := NewInputArea()
	if ia.textarea.Placeholder == "" {
		t.Error("placeholder should not be empty")
	}
}

func TestInputArea_Focus(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	if !ia.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestInputArea_Blur(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.Blur()
	if ia.Focused() {
		t.Error("should not be focused after Blur()")
	}
}

func TestInputArea_Value(t *testing.T) {
	ia := NewInputArea()
	ia.SetValue("hello")
	if ia.Value() != "hello" {
		t.Errorf("expected 'hello', got '%s'", ia.Value())
	}
}

func TestInputArea_Reset(t *testing.T) {
	ia := NewInputArea()
	ia.SetValue("hello")
	ia.Reset()
	if ia.Value() != "" {
		t.Errorf("expected empty after Reset, got '%s'", ia.Value())
	}
}

func TestInputArea_View(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	view := ia.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestInputArea_FooterShowsContextUsage(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	ia.ContextUsage = "context 12.5K"
	view := ia.View()
	if !strings.Contains(view, "context 12.5K") {
		t.Error("view should contain context usage")
	}
}

func TestInputArea_FooterShowsTokenUsage(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	ia.TokenUsage = "Tokens ↑1.2K ↓3.4K (4.6K)"
	view := ia.View()
	if !strings.Contains(view, "Tokens") {
		t.Error("view should contain token usage")
	}
}

func TestInputArea_FooterShowsWorkingDir(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	ia.WorkingDir = "/home/user/project"
	view := ia.View()
	if !strings.Contains(view, "/home/user/project") {
		t.Error("view should contain working directory")
	}
}

func TestInputArea_IsPasteActive(t *testing.T) {
	ia := NewInputArea()
	if ia.IsPasteActive() {
		t.Error("paste should not be active initially")
	}
}
