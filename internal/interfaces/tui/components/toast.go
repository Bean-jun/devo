package components

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type Toast struct {
	Message  string
	Type     string
	Duration int
	Width    int
}

func (t *Toast) Show(msg string, isError bool) {
	t.Message = msg
	if isError {
		t.Type = "error"
		t.Duration = 5
	} else {
		t.Type = "success"
		t.Duration = 3
	}
}

func (t *Toast) Hide() {
	t.Duration = 0
}

func (t *Toast) Tick() {
	if t.Duration > 0 {
		t.Duration--
	}
}

func (t *Toast) Render() string {
	if t.Duration <= 0 {
		return ""
	}
	var styleFn func() lipgloss.Style
	switch t.Type {
	case "error":
		styleFn = ToastError
	case "success":
		styleFn = ToastSuccess
	default:
		styleFn = ToastInfo
	}
	content := styleFn().Render(" " + t.Message + " ")
	rightPad := t.Width - lipgloss.Width(content) - 2
	if rightPad < 0 {
		rightPad = 0
	}
	return strings.Repeat(" ", rightPad) + content
}
