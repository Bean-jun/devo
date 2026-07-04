package components

import (
	"strings"
	"testing"
)

func TestInputArea_CharCount(t *testing.T) {
	ia := NewInputArea()
	ia.SetMaxChars(5000)
	ia.UpdateCharCount("hello")
	ia.Width = 80

	if ia.CharCount != 5 {
		t.Errorf("expected CharCount 5, got %d", ia.CharCount)
	}

	view := ia.View()
	if !strings.Contains(view, "5/5000") {
		t.Error("view should contain char count 5/5000")
	}
}

func TestInputArea_CharCountWarning(t *testing.T) {
	ia := NewInputArea()
	ia.SetMaxChars(100)
	ia.Width = 80

	longText := ""
	for i := 0; i < 85; i++ {
		longText += "x"
	}
	ia.UpdateCharCount(longText)

	view := ia.View()
	if !strings.Contains(view, "85/100") {
		t.Error("view should contain char count 85/100")
	}
}

func TestInputArea_CharCountExceeded(t *testing.T) {
	ia := NewInputArea()
	ia.SetMaxChars(100)
	ia.Width = 80

	longText := ""
	for i := 0; i < 120; i++ {
		longText += "x"
	}
	ia.UpdateCharCount(longText)

	view := ia.View()
	if !strings.Contains(view, "120/100") {
		t.Error("view should contain char count 120/100")
	}
}

func TestInputArea_PasteConfirm(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80

	ia.PasteConfirm = true
	view := ia.View()
	if !strings.Contains(view, "按 Enter 确认发送") {
		t.Error("view should show paste confirmation prompt")
	}

	ia.PasteConfirm = false
	view = ia.View()
	if strings.Contains(view, "按 Enter 确认发送") {
		t.Error("view should not show paste confirmation prompt when false")
	}
}

func TestInputArea_UpdateCharCount_Unicode(t *testing.T) {
	ia := NewInputArea()
	ia.SetMaxChars(5000)

	text := "你好世界"
	ia.UpdateCharCount(text)

	if ia.CharCount != 4 {
		t.Errorf("unicode char count should be 4, got %d", ia.CharCount)
	}
}

func TestInputArea_SetMaxChars(t *testing.T) {
	ia := NewInputArea()

	if ia.MaxChars != 0 {
		t.Error("MaxChars should be 0 by default")
	}

	ia.SetMaxChars(5000)
	if ia.MaxChars != 5000 {
		t.Errorf("MaxChars should be 5000, got %d", ia.MaxChars)
	}
}
