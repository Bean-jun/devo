package overlays

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/types"
)

type SessionPicker struct {
	Width    int
	Sessions []types.SessionInfo
	Selected int
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

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f4ac 切换会话"))

	for i, s := range sp.Sessions {
		active := ""
		icon := lipgloss.NewStyle().Foreground(components.ColorText()).Render("\U0001f4ac")
		name := lipgloss.NewStyle().Foreground(components.ColorText()).Render(s.Title)
		preview := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("  \"" + truncateStr(s.LastMessageContent, 30) + "\"")
		meta := lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(
			fmt.Sprintf("  %d条消息 · %s", s.MessageCount, s.LastMessageTime),
		)

		if i == sp.Selected {
			icon = lipgloss.NewStyle().Foreground(components.ColorText()).Render("\U0001f4ac")
			name = lipgloss.NewStyle().Foreground(components.ColorText()).Render(s.Title)
			preview = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render("  \"" + truncateStr(s.LastMessageContent, 30) + "\"")
			meta = lipgloss.NewStyle().Foreground(components.ColorMuted()).Render(
				fmt.Sprintf("  %d条消息 · %s", s.MessageCount, s.LastMessageTime),
			)
			prefix := lipgloss.NewStyle().Foreground(components.ColorAccent()).Render("\u25b8") + active + icon + " "
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

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193] 选择  [Enter] 确认  [Esc] 关闭"))
	return strings.Join(lines, "\n")
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}
