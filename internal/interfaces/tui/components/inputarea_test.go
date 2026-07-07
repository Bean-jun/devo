package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestInputArea_PasteMessage(t *testing.T) {
	ia := NewInputArea()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h', 'e', 'l', 'l', 'o'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "hello" {
		t.Errorf("expected 'hello', got '%s'", ia.Value())
	}
}

func TestInputArea_BulkRunesInsert(t *testing.T) {
	ia := NewInputArea()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a', 'b', 'c', 'd'},
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "abcd" {
		t.Errorf("expected 'abcd', got '%s'", ia.Value())
	}
}

func TestInputArea_SingleRune(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'x'},
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "x" {
		t.Errorf("expected 'x', got '%s'", ia.Value())
	}
}

func TestInputArea_Backspace(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.SetValue("abc")
	ia.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if ia.Value() != "ab" {
		t.Errorf("expected 'ab', got '%s'", ia.Value())
	}
}

func TestInputArea_NormalInputNotFiltered(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h', 'e', 'l', 'l', 'o'},
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "hello" {
		t.Errorf("expected 'hello', got '%s'", ia.Value())
	}
}

func TestInputArea_CtrlN_InsertsNewline(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.SetValue("line1")
	msg := tea.KeyMsg{Type: tea.KeyCtrlN}
	_, _ = ia.Update(msg)
	if !strings.Contains(ia.Value(), "\n") {
		t.Error("Ctrl+N should insert a newline")
	}
	if ia.textarea.LineCount() < 2 {
		t.Errorf("Ctrl+N should create at least 2 lines, got %d", ia.textarea.LineCount())
	}
}

func TestInputArea_PasteNormalizesCRLF(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h', 'e', 'l', 'l', 'o', '\r', '\n', 'w', 'o', 'r', 'l', 'd'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if strings.Contains(ia.Value(), "\r") {
		t.Error("pasted text should not contain \\r")
	}
	if !strings.Contains(ia.Value(), "hello\nworld") {
		t.Errorf("expected 'hello\\nworld', got '%s'", ia.Value())
	}
}

func TestInputArea_PasteNormalizesCR(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h', 'e', 'l', 'l', 'o', '\r', 'w', 'o', 'r', 'l', 'd'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if strings.Contains(ia.Value(), "\r") {
		t.Error("pasted text should not contain \\r")
	}
	if !strings.Contains(ia.Value(), "hello\nworld") {
		t.Errorf("expected 'hello\\nworld', got '%s'", ia.Value())
	}
}

func TestInputArea_BulkRunesNormalizesCRLF(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	runes := []rune{'a', 'b', '\r', '\n', 'c', 'd'}
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: runes,
	}
	_, _ = ia.Update(msg)
	if strings.Contains(ia.Value(), "\r") {
		t.Error("bulk runes should not contain \\r")
	}
	if !strings.Contains(ia.Value(), "ab\ncd") {
		t.Errorf("expected 'ab\\ncd', got '%s'", ia.Value())
	}
}

func TestNormalizePastedText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "crlf", input: "hello\r\nworld", expected: "hello\nworld"},
		{name: "cr only", input: "hello\rworld", expected: "hello\nworld"},
		{name: "trailing newline", input: "hello\n", expected: "hello"},
		{name: "normal text", input: "hello world", expected: "hello world"},
		{name: "mixed", input: "line1\r\nline2\rline3", expected: "line1\nline2\nline3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePastedText(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
