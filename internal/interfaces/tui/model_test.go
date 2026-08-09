package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/types"
)

func TestNewModel(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	if m.apiClient == nil {
		t.Error("API 客户端不应为 nil")
	}
	if m.loading == nil {
		t.Error("loading map 应初始化")
	}
}

func TestModel_ReadyAfterWindowSize(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if !updated.ready {
		t.Error("WindowSizeMsg 后 ready 应为 true")
	}
	if updated.width != 100 {
		t.Errorf("width 应为 100, got %d", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("height 应为 40, got %d", updated.height)
	}
}

func TestModel_OverlayPanelWidth(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.width = 100
	w := m.overlayPanelWidth()
	if w != 80 {
		t.Errorf("宽度 100 时面板宽度应为 80, got %d", w)
	}

	m.width = 50
	w = m.overlayPanelWidth()
	if w != 46 {
		t.Errorf("宽度 50 时面板宽度应为 46, got %d", w)
	}

	m.width = 30
	w = m.overlayPanelWidth()
	if w != 26 {
		t.Errorf("宽度 30 时面板宽度应为 26, got %d", w)
	}
}

func TestModel_RefreshViewport(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.messages = append(m.messages, types.Message{
		Role:    "user",
		Content: "test message",
	})
	m.refreshViewport()
}

func TestModel_RouteCommand(t *testing.T) {
	tests := []struct {
		cmd          string
		expectedType overlays.OverlayType
	}{
		{"/new", overlays.OverlayNewSession},
		{"/switch", overlays.OverlaySession},
		{"/rename", overlays.OverlayRename},
		{"/rollback", overlays.OverlayRollback},
		{"/files", overlays.OverlayFiles},
		{"/skills", overlays.OverlaySkills},
		{"/mcp", overlays.OverlayMCP},
		{"/memory", overlays.OverlayMemory},
		{"/workspace", overlays.OverlayWorkspace},
		{"/help", overlays.OverlayHelp},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			m := NewModel("http://localhost:8080", "1.0.0")
			m.routeCommand(tt.cmd)
			if m.overlay.Current != tt.expectedType {
				t.Errorf("routeCommand(%s) 应打开 %v, got %v", tt.cmd, tt.expectedType, m.overlay.Current)
			}
		})
	}
}

func TestModel_RouteCommandTheme(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.routeCommand("/theme")
	if m.toast.Duration != 3 {
		t.Error("theme 命令应触发 Toast")
	}
	components.CurrentTheme = components.Dark
}

func TestModel_RouteCommandYolo(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	oldYolo := m.statusBar.Yolo
	m.routeCommand("/yolo")
	if m.statusBar.Yolo == oldYolo {
		t.Error("yolo 命令应切换 YOLO 状态")
	}
}

func TestModel_RouteCommandPause(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	oldPaused := m.statusBar.Paused
	m.routeCommand("/pause")
	if m.statusBar.Paused == oldPaused {
		t.Error("pause 命令应切换暂停状态")
	}
}

func TestModel_RouteCommandUnknown(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.routeCommand("/unknown")
	if m.toast.Type != "error" {
		t.Error("未知命令应触发错误 Toast")
	}
}

func TestModel_RouteCommandExport(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-session-1"
	cmd := m.routeCommand("/export")
	if cmd == nil {
		t.Error("export 命令应返回 API Cmd")
	}
}

func TestModel_RouteCommandWCreate(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.routeCommand("/w-create")
	if m.toast.Duration != 3 {
		t.Error("w-create 命令应触发 Toast")
	}
}

func TestModel_RouteCommandQuit(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	cmd := m.routeCommand("/quit")
	if cmd == nil {
		t.Error("quit 命令应返回退出 Cmd")
	}
}

func TestModel_FindUserMessageYOffsets(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.messages = []types.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	m.refreshViewport()
	offsets := m.findUserMessageYOffsets()
	if len(offsets) == 0 {
		t.Error("有用户消息时应返回偏移量列表")
	}
}

func TestModel_JumpToPrevUserMessage(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.messages = []types.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	m.refreshViewport()
	m.viewport.SetYOffset(100)
	m.jumpToPrevUserMessage()
}

func TestModel_JumpToNextUserMessage(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.messages = []types.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	m.refreshViewport()
	m.viewport.SetYOffset(0)
	m.jumpToNextUserMessage()
}

func TestModel_View(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.ready = true
	m.width = 80
	m.height = 24
	view := m.View()
	if view.Content == "" {
		t.Error("View() 不应返回空字符串")
	}
	if view.Content == "loading..." {
		t.Error("ready 为 true 时不应显示 loading...")
	}
}

func TestModel_ViewNotReady(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.ready = false
	view := m.View()
	if view.Content != "loading..." {
		t.Errorf("未 ready 时应显示 loading..., got %s", view.Content)
	}
}

func TestModel_Init(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() 不应返回 nil Cmd")
	}
}

func TestModel_LoadingState(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")

	if m.isLoading(overlays.OverlayFiles) {
		t.Error("新 Model 不应有加载状态")
	}

	m.setLoading(overlays.OverlayFiles, true)
	if !m.isLoading(overlays.OverlayFiles) {
		t.Error("setLoading 后应返回 true")
	}

	m.setLoading(overlays.OverlayFiles, false)
	if m.isLoading(overlays.OverlayFiles) {
		t.Error("取消加载后应返回 false")
	}
}
