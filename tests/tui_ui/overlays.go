package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ─── Overlay 类型 ───

type OverlayType int

const (
	OverlayNone OverlayType = iota
	OverlayApproval
	OverlayHelp
	OverlayCommand
	OverlaySession
	OverlayToast
	OverlayFiles
	OverlaySkills
	OverlayMCP
	OverlayMemory
	OverlayWorkspace
	OverlayNewSession
	OverlayRename
	OverlayRollback
)

// ─── Overlay Stack ───

type OverlayStack struct {
	current OverlayType
}

func (os *OverlayStack) Open(t OverlayType) {
	os.current = t
}

func (os *OverlayStack) Close() bool {
	if os.current == OverlayNone {
		return false
	}
	os.current = OverlayNone
	return true
}

func (os *OverlayStack) IsOpen() bool {
	return os.current != OverlayNone
}

// ─── Toast 通知 ───

type Toast struct {
	Message  string
	Type     string
	Duration int
	Width    int
}

func (t *Toast) Render() string {
	if t.Duration <= 0 {
		return ""
	}
	var styleFn func() lipgloss.Style
	switch t.Type {
	case "error":
		styleFn = ToastError
	case "success":
		styleFn = ToastSuccess
	default:
		styleFn = ToastInfo
	}
	content := styleFn().Render(" " + t.Message + " ")
	rightPad := t.Width - lipgloss.Width(content) - 2
	if rightPad < 0 {
		rightPad = 0
	}
	return strings.Repeat(" ", rightPad) + content
}

// ─── 命令面板 CommandSheet ───

type FlatCommand struct {
	Name        string
	Description string
	GroupName   string
}

type CommandSheet struct {
	Width        int
	Height       int
	Filter       string
	Selected     int
	Groups       []CommandGroup
	FlatCommands []FlatCommand
}

type CommandGroup struct {
	Name     string
	Commands []CommandItem
}

type CommandItem struct {
	Name        string
	Description string
}

func NewCommandSheet() CommandSheet {
	cs := CommandSheet{
		Groups: []CommandGroup{
			{
				Name: "SESSION",
				Commands: []CommandItem{
					{"/new", "创建新会话"},
					{"/switch", "切换会话"},
					{"/rename", "重命名会话"},
					{"/export", "导出会话"},
					{"/rollback", "回滚到消息"},
				},
			},
			{
				Name: "PANEL",
				Commands: []CommandItem{
					{"/files", "文件管理"},
					{"/skills", "技能管理"},
					{"/mcp", "MCP 管理"},
					{"/memory", "记忆管理"},
				},
			},
			{
				Name: "WORKSPACE",
				Commands: []CommandItem{
					{"/workspace", "切换工作区"},
					{"/w-create", "新建工作区"},
				},
			},
			{
				Name: "APP",
				Commands: []CommandItem{
					{"/yolo", "切换 YOLO 模式"},
					{"/theme", "切换主题"},
					{"/help", "帮助"},
					{"/quit", "退出"},
				},
			},
		},
	}
	cs.buildFlat()
	return cs
}

func (cs *CommandSheet) buildFlat() {
	cs.FlatCommands = nil
	for _, g := range cs.Groups {
		for _, cmd := range g.Commands {
			cs.FlatCommands = append(cs.FlatCommands, FlatCommand{
				Name:        cmd.Name,
				Description: cmd.Description,
				GroupName:   g.Name,
			})
		}
	}
}

func (cs *CommandSheet) CursorUp() {
	if cs.Selected > 0 {
		cs.Selected--
	}
}

func (cs *CommandSheet) CursorDown() {
	if cs.Selected < len(cs.FlatCommands)-1 {
		cs.Selected++
	}
}

func (cs *CommandSheet) SelectedCommand() FlatCommand {
	if cs.Selected >= 0 && cs.Selected < len(cs.FlatCommands) {
		return cs.FlatCommands[cs.Selected]
	}
	return FlatCommand{}
}

func (cs *CommandSheet) Render() string {
	w := cs.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(colorAccent()).Bold(true)
	muted := lipgloss.NewStyle().Foreground(colorMuted())
	accent := lipgloss.NewStyle().Foreground(colorAccent())
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 🔍 命令面板"))

	currentIdx := 0
	for _, g := range cs.Groups {
		lines = append(lines, "  "+label.Render(g.Name))
		for _, cmd := range g.Commands {
			name := accent.Render(cmd.Name)
			desc := muted.Render(cmd.Description)
			if currentIdx == cs.Selected {
				name = accent.Background(selBg).Render("▸" + cmd.Name[1:])
				desc = lipgloss.NewStyle().Foreground(colorText()).Background(selBg).Render(cmd.Description)
			}
			line := name + strings.Repeat(" ", innerW-2-lipgloss.Width(name)-lipgloss.Width(desc)) + desc
			lines = append(lines, " "+line)
			currentIdx++
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 导航  [Enter] 执行  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── 会话选择器 SessionPicker ───

type SessionPicker struct {
	Width    int
	Sessions []Session
	Selected int
}

func NewSessionPicker(sessions []Session) SessionPicker {
	return SessionPicker{Sessions: sessions}
}

func (sp *SessionPicker) CursorUp() {
	if sp.Selected > 0 {
		sp.Selected--
	}
}

func (sp *SessionPicker) CursorDown() {
	if sp.Selected < len(sp.Sessions)-1 {
		sp.Selected++
	}
}

func (sp *SessionPicker) Render() string {
	w := sp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 💬 切换会话"))

	for i, s := range sp.Sessions {
		active := ""
		if s.Active {
			active = lipgloss.NewStyle().Foreground(colorAccent()).Render("● ")
		}
		icon := lipgloss.NewStyle().Foreground(colorText()).Render("💬")
		name := lipgloss.NewStyle().Foreground(colorText()).Render(s.Name)
		preview := lipgloss.NewStyle().Foreground(colorMuted()).Render("  \"" + truncateStr(s.LastMsg, 30) + "\"")
		meta := lipgloss.NewStyle().Foreground(colorMuted()).Render(
			fmt.Sprintf("  %d条消息 · %s", s.MsgCount, s.LastActivity),
		)

		if i == sp.Selected {
			icon = lipgloss.NewStyle().Foreground(colorText()).Background(selBg).Render("💬")
			name = lipgloss.NewStyle().Foreground(colorText()).Background(selBg).Render(s.Name)
			preview = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render("  \"" + truncateStr(s.LastMsg, 30) + "\"")
			meta = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(
				fmt.Sprintf("  %d条消息 · %s", s.MsgCount, s.LastActivity),
			)
			if s.Active {
				active = lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("● ")
			}
			prefix := lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("▸") + active + icon + " "
			lines = append(lines, " "+prefix+name)
		} else {
			prefix := "  " + active + icon + " "
			lines = append(lines, " "+prefix+name)
		}
		lines = append(lines, " "+preview)
		lines = append(lines, " "+meta)
		if i < len(sp.Sessions)-1 {
			lines = append(lines, " ")
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 选择  [Enter] 确认  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── 审批弹窗 ApprovalModal ───

type ApprovalModal struct {
	Width     int
	Height    int
	Operation string
	Risk      string
	Diff      string
}

func (am *ApprovalModal) Render() string {
	w := am.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	riskColor := colorSuccess()
	if am.Risk == "HIGH" {
		riskColor = colorError()
	} else if am.Risk == "MEDIUM" {
		riskColor = colorWarning()
	}

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" ⚠ Approval Required"))

	lines = append(lines, " Operation: "+lipgloss.NewStyle().Foreground(colorAccent()).Render(am.Operation))
	lines = append(lines, " Risk:      "+lipgloss.NewStyle().Foreground(riskColor).Bold(true).Render(am.Risk))
	lines = append(lines, " ")

	lines = append(lines, " "+lipgloss.NewStyle().Foreground(colorBorder()).Render("┌ Diff "+strings.Repeat("─", innerW-9)))
	for _, dl := range strings.Split(am.Diff, "\n") {
		dl = strings.TrimSpace(dl)
		if dl == "" {
			continue
		}
		color := colorText()
		if strings.HasPrefix(dl, "+") {
			color = colorSuccess()
		} else if strings.HasPrefix(dl, "-") {
			color = colorError()
		}
		lines = append(lines, " "+lipgloss.NewStyle().Foreground(color).Render("│ "+truncateStr(dl, innerW-3)))
	}
	lines = append(lines, " "+lipgloss.NewStyle().Foreground(colorBorder()).Render("└"+strings.Repeat("─", innerW-3)))

	lines = append(lines, PanelFooterStyle(innerW).Render("[Y] Approve  [N] Reject  [D] Diff"))
	return strings.Join(lines, "\n")
}

// ─── 帮助面板 HelpPanel ───

type HelpPanel struct {
	Width  int
	Height int
}

func (hp *HelpPanel) Render() string {
	w := hp.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	sections := []struct {
		name  string
		items []string
	}{
		{"Navigation", []string{
			"↑/↓      行滚动",
			"PgUp/Dn  页滚动",
			"Ctrl+U   跳到上一条用户消息",
			"Ctrl+D   跳到下一条用户消息",
			"Tab      展开/折叠工具卡片",
		}},
		{"Chat", []string{
			"Enter    发送消息",
			"/        打开命令面板（输入框为空时）",
			"Ctrl+N   新建会话",
			"Ctrl+S   会话列表",
			"F2       重命名会话",
		}},
		{"Mode", []string{
			"Ctrl+T   切换主题（暗/亮）",
			"Ctrl+Y   切换 YOLO 模式",
			"Ctrl+P   暂停/恢复",
		}},
		{"Overlay", []string{
			"Esc      关闭覆盖层/面板",
			"?        打开帮助",
			"↑/↓/j/k 面板内光标移动",
			"Enter    面板内确认选择",
		}},
		{"System", []string{
			"Ctrl+C   退出",
			"Ctrl+Q   退出",
		}},
	}

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" Help"))

	for _, sec := range sections {
		lines = append(lines, " "+lipgloss.NewStyle().Foreground(colorAccent()).Bold(true).Render(sec.name))
		for _, item := range sec.items {
			parts := strings.SplitN(item, "  ", 2)
			key := lipgloss.NewStyle().Foreground(colorAccent()).Render(parts[0])
			desc := ""
			if len(parts) > 1 {
				desc = lipgloss.NewStyle().Foreground(colorText()).Render(parts[1])
			}
			lines = append(lines, " "+key+"  "+desc)
		}
		lines = append(lines, " ")
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── 文件管理面板 FilesPanel ───

type FileEntry struct {
	Name     string
	Size     string
	Type     string
	Modified string
}

type FilesPanel struct {
	Width    int
	Height   int
	Selected int
	Files    []FileEntry
}

func NewFilesPanel() FilesPanel {
	return FilesPanel{
		Files: []FileEntry{
			{"📁 auth/", "-", "dir", "2026-08-07"},
			{"  📄 service.go", "3.2K", "go", "2小时前"},
			{"  📄 handler.go", "2.1K", "go", "2小时前"},
			{"  📄 middleware.go", "1.5K", "go", "1小时前"},
			{"📁 config/", "-", "dir", "2026-08-06"},
			{"  📄 database.yaml", "0.8K", "yaml", "3小时前"},
			{"  📄 app.yaml", "1.2K", "yaml", "昨天"},
			{"📁 pkg/", "-", "dir", "2026-08-05"},
			{"  📄 utils.go", "4.5K", "go", "30分钟前"},
			{"  📄 validator.go", "2.8K", "go", "昨天"},
			{"📄 main.go", "1.0K", "go", "刚刚"},
			{"📄 go.mod", "0.5K", "mod", "2026-08-01"},
			{"📄 README.md", "2.3K", "md", "2026-08-01"},
		},
	}
}

func (fp *FilesPanel) CursorUp() {
	if fp.Selected > 0 {
		fp.Selected--
	}
}

func (fp *FilesPanel) CursorDown() {
	if fp.Selected < len(fp.Files)-1 {
		fp.Selected++
	}
}

func (fp *FilesPanel) Render() string {
	w := fp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 📂 文件管理"))

	for i, f := range fp.Files {
		name := lipgloss.NewStyle().Foreground(colorText()).Render(f.Name)
		meta := lipgloss.NewStyle().Foreground(colorMuted()).Render(f.Size + "  " + f.Modified)
		if i == fp.Selected {
			name = lipgloss.NewStyle().Foreground(colorText()).Background(selBg).Render(f.Name)
			meta = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(f.Size + "  " + f.Modified)
			prefix := lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("▸")
			line := prefix + name
			pad := innerW - 2 - lipgloss.Width(line) - lipgloss.Width(meta)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, " "+line+strings.Repeat(" ", pad)+meta)
		} else {
			line := " " + name
			pad := innerW - 2 - lipgloss.Width(line) - lipgloss.Width(meta)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, " "+line+strings.Repeat(" ", pad)+meta)
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 导航  [Enter] 打开  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── 技能管理面板 SkillsPanel ───

type SkillEntry struct {
	Name        string
	Description string
	Enabled     bool
}

type SkillsPanel struct {
	Width    int
	Height   int
	Selected int
	Skills   []SkillEntry
}

func NewSkillsPanel() SkillsPanel {
	return SkillsPanel{
		Skills: []SkillEntry{
			{"code-reviewer", "自动代码审查，检查安全漏洞和代码规范", true},
			{"test-generator", "自动生成单元测试和集成测试", true},
			{"api-designer", "RESTful API 设计和文档生成", false},
			{"db-migrator", "数据库迁移脚本生成和执行", true},
			{"deploy-helper", "一键部署到 K8s/ECS 等平台", false},
			{"log-analyzer", "日志分析和异常检测", true},
			{"doc-writer", "自动生成 API 文档和 README", true},
			{"refactor-assist", "代码重构辅助，提取方法和接口", false},
		},
	}
}

func (sp *SkillsPanel) CursorUp() {
	if sp.Selected > 0 {
		sp.Selected--
	}
}

func (sp *SkillsPanel) CursorDown() {
	if sp.Selected < len(sp.Skills)-1 {
		sp.Selected++
	}
}

func (sp *SkillsPanel) Toggle() {
	sp.Skills[sp.Selected].Enabled = !sp.Skills[sp.Selected].Enabled
}

func (sp *SkillsPanel) Render() string {
	w := sp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 🛠 技能管理"))

	for i, sk := range sp.Skills {
		toggle := lipgloss.NewStyle().Foreground(colorMuted()).Render("○")
		if sk.Enabled {
			toggle = lipgloss.NewStyle().Foreground(colorSuccess()).Render("●")
		}
		name := lipgloss.NewStyle().Foreground(colorAccent()).Render(sk.Name)
		desc := lipgloss.NewStyle().Foreground(colorMuted()).Render(truncateStr(sk.Description, 28))

		if i == sp.Selected {
			if sk.Enabled {
				toggle = lipgloss.NewStyle().Foreground(colorSuccess()).Background(selBg).Render("●")
			} else {
				toggle = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render("○")
			}
			name = lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render(sk.Name)
			desc = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(truncateStr(sk.Description, 28))
			prefix := lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("▸")
			lines = append(lines, " "+prefix+" "+toggle+" "+name+"  "+desc)
		} else {
			lines = append(lines, "   "+toggle+" "+name+"  "+desc)
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 导航  [Enter] 启用/停用  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── MCP 管理面板 MCPPanel ───

type MCPEntry struct {
	Name   string
	URL    string
	Status string
}

type MCPPanel struct {
	Width    int
	Height   int
	Selected int
	Servers  []MCPEntry
}

func NewMCPPanel() MCPPanel {
	return MCPPanel{
		Servers: []MCPEntry{
			{"github-mcp", "https://mcp.github.com/api", "connected"},
			{"slack-mcp", "https://mcp.slack.com/api", "connected"},
			{"jira-mcp", "https://mcp.jira.internal/api", "disconnected"},
			{"database-mcp", "postgresql://db.internal:5432/mcp", "connected"},
			{"redis-mcp", "redis://cache.internal:6379", "error"},
			{"filesystem-mcp", "file:///home/project", "connected"},
		},
	}
}

func (mp *MCPPanel) CursorUp() {
	if mp.Selected > 0 {
		mp.Selected--
	}
}

func (mp *MCPPanel) CursorDown() {
	if mp.Selected < len(mp.Servers)-1 {
		mp.Selected++
	}
}

func (mp *MCPPanel) Render() string {
	w := mp.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 🔌 MCP 服务器管理"))

	for i, srv := range mp.Servers {
		statusIcon := "●"
		statusColor := colorSuccess()
		switch srv.Status {
		case "disconnected":
			statusColor = colorMuted()
		case "error":
			statusColor = colorError()
			statusIcon = "✗"
		}
		status := lipgloss.NewStyle().Foreground(statusColor).Render(statusIcon + " " + srv.Status)
		name := lipgloss.NewStyle().Foreground(colorAccent()).Render(srv.Name)
		url := lipgloss.NewStyle().Foreground(colorMuted()).Render(truncateStr(srv.URL, 26))

		if i == mp.Selected {
			status = lipgloss.NewStyle().Foreground(statusColor).Background(selBg).Render(statusIcon + " " + srv.Status)
			name = lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render(srv.Name)
			url = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(truncateStr(srv.URL, 26))
			prefix := lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("▸")
			lines = append(lines, " "+prefix+name+"  "+url+"  "+status)
		} else {
			lines = append(lines, "  "+name+"  "+url+"  "+status)
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 导航  [Enter] 连接/断开  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── 记忆管理面板 MemoryPanel ───

type MemoryEntry struct {
	Key     string
	Content string
}

type MemoryPanel struct {
	Width    int
	Height   int
	Selected int
	Memories []MemoryEntry
}

func NewMemoryPanel() MemoryPanel {
	return MemoryPanel{
		Memories: []MemoryEntry{
			{"user_pref", "用户偏好使用 Go 语言，代码风格遵循 Uber Go Style Guide"},
			{"project_structure", "项目采用 Clean Architecture，分层为 handler/service/repository"},
			{"db_config", "生产数据库使用 PostgreSQL 15，连接地址 10.0.1.50:5432"},
			{"deploy_target", "部署目标为 K8s 集群，使用 Helm Chart 管理"},
			{"api_convention", "API 路径使用 /api/v1/ 前缀，认证使用 JWT Bearer Token"},
			{"ci_pipeline", "CI 使用 GitHub Actions，测试覆盖率要求 >80%"},
		},
	}
}

func (mp *MemoryPanel) CursorUp() {
	if mp.Selected > 0 {
		mp.Selected--
	}
}

func (mp *MemoryPanel) CursorDown() {
	if mp.Selected < len(mp.Memories)-1 {
		mp.Selected++
	}
}

func (mp *MemoryPanel) Render() string {
	w := mp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 🧠 记忆管理"))

	for i, mem := range mp.Memories {
		key := lipgloss.NewStyle().Foreground(colorAccent()).Render(mem.Key)
		content := lipgloss.NewStyle().Foreground(colorMuted()).Render(truncateStr(mem.Content, 36))

		if i == mp.Selected {
			key = lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render(mem.Key)
			content = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(truncateStr(mem.Content, 36))
			prefix := lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("▸")
			lines = append(lines, " "+prefix+key+"  "+content)
		} else {
			lines = append(lines, "  "+key+"  "+content)
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 导航  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── 工作区面板 WorkspacePanel ───

type WorkspaceEntry struct {
	Name   string
	Path   string
	Active bool
}

type WorkspacePanel struct {
	Width      int
	Height     int
	Selected   int
	Workspaces []WorkspaceEntry
}

func NewWorkspacePanel() WorkspacePanel {
	return WorkspacePanel{
		Workspaces: []WorkspaceEntry{
			{"my-project", "/home/project", true},
			{"devo-backend", "/home/workspace/devo-backend", false},
			{"devo-frontend", "/home/workspace/devo-frontend", false},
			{"cli-tools", "/home/workspace/cli-tools", false},
		},
	}
}

func (wp *WorkspacePanel) CursorUp() {
	if wp.Selected > 0 {
		wp.Selected--
	}
}

func (wp *WorkspacePanel) CursorDown() {
	if wp.Selected < len(wp.Workspaces)-1 {
		wp.Selected++
	}
}

func (wp *WorkspacePanel) Render() string {
	w := wp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 📁 工作区"))

	for i, ws := range wp.Workspaces {
		active := "  "
		if ws.Active {
			active = lipgloss.NewStyle().Foreground(colorSuccess()).Render("● ")
		}
		name := lipgloss.NewStyle().Foreground(colorText()).Render(ws.Name)
		path := lipgloss.NewStyle().Foreground(colorMuted()).Render(ws.Path)

		if i == wp.Selected {
			if ws.Active {
				active = lipgloss.NewStyle().Foreground(colorSuccess()).Background(selBg).Render("● ")
			}
			name = lipgloss.NewStyle().Foreground(colorText()).Background(selBg).Render(ws.Name)
			path = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(ws.Path)
			prefix := lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("▸")
			lines = append(lines, " "+prefix+active+name+"  "+path)
		} else {
			lines = append(lines, "  "+active+name+"  "+path)
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 导航  [Enter] 切换  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

// ─── 重命名弹窗 RenameModal ───

type RenameModal struct {
	Width   int
	Current string
	NewName string
}

func (rm *RenameModal) Render() string {
	w := rm.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" ✏ 重命名会话"))

	lines = append(lines, " "+lipgloss.NewStyle().Foreground(colorMuted()).Render("当前名称: "+rm.Current))
	lines = append(lines, " ")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent()).
		Width(innerW - 2).
		Render(rm.NewName + lipgloss.NewStyle().Foreground(colorMuted()).Render("█"))
	lines = append(lines, " "+inputBox)

	lines = append(lines, PanelFooterStyle(innerW).Render("[Enter] 确认  [Esc] 取消"))
	return strings.Join(lines, "\n")
}

// ─── 回滚选择器 RollbackPicker ───

type RollbackPicker struct {
	Width    int
	Height   int
	Selected int
	Messages []RollbackItem
}

type RollbackItem struct {
	Content string
	Role    string
	Time    string
}

func NewRollbackPicker(messages []Message) RollbackPicker {
	items := make([]RollbackItem, 0, len(messages))
	for _, m := range messages {
		role := "用户"
		if m.Role == RoleAssistant {
			role = "助手"
		} else if m.Role == RoleSystem {
			role = "系统"
		}
		items = append(items, RollbackItem{
			Content: truncateStr(m.Content, 40),
			Role:    role,
			Time:    m.Time,
		})
	}
	return RollbackPicker{
		Selected: len(items) - 1,
		Messages: items,
	}
}

func (rp *RollbackPicker) CursorUp() {
	if rp.Selected > 0 {
		rp.Selected--
	}
}

func (rp *RollbackPicker) CursorDown() {
	if rp.Selected < len(rp.Messages)-1 {
		rp.Selected++
	}
}

func (rp *RollbackPicker) Render() string {
	w := rp.Width
	if w < 36 {
		w = 36
	}
	innerW := w - 4
	selBg := lipgloss.Color("#1f3045")

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" ↩ 回滚到消息"))

	start := 0
	if len(rp.Messages) > 10 {
		start = len(rp.Messages) - 10
	}
	for i := start; i < len(rp.Messages); i++ {
		msg := rp.Messages[i]
		roleColor := colorAccent()
		if msg.Role == "助手" {
			roleColor = colorText()
		} else if msg.Role == "系统" {
			roleColor = colorMuted()
		}
		role := lipgloss.NewStyle().Foreground(roleColor).Render(msg.Role)
		content := lipgloss.NewStyle().Foreground(colorMuted()).Render(msg.Content)
		time := lipgloss.NewStyle().Foreground(colorMuted()).Render(msg.Time)

		if i == rp.Selected {
			role = lipgloss.NewStyle().Foreground(roleColor).Background(selBg).Render(msg.Role)
			content = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(msg.Content)
			time = lipgloss.NewStyle().Foreground(colorMuted()).Background(selBg).Render(msg.Time)
			prefix := lipgloss.NewStyle().Foreground(colorAccent()).Background(selBg).Render("▸")
			lines = append(lines, " "+prefix+role+"  "+content+"  "+time)
		} else {
			lines = append(lines, "  "+role+"  "+content+"  "+time)
		}
	}

	lines = append(lines, PanelFooterStyle(innerW).Render("[↑↓] 导航  [Enter] 确认回滚  [Esc] 取消"))
	return strings.Join(lines, "\n")
}

// ─── 新建会话确认弹窗 ───

type NewSessionModal struct {
	Width int
}

func (nsm *NewSessionModal) Render() string {
	w := nsm.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, PanelHeaderStyle(innerW).Render(" 🆕 新建会话"))

	lines = append(lines, " "+lipgloss.NewStyle().Foreground(colorMuted()).Render("确定要创建新会话吗？"))
	lines = append(lines, " "+lipgloss.NewStyle().Foreground(colorMuted()).Render("当前会话将被保存。"))

	lines = append(lines, PanelFooterStyle(innerW).Render("[Enter] 确认  [Esc] 取消"))
	return strings.Join(lines, "\n")
}

// ─── 工具函数 ───

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}

func padL(s string, n int) string {
	return strings.Repeat(" ", n) + s
}
