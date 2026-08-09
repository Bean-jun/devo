package overlays

import (
	"strings"
	"testing"

	"devo/internal/interfaces/tui/types"
)

func TestOverlayStack_OpenClose(t *testing.T) {
	os := OverlayStack{}
	if os.IsOpen() {
		t.Error("新 OverlayStack 应为关闭状态")
	}

	os.Open(OverlayHelp)
	if !os.IsOpen() {
		t.Error("Open() 后应为打开状态")
	}
	if os.Current != OverlayHelp {
		t.Error("Open(OverlayHelp) 后 Current 应为 OverlayHelp")
	}

	os.Close()
	if os.IsOpen() {
		t.Error("Close() 后应为关闭状态")
	}
}

func TestOverlayStack_CloseTwice(t *testing.T) {
	os := OverlayStack{}
	os.Open(OverlayHelp)
	if !os.Close() {
		t.Error("首次 Close() 应返回 true")
	}
	if os.Close() {
		t.Error("第二次 Close() 应返回 false")
	}
}

func TestOverlayStack_Nested(t *testing.T) {
	os := OverlayStack{}
	os.Open(OverlayHelp)
	os.Open(OverlayCommand)
	if os.Current != OverlayCommand {
		t.Error("打开命令面板后 Current 应为 OverlayCommand")
	}
	os.Close()
	if os.Current != OverlayHelp {
		t.Error("关闭命令面板后应回到帮助面板")
	}
	os.Close()
	if os.IsOpen() {
		t.Error("关闭所有面板后应为关闭状态")
	}
}

func TestCommandSheet_New(t *testing.T) {
	cs := NewCommandSheet()
	if len(cs.FlatCommands) == 0 {
		t.Error("CommandSheet 应有命令列表")
	}
	if len(cs.Groups) == 0 {
		t.Error("CommandSheet 应有分组")
	}
}

func TestCommandSheet_CursorNavigation(t *testing.T) {
	cs := NewCommandSheet()
	cs.CursorDown()
	if cs.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	cs.CursorUp()
	if cs.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestCommandSheet_CursorUpBoundary(t *testing.T) {
	cs := NewCommandSheet()
	cs.CursorUp()
	if cs.Selected != 0 {
		t.Error("在顶部 CursorUp() 不应超出边界")
	}
}

func TestCommandSheet_CursorDownBoundary(t *testing.T) {
	cs := NewCommandSheet()
	for i := 0; i < len(cs.FlatCommands)+10; i++ {
		cs.CursorDown()
	}
	if cs.Selected >= len(cs.FlatCommands) {
		t.Error("CursorDown() 不应超出列表范围")
	}
}

func TestCommandSheet_SelectedCommand(t *testing.T) {
	cs := NewCommandSheet()
	cmd := cs.SelectedCommand()
	if cmd.Name != "/new" {
		t.Errorf("首个命令应为 /new, got %s", cmd.Name)
	}
}

func TestCommandSheet_CursorDownSelect(t *testing.T) {
	cs := NewCommandSheet()
	cs.CursorDown()
	cmd := cs.SelectedCommand()
	if cmd.Name != "/switch" {
		t.Errorf("第二个命令应为 /switch, got %s", cmd.Name)
	}
}

func TestCommandSheet_Render(t *testing.T) {
	cs := NewCommandSheet()
	cs.Width = 80
	result := cs.Render()
	if !strings.Contains(result, "命令面板") {
		t.Error("渲染结果应包含标题 '命令面板'")
	}
	if !strings.Contains(result, "▸new") {
		t.Error("渲染结果应包含选中命令 ▸new")
	}
	if !strings.Contains(result, "/help") {
		t.Error("渲染结果应包含 /help 命令")
	}
}

func TestHelpPanel_Render(t *testing.T) {
	hp := HelpPanel{Width: 80}
	result := hp.Render()
	if !strings.Contains(result, "Help") {
		t.Error("渲染结果应包含标题 Help")
	}
	if !strings.Contains(result, "Ctrl+C") {
		t.Error("渲染结果应包含 Ctrl+C 快捷键")
	}
	if !strings.Contains(result, "Enter") {
		t.Error("渲染结果应包含 Enter 快捷键")
	}
}

func TestHelpPanel_MinWidth(t *testing.T) {
	hp := HelpPanel{Width: 10}
	result := hp.Render()
	if result == "" {
		t.Error("即使宽度很小也应能渲染")
	}
}

func TestFilesPanel_Render(t *testing.T) {
	fp := NewFilesPanel()
	fp.Files = []FileEntry{
		{Name: "\U0001f4c4 auth.go", Size: "1.2K", Type: "go", Modified: "刚刚"},
	}
	fp.Width = 80
	result := fp.Render()
	if !strings.Contains(result, "文件管理") {
		t.Error("渲染结果应包含标题 '文件管理'")
	}
	if !strings.Contains(result, "auth.go") {
		t.Error("渲染结果应包含文件 auth.go")
	}
}

func TestFilesPanel_CursorNavigation(t *testing.T) {
	fp := NewFilesPanel()
	fp.Files = []FileEntry{{Name: "a"}, {Name: "b"}}
	fp.CursorDown()
	if fp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	fp.CursorUp()
	if fp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestSkillsPanel_Render(t *testing.T) {
	sp := NewSkillsPanel()
	sp.Skills = []SkillEntry{
		{Name: "code-reviewer", Description: "代码审查", Enabled: true},
	}
	sp.Width = 80
	result := sp.Render()
	if !strings.Contains(result, "技能管理") {
		t.Error("渲染结果应包含标题 '技能管理'")
	}
	if !strings.Contains(result, "code-reviewer") {
		t.Error("渲染结果应包含技能 code-reviewer")
	}
}

func TestSkillsPanel_CursorNavigation(t *testing.T) {
	sp := NewSkillsPanel()
	sp.Skills = []SkillEntry{{Name: "a"}, {Name: "b"}}
	sp.CursorDown()
	if sp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	sp.CursorUp()
	if sp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestMCPPanel_Render(t *testing.T) {
	mp := NewMCPPanel()
	mp.Servers = []MCPEntry{
		{Name: "filesystem", URL: "file:///tmp", Status: "connected"},
	}
	mp.Width = 80
	result := mp.Render()
	if !strings.Contains(result, "MCP 服务器") {
		t.Error("渲染结果应包含标题 'MCP 服务器'")
	}
	if !strings.Contains(result, "filesystem") {
		t.Error("渲染结果应包含 filesystem 服务器")
	}
}

func TestMCPPanel_CursorNavigation(t *testing.T) {
	mp := NewMCPPanel()
	mp.Servers = []MCPEntry{{Name: "a"}, {Name: "b"}}
	mp.CursorDown()
	if mp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	mp.CursorUp()
	if mp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestMemoryPanel_Render(t *testing.T) {
	mp := NewMemoryPanel()
	mp.Memories = []MemoryEntry{
		{Key: "user_pref", Content: "偏好设置"},
	}
	mp.Width = 80
	result := mp.Render()
	if !strings.Contains(result, "记忆管理") {
		t.Error("渲染结果应包含标题 '记忆管理'")
	}
	if !strings.Contains(result, "user_pref") {
		t.Error("渲染结果应包含 user_pref")
	}
}

func TestMemoryPanel_CursorNavigation(t *testing.T) {
	mp := NewMemoryPanel()
	mp.Memories = []MemoryEntry{{Key: "a"}, {Key: "b"}}
	mp.CursorDown()
	if mp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	mp.CursorUp()
	if mp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestWorkspacePanel_Render(t *testing.T) {
	wp := NewWorkspacePanel()
	wp.Workspaces = []WorkspaceEntry{
		{Name: "my-project", Path: "/tmp", Active: true},
	}
	wp.Width = 80
	result := wp.Render()
	if !strings.Contains(result, "工作区") {
		t.Error("渲染结果应包含标题 '工作区'")
	}
	if !strings.Contains(result, "my-project") {
		t.Error("渲染结果应包含工作区 my-project")
	}
}

func TestWorkspacePanel_CursorNavigation(t *testing.T) {
	wp := NewWorkspacePanel()
	wp.Workspaces = []WorkspaceEntry{{Name: "a"}, {Name: "b"}}
	wp.CursorDown()
	if wp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	wp.CursorUp()
	if wp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestNewSessionModal_Render(t *testing.T) {
	nsm := NewSessionModal{Width: 80}
	result := nsm.Render()
	if !strings.Contains(result, "新建会话") {
		t.Error("渲染结果应包含标题 '新建会话'")
	}
}

func TestRenameModal_Render(t *testing.T) {
	rm := RenameModal{Current: "demo-session", NewName: "新名称", Width: 80}
	result := rm.Render()
	if !strings.Contains(result, "重命名") {
		t.Error("渲染结果应包含标题 '重命名'")
	}
	if !strings.Contains(result, "demo-session") {
		t.Error("渲染结果应包含当前名称")
	}
	if !strings.Contains(result, "新名称") {
		t.Error("渲染结果应包含新名称")
	}
}

func TestRollbackPicker_Render(t *testing.T) {
	items := []RollbackItem{
		{Content: "第一条消息", Role: "用户", Time: "12:00"},
		{Content: "第二条消息", Role: "助手", Time: "12:01"},
	}
	rp := NewRollbackPicker(items)
	rp.Width = 80
	result := rp.Render()
	if !strings.Contains(result, "回滚") {
		t.Error("渲染结果应包含标题 '回滚'")
	}
	if !strings.Contains(result, "第一条消息") {
		t.Error("渲染结果应包含消息内容")
	}
}

func TestRollbackPicker_CursorNavigation(t *testing.T) {
	items := []RollbackItem{
		{Content: "a", Role: "用户", Time: "12:00"},
		{Content: "b", Role: "助手", Time: "12:01"},
	}
	rp := NewRollbackPicker(items)
	rp.CursorUp()
	if rp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestSessionPicker_Render(t *testing.T) {
	sp := SessionPicker{
		Width:    80,
		Selected: 0,
		Sessions: []types.SessionInfo{
			{ID: "1", Title: "test-session", MessageCount: 5},
		},
	}
	result := sp.Render()
	if !strings.Contains(result, "切换会话") {
		t.Error("渲染结果应包含标题 '切换会话'")
	}
	if !strings.Contains(result, "test-session") {
		t.Error("渲染结果应包含会话名称")
	}
}

func TestSessionPicker_CursorNavigation(t *testing.T) {
	sp := SessionPicker{
		Sessions: []types.SessionInfo{
			{ID: "1", Title: "a"},
			{ID: "2", Title: "b"},
			{ID: "3", Title: "c"},
		},
	}
	sp.CursorDown()
	if sp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	sp.CursorUp()
	if sp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestApprovalModal_Render(t *testing.T) {
	am := ApprovalModal{
		Width:     80,
		Operation: "delete",
		Risk:      "HIGH",
		Diff:      "- line1\n+ line2",
	}
	result := am.Render()
	if !strings.Contains(result, "Approval Required") {
		t.Error("渲染结果应包含标题 'Approval Required'")
	}
	if !strings.Contains(result, "delete") {
		t.Error("渲染结果应包含操作类型")
	}
	if !strings.Contains(result, "HIGH") {
		t.Error("渲染结果应包含风险等级")
	}
}
