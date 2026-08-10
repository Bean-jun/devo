package overlays

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/api"
	"devo/internal/interfaces/tui/components"
)

type DashboardPanel struct {
	Width        int
	Height       int
	SessionUsage *api.SessionUsageInfo
	ProjectUsage *api.ProjectUsageInfo
	GroupBy      string
}

func NewDashboardPanel() DashboardPanel {
	return DashboardPanel{
		GroupBy: "date",
	}
}

func (dp *DashboardPanel) Render() string {
	w := dp.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	accent := lipgloss.NewStyle().Foreground(components.ColorAccent())
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())
	text := lipgloss.NewStyle().Foreground(components.ColorText())

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" Dashboard"))

	// Session usage
	if dp.SessionUsage != nil {
		su := dp.SessionUsage
		lines = append(lines, "  "+label.Render("当前会话 Token 用量"))
		lines = append(lines, fmt.Sprintf("    输入: %s  输出: %s  合计: %s",
			accent.Render(formatTokenCount(su.TotalInputTokens)),
			accent.Render(formatTokenCount(su.TotalOutputTokens)),
			accent.Render(formatTokenCount(su.TotalTokens)),
		))
		if su.CompressionCount > 0 {
			lines = append(lines, fmt.Sprintf("    压缩次数: %d", su.CompressionCount))
		}

		if len(su.Steps) > 0 {
			lines = append(lines, "  "+muted.Render("步骤详情:"))
			maxTokens := 1
			for _, s := range su.Steps {
				if s.InputTokens+s.OutputTokens > maxTokens {
					maxTokens = s.InputTokens + s.OutputTokens
				}
			}
			barMax := innerW - 16
			for _, s := range su.Steps {
				total := s.InputTokens + s.OutputTokens
				barLen := 0
				if maxTokens > 0 {
					barLen = (total * barMax) / maxTokens
				}
				if barLen < 1 && total > 0 {
					barLen = 1
				}
				bar := strings.Repeat("\u2588", barLen)
				bar = accent.Render(bar)
				lines = append(lines, fmt.Sprintf("    #%-2d  %s %s",
					s.StepSeq, bar, text.Render(formatTokenCount(total))))
			}
		}
	} else {
		lines = append(lines, "  "+muted.Render("加载中..."))
	}

	lines = append(lines, "")

	// Project usage
	if dp.ProjectUsage != nil {
		pu := dp.ProjectUsage
		lines = append(lines, "  "+label.Render("项目 Token 用量"))
		lines = append(lines, fmt.Sprintf("    输入: %s  输出: %s  合计: %s",
			accent.Render(formatTokenCount(pu.Summary.Input)),
			accent.Render(formatTokenCount(pu.Summary.Output)),
			accent.Render(formatTokenCount(pu.Summary.Total)),
		))

		if len(pu.Groups) > 0 {
			lines = append(lines, "  "+muted.Render("分组详情:"))
			maxTokens := 1
			for _, g := range pu.Groups {
				if g.TotalTokens > maxTokens {
					maxTokens = g.TotalTokens
				}
			}
			barMax := innerW - 20
			for _, g := range pu.Groups {
				barLen := 0
				if maxTokens > 0 {
					barLen = (g.TotalTokens * barMax) / maxTokens
				}
				if barLen < 1 && g.TotalTokens > 0 {
					barLen = 1
				}
				bar := strings.Repeat("\u2588", barLen)
				bar = accent.Render(bar)
				key := g.Key
				if len(key) > 12 {
					key = key[:11] + "\u2026"
				}
				lines = append(lines, fmt.Sprintf("    %-12s  %s %s",
					key, bar, text.Render(formatTokenCount(g.TotalTokens))))
			}
		}
	} else {
		lines = append(lines, "  "+muted.Render("加载中..."))
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[Esc] 关闭"))
	return strings.Join(lines, "\n")
}
