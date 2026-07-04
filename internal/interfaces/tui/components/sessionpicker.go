package components

import (
	"fmt"
	"strings"
	"time"

	"devo/internal/interfaces/tui/types"

	"github.com/charmbracelet/lipgloss"
)

type SessionPicker struct {
	Visible  bool
	Sessions []types.SessionInfo
	ActiveID string
	Cursor   int
	Query    string
	Width    int
	Height   int
}

func NewSessionPicker() SessionPicker {
	return SessionPicker{}
}

func (s *SessionPicker) Show() {
	s.Visible = true
	s.Cursor = 0
	s.Query = ""
}

func (s *SessionPicker) Hide() {
	s.Visible = false
}

func (s *SessionPicker) CursorUp() {
	if s.Cursor > 0 {
		s.Cursor--
	}
}

func (s *SessionPicker) CursorDown() {
	filtered := s.filteredSessions()
	if s.Cursor < len(filtered)-1 {
		s.Cursor++
	}
}

func (s *SessionPicker) SelectedSession() *types.SessionInfo {
	filtered := s.filteredSessions()
	if s.Cursor >= 0 && s.Cursor < len(filtered) {
		return &filtered[s.Cursor]
	}
	return nil
}

func (s *SessionPicker) filteredSessions() []types.SessionInfo {
	if s.Query == "" {
		return s.Sessions
	}
	q := strings.ToLower(s.Query)
	var result []types.SessionInfo
	for _, sess := range s.Sessions {
		if strings.Contains(strings.ToLower(sess.Title), q) ||
			strings.Contains(strings.ToLower(sess.ID), q) {
			result = append(result, sess)
		}
	}
	return result
}

func formatDateTime(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z07:00", dateStr)
		if err != nil {
			return dateStr
		}
	}
	return t.Format("2006-01-02 15:04")
}

func formatTokenUsageStr(usage types.TokenUsage) string {
	total := usage.Input + usage.Output
	if total == 0 {
		return "0 token"
	}
	if total >= 1_000_000 {
		v := float64(total) / 1_000_000
		if v >= 10 {
			return fmt.Sprintf("%.0fM token", v)
		}
		return fmt.Sprintf("%.1fM token", v)
	}
	v := float64(total) / 1000
	if v >= 10 {
		return fmt.Sprintf("%.0fK token", v)
	}
	return fmt.Sprintf("%.1fK token", v)
}

var statusLabels = map[string]string{
	"idle":              "空闲",
	"processing":        "处理中",
	"awaiting_approval": "等待审批",
	"paused":            "已暂停",
	"completed":         "已完成",
	"archived":          "已归档",
}

func statusLabel(state string) string {
	if label, ok := statusLabels[state]; ok {
		return label
	}
	return state
}

func truncateTitle(title string, maxW int) string {
	runes := []rune(title)
	w := 0
	for i, r := range runes {
		cw := 1
		if r > 127 {
			cw = 2
		}
		if w+cw > maxW-1 {
			return string(runes[:i]) + "…"
		}
		w += cw
	}
	return title
}

func (s *SessionPicker) View() string {
	if !s.Visible {
		return ""
	}

	modalWidth := 70
	modalHeight := 22
	if s.Width > 0 && s.Width < modalWidth {
		modalWidth = s.Width - 4
	}
	if s.Height > 0 && s.Height < modalHeight {
		modalHeight = s.Height - 4
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Padding(0, 1)
	title := titleStyle.Render("会话列表")

	queryStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 1)
	queryDisplay := queryStyle.Render("> " + s.Query + "█")

	divider := lipgloss.NewStyle().
		Foreground(ColorBorder).
		Render(strings.Repeat("─", modalWidth-2))

	filtered := s.filteredSessions()
	listHeight := modalHeight - 6
	if listHeight > len(filtered) {
		listHeight = len(filtered)
	}
	if listHeight < 1 {
		listHeight = 1
	}

	innerW := modalWidth - 4

	statusStyle := func(state string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(StateColor(state))
	}

	var itemLines []string
	for i, sess := range filtered {
		if i >= listHeight {
			break
		}

		indicator := " "
		if sess.ID == s.ActiveID {
			indicator = "●"
		}

		statusStr := statusStyle(sess.State).Render(statusLabel(sess.State))
		statusWidth := lipgloss.Width(statusStr)

		sessTitle := sess.Title
		maxTitleW := innerW - 3 - statusWidth
		if lipgloss.Width(sessTitle) > maxTitleW {
			sessTitle = truncateTitle(sessTitle, maxTitleW)
		}

		titleContent := indicator + " " + lipgloss.NewStyle().Bold(true).Render(sessTitle)
		titleWidth := lipgloss.Width(titleContent)
		titlePad := innerW - titleWidth - statusWidth
		if titlePad < 0 {
			titlePad = 0
		}
		titleLine := titleContent + strings.Repeat(" ", titlePad) + statusStr

		meta := fmt.Sprintf("  %d 条消息 · %s · %s",
			sess.MessageCount,
			formatTokenUsageStr(sess.TokenUsage),
			formatDateTime(sess.CreatedAt),
		)
		metaLine := lipgloss.NewStyle().Foreground(ColorMuted).Width(innerW).Render(meta)

		item := lipgloss.JoinVertical(lipgloss.Left, titleLine, metaLine)

		if i == s.Cursor {
			itemLines = append(itemLines, SidebarActiveStyle.Padding(0, 1).Width(innerW).Render(item))
		} else {
			itemLines = append(itemLines, SidebarItemStyle.Padding(0, 1).Width(innerW).Render(item))
		}
	}

	emptyLines := listHeight - len(itemLines)
	for i := 0; i < emptyLines; i++ {
		itemLines = append(itemLines, "")
	}

	footer := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 1).
		Render("↑↓ 选择  Enter 确认  Esc 关闭")

	var parts []string
	parts = append(parts, title)
	parts = append(parts, queryDisplay)
	parts = append(parts, divider)
	parts = append(parts, itemLines...)
	parts = append(parts, divider)
	parts = append(parts, footer)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Render(content)
}
