package main

import "testing"

// ─── 命令路由测试 ───

func TestRouteCommand_OpensCorrectOverlay(t *testing.T) {
	tests := []struct {
		cmd     string
		overlay OverlayType
	}{
		{"/new", OverlayNewSession},
		{"/switch", OverlaySession},
		{"/rename", OverlayRename},
		{"/rollback", OverlayRollback},
		{"/files", OverlayFiles},
		{"/skills", OverlaySkills},
		{"/mcp", OverlayMCP},
		{"/memory", OverlayMemory},
		{"/workspace", OverlayWorkspace},
		{"/help", OverlayHelp},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			m := newModel()
			m.routeCommand(tt.cmd)
			if m.overlay.current != tt.overlay {
				t.Errorf("routeCommand(%q) 应打开 %v: got %v", tt.cmd, tt.overlay, m.overlay.current)
			}
		})
	}
}

func TestRouteCommand_ExportShowsToast(t *testing.T) {
	m := newModel()
	m.routeCommand("/export")
	if m.toast.Duration <= 0 {
		t.Error("导出命令应显示 Toast")
	}
}

func TestRouteCommand_YoloToggles(t *testing.T) {
	m := newModel()
	initial := m.statusBar.Yolo
	m.routeCommand("/yolo")
	if m.statusBar.Yolo == initial {
		t.Error("YOLO 命令应切换状态")
	}
}

func TestRouteCommand_ThemeToggles(t *testing.T) {
	m := newModel()
	initialTheme := currentTheme.Name
	m.routeCommand("/theme")
	if currentTheme.Name == initialTheme {
		t.Error("主题命令应切换主题")
	}
	toggleTheme()
}
