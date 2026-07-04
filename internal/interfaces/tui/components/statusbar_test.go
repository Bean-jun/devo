package components

import (
	"strings"
	"testing"
)

func TestStatusBar_YOLOMode(t *testing.T) {
	s := NewStatusBar()
	s.SessionState = "idle"
	s.Width = 80
	s.YOLOMode = true

	view := s.View()
	if !strings.Contains(view, "YOLO") {
		t.Error("status bar should display YOLO when YOLOMode is true")
	}
}

func TestStatusBar_YOLOModeOff(t *testing.T) {
	s := NewStatusBar()
	s.SessionState = "idle"
	s.Width = 80
	s.YOLOMode = false

	view := s.View()
	if strings.Contains(view, "YOLO") {
		t.Error("status bar should not display YOLO when YOLOMode is false")
	}
}

func TestStatusBar_StateDot(t *testing.T) {
	s := NewStatusBar()
	s.SessionState = "idle"
	s.Width = 80

	view := s.View()
	if !strings.Contains(view, "●") {
		t.Error("status bar should display a state dot ●")
	}
}

func TestStatusBar_AllStates(t *testing.T) {
	states := []string{
		"idle", "thinking", "tool_executing", "processing",
		"awaiting_approval", "paused", "cancelled", "completed", "archived",
	}

	for _, state := range states {
		s := NewStatusBar()
		s.SessionState = state
		s.Width = 80

		view := s.View()
		if view == "" {
			t.Errorf("status bar view should not be empty for state %q", state)
		}
		if !strings.Contains(view, "●") {
			t.Errorf("status bar should have state dot for state %q", state)
		}
	}
}
