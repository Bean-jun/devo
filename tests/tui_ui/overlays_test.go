package main

import (
	"strings"
	"testing"
)

// ─── Overlay Stack 测试 ───

func TestOverlayStack_OpenClose(t *testing.T) {
	os := &OverlayStack{}
	if os.IsOpen() {
		t.Error("新 OverlayStack 不应是打开状态")
	}

	os.Open(OverlayCommand)
	if !os.IsOpen() {
		t.Error("Open 后应是打开状态")
	}
	if os.current != OverlayCommand {
		t.Error("current 应为 OverlayCommand")
	}

	closed := os.Close()
	if !closed {
		t.Error("Close 应返回 true")
	}
	if os.IsOpen() {
		t.Error("Close 后不应是打开状态")
	}

	closed = os.Close()
	if closed {
		t.Error("再次 Close 应返回 false")
	}
}

// ─── 命令面板测试 ───

func TestCommandSheet_CursorNavigation(t *testing.T) {
	cs := NewCommandSheet()

	if len(cs.FlatCommands) == 0 {
		t.Fatal("FlatCommands 不应为空")
	}

	if cs.Selected != 0 {
		t.Errorf("初始 Selected 应为 0: got %d", cs.Selected)
	}

	cs.CursorUp()
	if cs.Selected != 0 {
		t.Error("CursorUp 在顶部不应越界")
	}

	cs.CursorDown()
	if cs.Selected != 1 {
		t.Errorf("CursorDown 应移动至 1: got %d", cs.Selected)
	}

	for i := 0; i < len(cs.FlatCommands); i++ {
		cs.CursorDown()
	}
	lastIdx := len(cs.FlatCommands) - 1
	if cs.Selected != lastIdx {
		t.Errorf("应移动至末尾 %d: got %d", lastIdx, cs.Selected)
	}

	cs.CursorDown()
	if cs.Selected != lastIdx {
		t.Error("CursorDown 在底部不应越界")
	}
}

func TestCommandSheet_SelectedCommand(t *testing.T) {
	cs := NewCommandSheet()

	sel := cs.SelectedCommand()
	if sel.Name == "" {
		t.Error("选中命令 Name 不应为空")
	}

	cs.Selected = len(cs.FlatCommands) - 1
	sel = cs.SelectedCommand()
	if sel.Name == "" {
		t.Error("末尾选中命令 Name 不应为空")
	}
}

func TestCommandSheet_RenderWithSelection(t *testing.T) {
	cs := NewCommandSheet()
	cs.Width = 80
	cs.Selected = 2

	result := cs.Render()
	if !strings.Contains(result, "▸") {
		t.Error("渲染结果应包含选中标记 '▸'")
	}
	if !strings.Contains(result, "[↑↓]") {
		t.Error("渲染结果应包含导航提示")
	}
	if !strings.Contains(result, "[Enter]") {
		t.Error("渲染结果应包含 Enter 提示")
	}
}

// ─── 会话选择器测试 ───

func TestSessionPicker_CursorNavigation(t *testing.T) {
	sessions := mockSessions()
	sp := NewSessionPicker(sessions)

	if sp.Selected != 0 {
		t.Errorf("初始 Selected 应为 0: got %d", sp.Selected)
	}

	sp.CursorUp()
	if sp.Selected != 0 {
		t.Error("CursorUp 在顶部不应越界")
	}

	sp.CursorDown()
	if sp.Selected != 1 {
		t.Errorf("CursorDown 应移动至 1: got %d", sp.Selected)
	}

	for i := 0; i < len(sessions); i++ {
		sp.CursorDown()
	}
	lastIdx := len(sessions) - 1
	if sp.Selected != lastIdx {
		t.Errorf("应移动至末尾 %d: got %d", lastIdx, sp.Selected)
	}

	sp.CursorDown()
	if sp.Selected != lastIdx {
		t.Error("CursorDown 在底部不应越界")
	}
}

func TestSessionPicker_RenderWithSelection(t *testing.T) {
	sessions := mockSessions()
	sp := NewSessionPicker(sessions)
	sp.Width = 80
	sp.Selected = 1

	result := sp.Render()
	if !strings.Contains(result, "▸") {
		t.Error("渲染结果应包含选中标记 '▸'")
	}
	if !strings.Contains(result, "[↑↓]") {
		t.Error("渲染结果应包含导航提示")
	}
	if !strings.Contains(result, "[Enter]") {
		t.Error("渲染结果应包含 Enter 提示")
	}
}

// ─── 文件管理面板测试 ───

func TestFilesPanel_CursorNavigation(t *testing.T) {
	fp := NewFilesPanel()
	if fp.Selected != 0 {
		t.Errorf("初始 Selected 应为 0: got %d", fp.Selected)
	}

	fp.CursorUp()
	if fp.Selected != 0 {
		t.Error("CursorUp 在顶部不应越界")
	}

	fp.CursorDown()
	if fp.Selected != 1 {
		t.Errorf("CursorDown 应移动至 1: got %d", fp.Selected)
	}

	lastIdx := len(fp.Files) - 1
	for i := 0; i < len(fp.Files); i++ {
		fp.CursorDown()
	}
	if fp.Selected != lastIdx {
		t.Errorf("应移动至末尾 %d: got %d", lastIdx, fp.Selected)
	}
}

func TestFilesPanel_Render(t *testing.T) {
	fp := NewFilesPanel()
	fp.Width = 80
	result := fp.Render()
	if !strings.Contains(result, "文件管理") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "auth") {
		t.Error("渲染结果应包含文件列表")
	}
}

// ─── 技能管理面板测试 ───

func TestSkillsPanel_CursorNavigation(t *testing.T) {
	sp := NewSkillsPanel()
	if sp.Selected != 0 {
		t.Errorf("初始 Selected 应为 0: got %d", sp.Selected)
	}

	sp.CursorDown()
	if sp.Selected != 1 {
		t.Errorf("CursorDown 应移动至 1: got %d", sp.Selected)
	}

	lastIdx := len(sp.Skills) - 1
	for i := 0; i < len(sp.Skills); i++ {
		sp.CursorDown()
	}
	if sp.Selected != lastIdx {
		t.Errorf("应移动至末尾 %d: got %d", lastIdx, sp.Selected)
	}
}

func TestSkillsPanel_Toggle(t *testing.T) {
	sp := NewSkillsPanel()
	initial := sp.Skills[0].Enabled
	sp.Toggle()
	if sp.Skills[0].Enabled == initial {
		t.Error("Toggle 应改变启用状态")
	}
	sp.Toggle()
	if sp.Skills[0].Enabled != initial {
		t.Error("再次 Toggle 应恢复初始状态")
	}
}

func TestSkillsPanel_Render(t *testing.T) {
	sp := NewSkillsPanel()
	sp.Width = 80
	result := sp.Render()
	if !strings.Contains(result, "技能管理") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "code-reviewer") {
		t.Error("渲染结果应包含技能列表")
	}
}

// ─── MCP 面板测试 ───

func TestMCPPanel_CursorNavigation(t *testing.T) {
	mp := NewMCPPanel()
	if mp.Selected != 0 {
		t.Errorf("初始 Selected 应为 0: got %d", mp.Selected)
	}

	mp.CursorDown()
	if mp.Selected != 1 {
		t.Errorf("CursorDown 应移动至 1: got %d", mp.Selected)
	}

	lastIdx := len(mp.Servers) - 1
	for i := 0; i < len(mp.Servers); i++ {
		mp.CursorDown()
	}
	if mp.Selected != lastIdx {
		t.Errorf("应移动至末尾 %d: got %d", lastIdx, mp.Selected)
	}
}

func TestMCPPanel_Render(t *testing.T) {
	mp := NewMCPPanel()
	mp.Width = 80
	result := mp.Render()
	if !strings.Contains(result, "MCP") {
		t.Error("渲染结果应包含 MCP 标题")
	}
	if !strings.Contains(result, "github-mcp") {
		t.Error("渲染结果应包含服务器列表")
	}
}

// ─── 记忆面板测试 ───

func TestMemoryPanel_CursorNavigation(t *testing.T) {
	mp := NewMemoryPanel()
	if mp.Selected != 0 {
		t.Errorf("初始 Selected 应为 0: got %d", mp.Selected)
	}

	mp.CursorDown()
	if mp.Selected != 1 {
		t.Errorf("CursorDown 应移动至 1: got %d", mp.Selected)
	}

	lastIdx := len(mp.Memories) - 1
	for i := 0; i < len(mp.Memories); i++ {
		mp.CursorDown()
	}
	if mp.Selected != lastIdx {
		t.Errorf("应移动至末尾 %d: got %d", lastIdx, mp.Selected)
	}
}

func TestMemoryPanel_Render(t *testing.T) {
	mp := NewMemoryPanel()
	mp.Width = 80
	result := mp.Render()
	if !strings.Contains(result, "记忆管理") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "user_pref") {
		t.Error("渲染结果应包含记忆列表")
	}
}

// ─── 工作区面板测试 ───

func TestWorkspacePanel_CursorNavigation(t *testing.T) {
	wp := NewWorkspacePanel()
	if wp.Selected != 0 {
		t.Errorf("初始 Selected 应为 0: got %d", wp.Selected)
	}

	wp.CursorDown()
	if wp.Selected != 1 {
		t.Errorf("CursorDown 应移动至 1: got %d", wp.Selected)
	}

	lastIdx := len(wp.Workspaces) - 1
	for i := 0; i < len(wp.Workspaces); i++ {
		wp.CursorDown()
	}
	if wp.Selected != lastIdx {
		t.Errorf("应移动至末尾 %d: got %d", lastIdx, wp.Selected)
	}
}

func TestWorkspacePanel_Render(t *testing.T) {
	wp := NewWorkspacePanel()
	wp.Width = 80
	result := wp.Render()
	if !strings.Contains(result, "工作区") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "my-project") {
		t.Error("渲染结果应包含工作区列表")
	}
}

// ─── 回滚选择器测试 ───

func TestRollbackPicker_CursorNavigation(t *testing.T) {
	msgs := mockMessages()
	rp := NewRollbackPicker(msgs)

	if rp.Selected != len(rp.Messages)-1 {
		t.Errorf("初始 Selected 应为最后一条: got %d, want %d", rp.Selected, len(rp.Messages)-1)
	}

	rp.CursorUp()
	if rp.Selected != len(rp.Messages)-2 {
		t.Errorf("CursorUp 应移动至 %d: got %d", len(rp.Messages)-2, rp.Selected)
	}

	rp.CursorDown()
	if rp.Selected != len(rp.Messages)-1 {
		t.Errorf("CursorDown 应恢复至 %d: got %d", len(rp.Messages)-1, rp.Selected)
	}
}

func TestRollbackPicker_Render(t *testing.T) {
	msgs := mockMessages()
	rp := NewRollbackPicker(msgs)
	rp.Width = 80
	result := rp.Render()
	if !strings.Contains(result, "回滚") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "用户") {
		t.Error("渲染结果应包含消息角色")
	}
}

// ─── 重命名弹窗测试 ───

func TestRenameModal_Render(t *testing.T) {
	rm := RenameModal{Current: "demo-session", NewName: "新会话", Width: 80}
	result := rm.Render()
	if !strings.Contains(result, "重命名") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "demo-session") {
		t.Error("渲染结果应包含当前名称")
	}
	if !strings.Contains(result, "新会话") {
		t.Error("渲染结果应包含新名称")
	}
}

// ─── 新建会话弹窗测试 ───

func TestNewSessionModal_Render(t *testing.T) {
	nsm := NewSessionModal{Width: 80}
	result := nsm.Render()
	if !strings.Contains(result, "新建会话") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "确定") {
		t.Error("渲染结果应包含确认按钮")
	}
}

// ─── 工具函数测试 ───

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		input       string
		maxLen      int
		expectedLen int
	}{
		{"Hello", 10, 5},
		{"Hello World", 8, 8},
		{"你好世界", 3, 3},
		{"", 10, 0},
	}

	for _, tt := range tests {
		result := truncateStr(tt.input, tt.maxLen)
		runes := []rune(result)
		if len(runes) != tt.expectedLen {
			t.Errorf("truncateStr(%q, %d) = %q (len=%d), want len=%d", tt.input, tt.maxLen, result, len(runes), tt.expectedLen)
		}
	}
}

func TestPadL(t *testing.T) {
	result := padL("hello", 5)
	expected := "     hello"
	if result != expected {
		t.Errorf("padL 错误: got %q, want %q", result, expected)
	}
}
