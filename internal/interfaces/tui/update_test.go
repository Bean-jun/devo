package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/types"
)

func TestUpdate_WindowSizeMsg(t *testing.T) {
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

func TestUpdate_QuitKeys(t *testing.T) {
	tests := []tea.KeyMsg{
		tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl},
	}
	for _, key := range tests {
		m := NewModel("http://localhost:8080", "1.0.0")
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Errorf("%v 应返回退出命令", key)
		}
	}
}

func TestUpdate_OpenCommandPalette(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("")
	msg := tea.KeyPressMsg{Text: "/", Code: '/'}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if updated.overlay.Current != overlays.OverlayCommand {
		t.Error("输入框为空时按 / 应打开命令面板")
	}
}

func TestUpdate_SendMessage(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	msgCount := len(m.messages)
	m.textarea.SetValue("test message")
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if len(updated.messages) <= msgCount {
		t.Error("发送消息后消息列表应增加")
	}
}

func TestUpdate_EmptyEnter(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("")
	msgCount := len(m.messages)
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if len(updated.messages) != msgCount {
		t.Error("空输入按 Enter 不应增加消息")
	}
}

func TestUpdate_OpenHelp(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	msg := tea.KeyPressMsg{Text: "?", Code: '?'}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if updated.overlay.Current != overlays.OverlayHelp {
		t.Error("按 ? 应打开帮助面板")
	}
}

func TestHandleOverlayKey_Esc(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayHelp)
	msg := tea.KeyPressMsg{Code: tea.KeyEsc}
	newModel, _ := m.handleOverlayKey(msg)
	updated := newModel.(*Model)

	if updated.overlay.IsOpen() {
		t.Error("Esc 应关闭覆盖层")
	}
}

func TestHandleOverlayKey_Up(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayCommand)
	msg := tea.KeyPressMsg{Code: tea.KeyUp}
	_, _ = m.handleOverlayKey(msg)
}

func TestHandleOverlayKey_Down(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayCommand)
	msg := tea.KeyPressMsg{Code: tea.KeyDown}
	_, _ = m.handleOverlayKey(msg)
}

func TestHandleOverlayKey_JK(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayCommand)
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "j", Code: 'j'})
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "k", Code: 'k'})
}

func TestHandleOverlayKey_SpaceInSkills(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.skillsPanel.Skills = []overlays.SkillEntry{
		{Name: "code-reviewer", Description: "代码审查", Enabled: true},
	}
	m.overlay.Open(overlays.OverlaySkills)
	initialEnabled := m.skillsPanel.Skills[0].Enabled
	msg := tea.KeyPressMsg{Text: " ", Code: ' '}
	newModel, _ := m.handleOverlayKey(msg)
	updated := newModel.(*Model)

	if updated.skillsPanel.Skills[0].Enabled == initialEnabled {
		t.Error("Space 应切换技能启用状态")
	}
}

func TestHandleOverlayEnter_Command(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayCommand)
	_, _ = m.handleOverlayEnter()
}

func TestHandleOverlayEnter_Session(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.sessions = []types.SessionInfo{
		{ID: "1", Title: "test-session"},
	}
	m.sessPicker.Sessions = m.sessions
	m.overlay.Open(overlays.OverlaySession)
	newModel, _ := m.handleOverlayEnter()
	updated := newModel.(*Model)

	if updated.overlay.IsOpen() {
		t.Error("选择会话后应关闭覆盖层")
	}
}

func TestHandleOverlayEnter_Workspace(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayWorkspace)
	_, _ = m.handleOverlayEnter()
}

func TestHandleOverlayEnter_NewSession(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayNewSession)
	newModel, _ := m.handleOverlayEnter()
	updated := newModel.(*Model)

	if updated.overlay.IsOpen() {
		t.Error("新建会话后应关闭覆盖层")
	}
}

func TestHandleOverlayEnter_Rename(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.renameModal.Current = "test-session"
	m.renameModal.NewName = ""
	m.overlay.Open(overlays.OverlayRename)
	newModel, _ := m.handleOverlayEnter()
	updated := newModel.(*Model)

	if updated.overlay.IsOpen() {
		t.Error("重命名后应关闭覆盖层")
	}
}

func TestHandleOverlayEnter_Rollback(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.rollback = overlays.NewRollbackPicker([]overlays.RollbackItem{
		{Content: "test", Role: "user", Time: "12:00"},
	})
	m.overlay.Open(overlays.OverlayRollback)
	_, _ = m.handleOverlayEnter()
}

func TestHandleOverlayEnter_Skills(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlaySkills)
	_, _ = m.handleOverlayEnter()
}

func TestHandleOverlayEnter_MCP(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMCP)
	_, _ = m.handleOverlayEnter()
}

func TestHandleOverlayEnter_Memory(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayMemory)
	_, _ = m.handleOverlayEnter()
}

func TestHandleOverlayCursorUp_AllTypes(t *testing.T) {
	panels := []overlays.OverlayType{
		overlays.OverlayCommand,
		overlays.OverlaySession,
		overlays.OverlaySkills,
		overlays.OverlayMCP,
		overlays.OverlayMemory,
		overlays.OverlayWorkspace,
		overlays.OverlayRollback,
		overlays.OverlayBackground,
		overlays.OverlaySettings,
	}
	for _, pt := range panels {
		m := NewModel("http://localhost:8080", "1.0.0")
		m.overlay.Open(pt)
		m.handleOverlayCursorUp()
	}
}

func TestHandleOverlayCursorDown_AllTypes(t *testing.T) {
	panels := []overlays.OverlayType{
		overlays.OverlayCommand,
		overlays.OverlaySession,
		overlays.OverlaySkills,
		overlays.OverlayMCP,
		overlays.OverlayMemory,
		overlays.OverlayWorkspace,
		overlays.OverlayRollback,
		overlays.OverlayBackground,
		overlays.OverlaySettings,
	}
	for _, pt := range panels {
		m := NewModel("http://localhost:8080", "1.0.0")
		m.overlay.Open(pt)
		m.handleOverlayCursorDown()
	}
}

func TestUpdate_ToastTick(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.toast.Show("test", false)
	initialDuration := m.toast.Duration
	m.toast.Tick()

	if m.toast.Duration >= initialDuration {
		t.Error("Toast Tick 应减少 Duration")
	}
}

func TestUpdate_JumpToPrevUserMessage(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.Reset()
	msg := tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	_, _ = m.Update(msg)
}

func TestUpdate_JumpToNextUserMessage(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.Reset()
	msg := tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	_, _ = m.Update(msg)
}

func TestUpdate_JumpBlockedWhenTyping(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("typing")
	msgCount := len(m.messages)
	msg := tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if len(updated.messages) != msgCount {
		t.Error("输入框有内容时不应触发跳转（消息列表不应变化）")
	}
}

func TestHandleOverlayKey_JK_EditMode(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayRename)
	m.renameModal.NewName = ""
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "j", Code: 'j'})
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "k", Code: 'k'})
	if m.renameModal.NewName != "jk" {
		t.Errorf("编辑模式下 j/k 应输入字符, got %s", m.renameModal.NewName)
	}
}

func TestHandleOverlayKey_JK_NavMode(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlaySkills)
	m.skillsPanel.Skills = []overlays.SkillEntry{{Name: "a"}, {Name: "b"}, {Name: "c"}}
	m.skillsPanel.Selected = 0
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "j", Code: 'j'})
	if m.skillsPanel.Selected != 1 {
		t.Errorf("非编辑模式下 j 应导航下移, Selected=%d", m.skillsPanel.Selected)
	}
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "k", Code: 'k'})
	if m.skillsPanel.Selected != 0 {
		t.Errorf("非编辑模式下 k 应导航上移, Selected=%d", m.skillsPanel.Selected)
	}
}

func TestHandleOverlayKey_UpDown_AlwaysNav(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayRename)
	m.skillsPanel.Skills = []overlays.SkillEntry{{Name: "a"}, {Name: "b"}}
	m.skillsPanel.Selected = 0

	m.overlay.Open(overlays.OverlaySkills)
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.skillsPanel.Selected != 1 {
		t.Error("箭头下键应始终导航")
	}
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.skillsPanel.Selected != 0 {
		t.Error("箭头上键应始终导航")
	}
}

func TestHandleOverlayKey_NY_EditMode(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayRename)
	m.renameModal.NewName = ""
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "n", Code: 'n'})
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if m.renameModal.NewName != "ny" {
		t.Errorf("编辑模式下 n/y 应输入字符, got %s", m.renameModal.NewName)
	}
}

func TestHandleOverlayKey_NY_NonEdit(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlaySkills)
	m.skillsPanel.Skills = []overlays.SkillEntry{{Name: "test", Enabled: true}}
	m.skillsPanel.Selected = 0
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "n", Code: 'n'})
	_, _ = m.handleOverlayKey(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if m.skillsPanel.EditBuffer != "" {
		t.Error("非编辑模式下 n/y 不应追加到编辑缓冲")
	}
}

func TestHandleOverlayKey_ApprovalN(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayApproval)
	newModel, _ := m.handleOverlayKey(tea.KeyPressMsg{Text: "n", Code: 'n'})
	updated := newModel.(*Model)
	if updated.overlay.IsOpen() {
		t.Error("审批面板按 n 应关闭面板")
	}
}

func TestHandleOverlayKey_ApprovalY(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.overlay.Open(overlays.OverlayApproval)
	_, cmd := m.handleOverlayKey(tea.KeyPressMsg{Text: "y", Code: 'y'})
	if cmd == nil {
		t.Error("审批面板按 y 应返回审批命令")
	}
}
