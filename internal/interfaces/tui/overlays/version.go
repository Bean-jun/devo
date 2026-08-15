package overlays

import (
	"strings"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type UpdateInfo struct {
	HasUpdate     bool
	LatestVersion string
	ReleaseName   string
	ReleaseBody   string
	ReleaseURL    string
	PublishedAt   string
}

type VersionPanel struct {
	Width      int
	Height     int
	Version    string
	UpdateInfo *UpdateInfo
}

func (vp *VersionPanel) Render() string {
	w := vp.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	value := lipgloss.NewStyle().Foreground(components.ColorText()).Bold(true)
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())
	highlight := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" Version"))

	lines = append(lines, "")
	lines = append(lines, "  "+label.Render("Devo")+"  "+value.Render("v"+vp.Version))
	lines = append(lines, "")

	if vp.UpdateInfo != nil && vp.UpdateInfo.HasUpdate {
		lines = append(lines, "  "+highlight.Render("\u25B6 新版本可用: "+vp.UpdateInfo.LatestVersion))
		if vp.UpdateInfo.PublishedAt != "" {
			lines = append(lines, "  "+muted.Render("发布时间: "+vp.UpdateInfo.PublishedAt))
		}
		lines = append(lines, "")

		if vp.UpdateInfo.ReleaseBody != "" {
			lines = append(lines, "  "+label.Render("更新内容"))
			lines = append(lines, "")

			renderedBody := vp.renderMarkdown(vp.UpdateInfo.ReleaseBody, innerW-2)
			for _, line := range strings.Split(renderedBody, "\n") {
				lines = append(lines, "  "+line)
			}
			lines = append(lines, "")
		}

		if vp.UpdateInfo.ReleaseURL != "" {
			lines = append(lines, "  "+muted.Render(vp.UpdateInfo.ReleaseURL))
			lines = append(lines, "")
		}
	} else {
		lines = append(lines, "  "+muted.Render("AI-Powered Development Orchestrator"))
		lines = append(lines, "  "+muted.Render("Terminal User Interface"))
		lines = append(lines, "")
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[Esc] 关闭"))
	return strings.Join(lines, "\n")
}

func (vp *VersionPanel) renderMarkdown(md string, width int) string {
	style := "dark"
	if !components.CurrentTheme.IsDark {
		style = "light"
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return md
	}
	rendered, err := renderer.Render(md)
	if err != nil {
		return md
	}
	return rendered
}
