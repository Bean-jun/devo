package components

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStateColors_HasAllStates(t *testing.T) {
	expectedStates := []string{
		"idle",
		"thinking",
		"tool_executing",
		"processing",
		"awaiting_approval",
		"paused",
		"cancelled",
		"completed",
		"archived",
	}

	for _, state := range expectedStates {
		if _, ok := StateColors[state]; !ok {
			t.Errorf("StateColors missing state: %q", state)
		}
	}

	if len(StateColors) != len(expectedStates) {
		t.Errorf("StateColors has %d entries, expected %d", len(StateColors), len(expectedStates))
	}
}

func TestStateColors_ColorMapping(t *testing.T) {
	tests := []struct {
		state     string
		wantColor lipgloss.Color
	}{
		{"idle", ColorSuccess},
		{"thinking", ColorInfo},
		{"tool_executing", ColorPrimary},
		{"processing", ColorInfo},
		{"awaiting_approval", ColorWarning},
		{"paused", ColorMuted},
		{"cancelled", ColorDanger},
		{"completed", ColorSuccess},
		{"archived", ColorMuted},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := StateColor(tt.state)
			if got != tt.wantColor {
				t.Errorf("StateColor(%q) = %v, want %v", tt.state, got, tt.wantColor)
			}
		})
	}
}

func TestStateColor_UnknownState(t *testing.T) {
	got := StateColor("nonexistent")
	if got != ColorMuted {
		t.Errorf("StateColor for unknown state = %v, want %v (ColorMuted)", got, ColorMuted)
	}
}
