package tui

import (
	"strings"
	"testing"
	"time"

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
	m.activeSessionID = "test-session"
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
	m.messages = append(m.messages, types.Message{
		ID: "msg-1", Role: "user", Content: "test",
	})
	m.rollback = overlays.NewRollbackPicker([]overlays.RollbackItem{
		{Content: "test", Role: "user", Time: "12:00", MsgIndex: 0},
	})
	m.rollback.TotalMessages = 1
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

func TestTruncateActivity_Short(t *testing.T) {
	result := truncateActivity("hello")
	if result != "hello" {
		t.Errorf("短文本不应被截断, got %s", result)
	}
}

func TestTruncateActivity_Long(t *testing.T) {
	long := "这是一段很长的文本用于测试截断功能看看会不会超过四十个字符还需要更多文字才行继续添加更多内容"
	result := truncateActivity(long)
	if len([]rune(result)) > 43 {
		t.Errorf("超长文本应被截断, got len=%d", len([]rune(result)))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("截断文本应以 ... 结尾, got %s", result)
	}
}

func TestTruncateActivity_Newlines(t *testing.T) {
	result := truncateActivity("hello\nworld")
	if strings.Contains(result, "\n") {
		t.Error("换行符应被替换为空格")
	}
	if !strings.Contains(result, "hello world") {
		t.Errorf("换行符应替换为空格, got %s", result)
	}
}

func TestTruncateActivity_CarriageReturn(t *testing.T) {
	result := truncateActivity("hello\rworld")
	if strings.Contains(result, "\r") {
		t.Error("回车符应被移除")
	}
}

func TestTruncateActivity_StreamingToken(t *testing.T) {
	result := truncateActivity("这是一段流式响应")
	if result != "这是一段流式响应" {
		t.Errorf("正常流式令牌不应被修改, got %s", result)
	}
}

func TestTruncateActivity_ReasoningPrefix(t *testing.T) {
	chunk := "让我们来分析一下这个问题的各个方面"
	result := truncateActivity(chunk)
	if !strings.HasPrefix(result, chunk) {
		t.Error("短推理内容不应被截断")
	}
}

func TestUpdate_ShiftEnterInsertsNewline(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("hello")
	msg := tea.KeyMsg(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModCtrl})
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if !strings.Contains(updated.textarea.Value(), "\n") {
		t.Error("Ctrl+Enter 应在 textarea 中插入换行符")
	}
	if !strings.HasPrefix(updated.textarea.Value(), "hello") {
		t.Error("Ctrl+Enter 应保留原有内容，在光标位置插入换行")
	}
}

func TestUpdate_EnterSendsMessage(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("test message")
	msgCount := len(m.messages)
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if len(updated.messages) <= msgCount {
		t.Error("Enter 应发送消息")
	}
}

func TestUpdate_EnterSavesToHistory(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("first message")
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if len(updated.inputHistory) != 1 {
		t.Errorf("发送消息后应保存到历史，期望 1 条，实际 %d 条", len(updated.inputHistory))
	}
	if updated.inputHistory[0] != "first message" {
		t.Errorf("历史内容应为 'first message'，实际为 '%s'", updated.inputHistory[0])
	}
}

func TestUpdate_EnterSavesMultipleHistory(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	model := &m

	model.textarea.SetValue("msg1")
	newModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = newModel.(*Model)

	model.textarea.SetValue("msg2")
	newModel, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = newModel.(*Model)

	if len(model.inputHistory) != 2 {
		t.Errorf("应保存 2 条历史，实际 %d 条", len(model.inputHistory))
	}
}

func TestUpdate_EnterNoDuplicateHistory(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	model := &m

	model.textarea.SetValue("same")
	newModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = newModel.(*Model)

	model.textarea.SetValue("same")
	newModel, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = newModel.(*Model)

	if len(model.inputHistory) != 1 {
		t.Errorf("连续相同内容不应重复保存，期望 1 条，实际 %d 条", len(model.inputHistory))
	}
}

func TestUpdate_ShiftUpHistoryPrev(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")

	m.pushInputHistory("msg1")
	m.pushInputHistory("msg2")

	m.textarea.SetValue("current draft")
	msg := tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if updated.textarea.Value() != "msg2" {
		t.Errorf("Ctrl+Up 应恢复最近一条历史，期望 'msg2'，实际 '%s'", updated.textarea.Value())
	}
	if updated.historyIndex != 0 {
		t.Errorf("historyIndex 应为 0，实际 %d", updated.historyIndex)
	}
	if updated.historyDraft != "current draft" {
		t.Errorf("应保存草稿 'current draft'，实际 '%s'", updated.historyDraft)
	}
}

func TestUpdate_ShiftUpHistoryPrevTwice(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	model := &m

	model.pushInputHistory("msg1")
	model.pushInputHistory("msg2")

	newModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})
	model = newModel.(*Model)
	newModel, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})
	model = newModel.(*Model)

	if model.textarea.Value() != "msg1" {
		t.Errorf("两次 Ctrl+Up 应恢复更早的历史，期望 'msg1'，实际 '%s'", model.textarea.Value())
	}
	if model.historyIndex != 1 {
		t.Errorf("historyIndex 应为 1，实际 %d", model.historyIndex)
	}
}

func TestUpdate_ShiftDownHistoryNext(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	model := &m

	model.pushInputHistory("msg1")
	model.pushInputHistory("msg2")

	model.textarea.SetValue("draft")
	newModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}) // go to msg2
	model = newModel.(*Model)
	newModel, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}) // go to msg1
	model = newModel.(*Model)
	newModel, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}) // go back to msg2
	model = newModel.(*Model)

	if model.textarea.Value() != "msg2" {
		t.Errorf("Ctrl+Down 应返回较新的历史，期望 'msg2'，实际 '%s'", model.textarea.Value())
	}
}

func TestUpdate_ShiftDownRestoresDraft(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	model := &m

	model.pushInputHistory("msg1")

	model.textarea.SetValue("my draft")
	newModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}) // go to msg1
	model = newModel.(*Model)
	newModel, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}) // go back to draft
	model = newModel.(*Model)

	if model.textarea.Value() != "my draft" {
		t.Errorf("Ctrl+Down 到最后应恢复草稿，期望 'my draft'，实际 '%s'", model.textarea.Value())
	}
	if model.historyIndex != -1 {
		t.Errorf("historyIndex 应恢复为 -1，实际 %d", model.historyIndex)
	}
}

func TestUpdate_ShiftUpEmptyHistory(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("some text")
	msg := tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if updated.textarea.Value() != "some text" {
		t.Error("没有历史时 Ctrl+Up 不应改变内容")
	}
}

func TestUpdate_ShiftDownEmptyHistory(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.textarea.SetValue("some text")
	msg := tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl}
	newModel, _ := m.Update(msg)
	updated := newModel.(*Model)

	if updated.textarea.Value() != "some text" {
		t.Error("没有历史时 Shift+Down 不应改变内容")
	}
}

func TestUpdate_EnterResetsHistoryIndex(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	model := &m

	model.pushInputHistory("msg1")
	newModel, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})
	model = newModel.(*Model)
	model.textarea.SetValue("new message")
	newModel, _ = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	model = newModel.(*Model)

	if model.historyIndex != -1 {
		t.Errorf("发送消息后 historyIndex 应重置为 -1，实际 %d", model.historyIndex)
	}
	if model.historyDraft != "" {
		t.Errorf("发送消息后 historyDraft 应清空，实际 '%s'", model.historyDraft)
	}
}

func TestShouldFoldPaste_LargeText(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	oldVal := "hello"
	newVal := "hello" + strings.Repeat("x", 250)

	if !m.shouldFoldPaste(oldVal, newVal) {
		t.Error("粘贴超过 200 字符应触发折叠")
	}
}

func TestShouldFoldPaste_ManyLines(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	oldVal := "hello"
	newVal := "hello\nline2\nline3\nline4\nline5\nline6"

	if !m.shouldFoldPaste(oldVal, newVal) {
		t.Error("粘贴超过 4 行应触发折叠")
	}
}

func TestShouldFoldPaste_SmallText(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	oldVal := "hello"
	newVal := "hello world"

	if m.shouldFoldPaste(oldVal, newVal) {
		t.Error("少量粘贴不应触发折叠")
	}
}

func TestShouldFoldPaste_NormalTyping(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	oldVal := "hello"
	newVal := "hello!"

	if m.shouldFoldPaste(oldVal, newVal) {
		t.Error("正常输入不应触发折叠")
	}
}

func TestResolvePasteBuffer_WithPaste(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.pasteBuffer = "full pasted text"
	result := m.resolvePasteBuffer("[Pasted text +100 chars, 3 lines]")

	if result != "full pasted text" {
		t.Errorf("应替换标记为粘贴内容，期望 'full pasted text'，实际 '%s'", result)
	}
}

func TestResolvePasteBuffer_WithExtraText(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.pasteBuffer = "hello\nworld"
	result := m.resolvePasteBuffer("[Pasted text +11 chars, 2 lines] extra")

	if result != "hello\nworld extra" {
		t.Errorf("应保留标记后的额外文本，期望 'hello\\nworld extra'，实际 '%s'", result)
	}
}

func TestResolvePasteBuffer_WithPrefix(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.pasteBuffer = "hello\nworld"
	result := m.resolvePasteBuffer("prefix [Pasted text +11 chars, 2 lines] suffix")

	if result != "prefix hello\nworld suffix" {
		t.Errorf("应保留标记前后的文本，期望 'prefix hello\\nworld suffix'，实际 '%s'", result)
	}
}

func TestResolvePasteBuffer_WithoutPaste(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.pasteBuffer = ""
	result := m.resolvePasteBuffer("normal text")

	if result != "normal text" {
		t.Errorf("无 pasteBuffer 时应返回原值，期望 'normal text'，实际 '%s'", result)
	}
}

func TestHandleEscNoOverlay_Idle(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStateIdle},
	}
	_, cmd := m.handleEscNoOverlay()
	if cmd != nil {
		t.Error("idle 状态第一次 ESC 不应返回 API 命令")
	}
}

func TestHandleEscNoOverlay_ToolExecuting(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStateToolExecuting},
	}
	_, cmd := m.handleEscNoOverlay()
	if cmd == nil {
		t.Error("tool_executing 状态 ESC 应返回暂停命令")
	}
}

func TestHandleEscNoOverlay_Paused(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStatePaused},
	}
	_, cmd := m.handleEscNoOverlay()
	if cmd == nil {
		t.Error("paused 状态 ESC 应返回取消命令")
	}
}

func TestHandleEscNoOverlay_Thinking(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStateThinking},
	}
	_, cmd := m.handleEscNoOverlay()
	if cmd == nil {
		t.Error("thinking 状态 ESC 应返回取消命令")
	}
}

func TestHandleEscNoOverlay_Processing(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStateProcessing},
	}
	_, cmd := m.handleEscNoOverlay()
	if cmd == nil {
		t.Error("processing 状态 ESC 应返回取消命令")
	}
}

func TestHandleEscNoOverlay_AwaitingApproval(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStateAwaitingApproval},
	}
	_, cmd := m.handleEscNoOverlay()
	if cmd == nil {
		t.Error("awaiting_approval 状态 ESC 应返回取消命令")
	}
}

func TestHandleEscNoOverlay_DoubleEsc(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStateIdle},
	}
	m.messages = append(m.messages, types.Message{
		Role: "user", Content: "test",
	})
	m.lastEscAt = time.Now().Add(-200 * time.Millisecond)
	newModel, _ := m.handleEscNoOverlay()
	updated := newModel.(*Model)

	if !updated.overlay.IsOpen() {
		t.Error("双击 ESC 应打开回滚面板")
	}
	if updated.overlay.Current != overlays.OverlayRollback {
		t.Errorf("双击 ESC 应打开回滚面板，实际 %v", updated.overlay.Current)
	}
}

func TestHandleEscNoOverlay_NoActiveSession(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = ""
	_, cmd := m.handleEscNoOverlay()
	if cmd != nil {
		t.Error("无活动会话时 ESC 不应返回命令")
	}
}

func TestGetActiveSessionState(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "test-1"
	m.sessions = []types.SessionInfo{
		{ID: "test-1", State: types.SessionStateThinking},
	}
	state := m.getActiveSessionState()
	if state != types.SessionStateThinking {
		t.Errorf("应返回活动会话状态，期望 thinking，实际 %s", state)
	}
}

func TestGetActiveSessionState_NoSession(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = ""
	state := m.getActiveSessionState()
	if state != types.SessionStateIdle {
		t.Errorf("无活动会话时应返回 idle，实际 %s", state)
	}
}

func TestGetActiveSessionState_NotFound(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.activeSessionID = "nonexistent"
	m.sessions = []types.SessionInfo{
		{ID: "other", State: types.SessionStateThinking},
	}
	state := m.getActiveSessionState()
	if state != types.SessionStateIdle {
		t.Errorf("找不到时应返回 idle，实际 %s", state)
	}
}

func TestFindPasteContent_Append(t *testing.T) {
	prefix, paste, suffix := findPasteContent("hello", "helloworld")
	if prefix != "hello" {
		t.Errorf("prefix should be 'hello', got '%s'", prefix)
	}
	if paste != "world" {
		t.Errorf("paste should be 'world', got '%s'", paste)
	}
	if suffix != "" {
		t.Errorf("suffix should be empty, got '%s'", suffix)
	}
}

func TestFindPasteContent_Insert(t *testing.T) {
	prefix, paste, suffix := findPasteContent("hello world", "hello big world")
	if prefix != "hello " {
		t.Errorf("prefix should be 'hello ', got '%s'", prefix)
	}
	if paste != "big " {
		t.Errorf("paste should be 'big ', got '%s'", paste)
	}
	if suffix != "world" {
		t.Errorf("suffix should be 'world', got '%s'", suffix)
	}
}

func TestPasteMarker(t *testing.T) {
	marker := pasteMarker(250, 5)
	if marker != "[Pasted text +250 chars, 5 lines]" {
		t.Errorf("marker should be '[Pasted text +250 chars, 5 lines]', got '%s'", marker)
	}
}

func TestAutoResizeTextarea_SingleLine(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.height = 30
	m.textarea.SetValue("hello")
	m.autoResizeTextarea()

	if m.textarea.Height() != 3 {
		t.Errorf("single line: textarea height should be 3, got %d", m.textarea.Height())
	}
}

func TestAutoResizeTextarea_MultiLine(t *testing.T) {
	m := NewModel("http://localhost:8080", "1.0.0")
	m.height = 30
	m.textarea.SetValue("line1\nline2\nline3\nline4\nline5")
	m.autoResizeTextarea()

	if m.textarea.Height() != 6 {
		t.Errorf("multi line: textarea height should be 6, got %d", m.textarea.Height())
	}
}
