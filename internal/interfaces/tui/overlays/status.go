package overlays

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type StatusInfo struct {
	SessionName   string
	SessionStatus string
	Yolo          bool
	WorkingDir    string
	Version       string
	InputTokens   int
	OutputTokens  int
	ContextTokens int
	Processing    bool
}

type StatusPanel struct {
	Width  int
	Height int
	Info   StatusInfo
}

func (sp *StatusPanel) Render() string {
	w := sp.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	value := lipgloss.NewStyle().Foreground(components.ColorText())

	yoloStatus := "关闭"
	if sp.Info.Yolo {
		yoloStatus = "开启"
	}
	processingStatus := "空闲"
	if sp.Info.Processing {
		processingStatus = "处理中"
	}

	rows := []struct {
		key string
		val string
	}{
		{"会话", sp.Info.SessionName},
		{"状态", sp.Info.SessionStatus},
		{"处理", processingStatus},
		{"YOLO", yoloStatus},
		{"工作目录", sp.Info.WorkingDir},
		{"版本", sp.Info.Version},
		{"上下文 Tokens", formatTokenCount(sp.Info.ContextTokens)},
		{"输入 Tokens", formatTokenCount(sp.Info.InputTokens)},
		{"输出 Tokens", formatTokenCount(sp.Info.OutputTokens)},
	}

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" Status"))

	for _, row := range rows {
		keyW := lipgloss.Width(row.key)
		keyText := label.Render(row.key)
		valText := value.Render(row.val)
		pad := innerW - 4 - keyW - lipgloss.Width(row.val)
		if pad < 1 {
			pad = 1
		}
		lines = append(lines, fmt.Sprintf("  %s%s%s", keyText, strings.Repeat(" ", pad), valText))
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[Esc] 关闭"))
	return strings.Join(lines, "\n")
}

func formatTokenCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
