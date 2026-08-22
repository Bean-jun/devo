package overlays

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type NewSessionModal struct {
	Width    int
	Agents   []AgentItem
	Selected int
}

type AgentItem struct {
	ID          string
	Name        string
	Description string
}

func (nsm *NewSessionModal) CursorUp() {
	if nsm.Selected > 0 {
		nsm.Selected--
	}
}

func (nsm *NewSessionModal) CursorDown() {
	if nsm.Selected < len(nsm.Agents)-1 {
		nsm.Selected++
	}
}

func (nsm *NewSessionModal) SelectedAgentID() string {
	if nsm.Selected >= 0 && nsm.Selected < len(nsm.Agents) {
		return nsm.Agents[nsm.Selected].ID
	}
	return ""
}

func (nsm *NewSessionModal) Render() string {
	w := nsm.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f195 新建会话"))

	lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("确定要创建新会话吗？"))
	lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("当前会话将被保存。"))

	if len(nsm.Agents) > 0 {
		lines = append(lines, "")
		lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true).Render("选择 Agent:"))

		for i, a := range nsm.Agents {
			prefix := "  "
			style := lipgloss.NewStyle().Foreground(components.ColorMuted())
			if i == nsm.Selected {
				prefix = lipgloss.NewStyle().Foreground(components.ColorAccent()).Render(" \u25b8")
				style = lipgloss.NewStyle().Foreground(components.ColorText())
			}
			line := fmt.Sprintf("%s%s %s", prefix, style.Render(a.Name), lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(a.Description))
			lines = append(lines, line)
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 选择 Agent  [Enter] 确认  [Esc] 取消"))
	return strings.Join(lines, "\n")
}

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
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \u270f 重命名会话"))

	lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("当前名称: "+rm.Current))
	lines = append(lines, " ")

	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(components.ColorAccent()).
		Width(innerW - 2).
		Render(rm.NewName + lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("\u2588"))
	lines = append(lines, " "+inputBox)

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[Enter] 确认  [Esc] 取消"))
	return strings.Join(lines, "\n")
}

type RollbackItem struct {
	Content  string
	Role     string
	Time     string
	MsgIndex int
}

type RollbackPicker struct {
	Width         int
	Height        int
	Selected      int
	Messages      []RollbackItem
	TotalMessages int
}

func NewRollbackPicker(messages []RollbackItem) RollbackPicker {
	return RollbackPicker{
		Selected: 0,
		Messages: messages,
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

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \u21a9 回滚到消息"))

	for i, msg := range rp.Messages {
		roleColor := components.ColorAccent()
		if msg.Role == "助手" {
			roleColor = components.ColorText()
		} else if msg.Role == "系统" {
			roleColor = components.ColorMuted()
		}
		role := lipgloss.NewStyle().Foreground(roleColor).Render("你")
		content := truncateStr(msg.Content, 40)
		contentStyled := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(content)
		time := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(msg.Time)

		if i == rp.Selected {
			role = lipgloss.NewStyle().Foreground(roleColor).Render("你")
			contentStyled = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(content)
			time = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(msg.Time)
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8")
			lines = append(lines, " "+prefix+role+"  "+contentStyled+"  "+time)
		} else {
			lines = append(lines, "  "+role+"  "+contentStyled+"  "+time)
		}
	}

	if rp.Selected >= 0 && rp.Selected < len(rp.Messages) {
		selectedItem := rp.Messages[rp.Selected]
		deletedCount := rp.TotalMessages - selectedItem.MsgIndex - 1
		if deletedCount > 0 {
			lines = append(lines, "")
			warning := fmt.Sprintf(" 将删除 %d 条消息，此操作不可撤销", deletedCount)
			lines = append(lines, " "+lipgloss.NewStyle().Foreground(components.ColorError()).Render(warning))
		}
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 导航  [Enter] 确认回滚  [Esc] 取消"))
	return strings.Join(lines, "\n")
}
