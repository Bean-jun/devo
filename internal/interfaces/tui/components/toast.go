package components

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

type Toast struct {
	Message   string
	IsError   bool
	Visible   bool
	ExpiresAt time.Time
	Width     int
}

func NewToast() Toast {
	return Toast{}
}

func (t *Toast) Show(message string, isError bool) {
	t.Message = message
	t.IsError = isError
	t.Visible = true
	t.ExpiresAt = time.Now().Add(3 * time.Second)
}

func (t *Toast) Hide() {
	t.Visible = false
}

func (t *Toast) Tick() {
	if t.Visible && time.Now().After(t.ExpiresAt) {
		t.Visible = false
	}
}

func (t *Toast) View() string {
	if !t.Visible {
		return ""
	}

	style := ToastInfoStyle
	if t.IsError {
		style = ToastErrorStyle
	}

	toast := style.Copy().Width(t.Width - 4).Render(t.Message)
	return lipgloss.NewStyle().
		Width(t.Width).
		Align(lipgloss.Center).
		Render(toast)
}
