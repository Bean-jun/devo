package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/api"
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
		{"/skills", overlays.OverlaySkills},
		{"/mcp", overlays.OverlayMCP},
		{"/memory", overlays.OverlayMemory},
		{"/workspace-switch", overlays.OverlayWorkspace},
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

func TestModel_RouteCommandToggleTheme(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.routeCommand("/toggle-theme")
	if m.toast.Duration != 3 {
		t.Error("toggle-theme 命令应触发 Toast")
	}
	components.CurrentTheme = components.Dark
}

func TestModel_RouteCommandPause(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-session-1"
	cmd := m.routeCommand("/pause")
	if cmd == nil {
		t.Error("pause 命令应返回 API Cmd")
	}
}

func TestModel_RouteCommandResume(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-session-1"
	cmd := m.routeCommand("/resume")
	if cmd == nil {
		t.Error("resume 命令应返回 API Cmd")
	}
}

func TestModel_RouteCommandCompact(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-session-1"
	cmd := m.routeCommand("/compact")
	if cmd == nil {
		t.Error("compact 命令应返回 API Cmd")
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

func TestModel_RouteCommandVersion(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.routeCommand("/version")
	if m.overlay.Current != overlays.OverlayVersion {
		t.Errorf("version 命令应打开 Version 面板, got %v", m.overlay.Current)
	}
}

func TestModel_RouteCommandStatus(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.routeCommand("/status")
	if m.overlay.Current != overlays.OverlayStatus {
		t.Errorf("status 命令应打开 Status 面板, got %v", m.overlay.Current)
	}
}

func TestModel_RouteCommandBackground(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-session-1"
	cmd := m.routeCommand("/background")
	if m.overlay.Current != overlays.OverlayBackground {
		t.Errorf("background 命令应打开 Background 面板, got %v", m.overlay.Current)
	}
	if cmd == nil {
		t.Error("background 命令应返回 API Cmd")
	}
}

func TestModel_RouteCommandDashboard(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-session-1"
	cmd := m.routeCommand("/dashboard")
	if m.overlay.Current != overlays.OverlayDashboard {
		t.Errorf("dashboard 命令应打开 Dashboard 面板, got %v", m.overlay.Current)
	}
	if cmd == nil {
		t.Error("dashboard 命令应返回 API Cmd")
	}
}

func TestModel_RouteCommandSettings(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	cmd := m.routeCommand("/settings")
	if m.overlay.Current != overlays.OverlaySettings {
		t.Errorf("settings 命令应打开 Settings 面板, got %v", m.overlay.Current)
	}
	if cmd == nil {
		t.Error("settings 命令应返回 API Cmd")
	}
}

func TestModel_RouteCommandReasoning(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.routeCommand("/reasoning")
	if m.overlay.Current != overlays.OverlayReasoning {
		t.Errorf("/reasoning 应打开 ReasoningPicker 面板, got %v", m.overlay.Current)
	}
}

func TestModel_RouteCommandReasoningSyncsStatusBar(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")

	m.applyReasoningOption(overlays.ReasoningOptions[1])
	if !m.enableReasoning {
		t.Error("选择 low 应启用思维链")
	}
	if m.reasoningEffort != "low" {
		t.Errorf("选择 low 后 reasoningEffort 应为 low, got %s", m.reasoningEffort)
	}
	if !m.statusBar.ReasoningEnabled {
		t.Error("statusBar 应同步 ReasoningEnabled=true")
	}
	if m.statusBar.ReasoningEffort != "low" {
		t.Errorf("statusBar 应同步 ReasoningEffort=low, got %s", m.statusBar.ReasoningEffort)
	}

	m.applyReasoningOption(overlays.ReasoningOptions[3])
	if m.reasoningEffort != "high" {
		t.Errorf("选择 high 后 reasoningEffort 应为 high, got %s", m.reasoningEffort)
	}

	m.applyReasoningOption(overlays.ReasoningOptions[0])
	if m.enableReasoning {
		t.Error("选择 off 应关闭思维链")
	}
	if m.statusBar.ReasoningEnabled {
		t.Error("关闭后 statusBar 应同步 ReasoningEnabled=false")
	}
}

func TestModel_SyncReasoningFromConfig(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")

	cfg := &api.GlobalConfigInfo{
		LLM: &api.LLMConfigInfo{
			EnableReasoning: true,
			ReasoningEffort: "high",
		},
	}
	m.syncReasoningFromConfig(cfg)

	if !m.enableReasoning {
		t.Error("syncReasoningFromConfig 应设置 enableReasoning=true")
	}
	if m.reasoningEffort != "high" {
		t.Errorf("syncReasoningFromConfig 应设置 reasonEffort=high, got %s", m.reasoningEffort)
	}
	if !m.statusBar.ReasoningEnabled {
		t.Error("syncReasoningFromConfig 应同步 statusBar.ReasoningEnabled=true")
	}
	if m.statusBar.ReasoningEffort != "high" {
		t.Errorf("syncReasoningFromConfig 应同步 statusBar.ReasoningEffort=high, got %s", m.statusBar.ReasoningEffort)
	}
}

func TestModel_SyncReasoningFromConfigNilLLM(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.enableReasoning = true
	m.reasoningEffort = "medium"

	cfg := &api.GlobalConfigInfo{
		LLM: nil,
	}
	m.syncReasoningFromConfig(cfg)

	if m.enableReasoning {
		t.Error("LLM 为 nil 时 enableReasoning 应为 false")
	}
	if m.reasoningEffort != "" {
		t.Errorf("LLM 为 nil 时 reasoningEffort 应为空, got %s", m.reasoningEffort)
	}
}

func TestModel_UpdateReasoningConfig(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.enableReasoning = true
	m.reasoningEffort = "medium"

	cmd := m.updateReasoningConfig()
	if cmd == nil {
		t.Error("updateReasoningConfig 应返回 API Cmd")
	}
}

func TestModel_NewModelDefaultReasoning(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")

	if m.enableReasoning {
		t.Error("新 Model 默认 enableReasoning 应为 false")
	}
	if m.reasoningEffort != "medium" {
		t.Errorf("新 Model 默认 reasoningEffort 应为 medium, got %s", m.reasoningEffort)
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

	if m.isLoading(overlays.OverlaySkills) {
		t.Error("新 Model 不应有加载状态")
	}

	m.setLoading(overlays.OverlaySkills, true)
	if !m.isLoading(overlays.OverlaySkills) {
		t.Error("setLoading 后应返回 true")
	}

	m.setLoading(overlays.OverlaySkills, false)
	if m.isLoading(overlays.OverlaySkills) {
		t.Error("取消加载后应返回 false")
	}
}

func TestModel_IsEditing_Rename(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayRename)
	if !m.isEditing() {
		t.Error("重命名面板应处于编辑模式")
	}
}

func TestModel_IsEditing_Command(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayCommand)
	if !m.isEditing() {
		t.Error("命令面板应处于编辑模式")
	}
}

func TestModel_IsEditing_SkillsIdle(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlaySkills)
	if m.isEditing() {
		t.Error("技能面板未编辑时不应处于编辑模式")
	}
}

func TestModel_IsEditing_SkillsEditing(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlaySkills)
	m.skillsPanel.StartEditing()
	if !m.isEditing() {
		t.Error("技能面板编辑中应处于编辑模式")
	}
}

func TestModel_IsEditing_MCPIdle(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMCP)
	if m.isEditing() {
		t.Error("MCP 面板未编辑时不应处于编辑模式")
	}
}

func TestModel_IsEditing_MCPEditing(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMCP)
	m.mcpPanel.StartEditing()
	if !m.isEditing() {
		t.Error("MCP 面板编辑中应处于编辑模式")
	}
}

func TestModel_IsEditing_MemoryIdle(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMemory)
	if m.isEditing() {
		t.Error("记忆面板未编辑时不应处于编辑模式")
	}
}

func TestModel_IsEditing_MemoryEditing(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMemory)
	m.memoryPanel.StartEditing()
	if !m.isEditing() {
		t.Error("记忆面板编辑中应处于编辑模式")
	}
}

func TestModel_IsEditing_SettingsIdle(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlaySettings)
	if m.isEditing() {
		t.Error("设置面板未编辑时不应处于编辑模式")
	}
}

func TestModel_IsEditing_Help(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayHelp)
	if m.isEditing() {
		t.Error("帮助面板不应处于编辑模式")
	}
}

func TestModel_AppendEditChar_Rename(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayRename)
	m.appendEditChar("n")
	m.appendEditChar("y")
	m.appendEditChar("j")
	m.appendEditChar("k")
	if m.renameModal.NewName != "nyjk" {
		t.Errorf("重命名面板输入应为 nyjk, got %s", m.renameModal.NewName)
	}
}

func TestModel_AppendEditChar_Command(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayCommand)
	m.appendEditChar("n")
	m.appendEditChar("y")
	if m.cmdSheet.Filter != "ny" {
		t.Errorf("命令面板过滤应为 ny, got %s", m.cmdSheet.Filter)
	}
}

func TestModel_AppendEditChar_Skills(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlaySkills)
	m.skillsPanel.StartEditing()
	m.appendEditChar("y")
	m.appendEditChar("n")
	if m.skillsPanel.EditBuffer != "yn" {
		t.Errorf("技能面板编辑缓冲应为 yn, got %s", m.skillsPanel.EditBuffer)
	}
}

func TestModel_AppendEditChar_MCP(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMCP)
	m.mcpPanel.StartEditing()
	m.appendEditChar("j")
	m.appendEditChar("k")
	if m.mcpPanel.EditBuffer != "jk" {
		t.Errorf("MCP 面板编辑缓冲应为 jk, got %s", m.mcpPanel.EditBuffer)
	}
}

func TestModel_AppendEditChar_Memory(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMemory)
	m.memoryPanel.StartEditing()
	m.appendEditChar("n")
	m.appendEditChar("y")
	if m.memoryPanel.EditBuffer != "ny" {
		t.Errorf("记忆面板编辑缓冲应为 ny, got %s", m.memoryPanel.EditBuffer)
	}
}

func TestModel_ExtractPort(t *testing.T) {
	if p := extractPort("http://localhost:8080"); p != "8080" {
		t.Errorf("端口应为 8080, got %s", p)
	}
	if p := extractPort("http://127.0.0.1:3000"); p != "3000" {
		t.Errorf("端口应为 3000, got %s", p)
	}
	if p := extractPort("invalid-url"); p != "" {
		t.Errorf("无效 URL 应返回空字符串, got %s", p)
	}
}
