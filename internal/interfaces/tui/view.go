package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/overlays"
)

func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("loading...")
	}

	statusBar := m.statusBar.Render()
	inputArea := m.renderInputArea()

	var centerArea string
	if m.overlay.IsOpen() {
		centerArea = m.renderOverlay()
	} else {
		centerArea = m.viewport.View()
	}

	main := lipgloss.JoinVertical(lipgloss.Left,
		m.renderToastLine(),
		statusBar,
		centerArea,
		inputArea,
	)

	v := tea.NewView(main)
	v.AltScreen = true
	return v
}

func (m Model) renderToastLine() string {
	toast := m.toast.Render()
	if toast == "" {
		return strings.Repeat(" ", m.width)
	}
	return toast
}

func (m Model) renderInputArea() string {
	w := m.width

	sep := lipgloss.NewStyle().
		Foreground(components.ColorBorder()).
		Render(strings.Repeat("\u2500", w))

	taView := m.textarea.View()

	sendIcon := lipgloss.NewStyle().
		Foreground(components.ColorAccent()).
		Bold(true).
		Render(" \u23ce ")
	if m.statusBar.Processing {
		sendIcon = lipgloss.NewStyle().Foreground(components.ColorError()).Bold(true).Render(" \u25a0 ")
	}

	footerText := m.buildFooterText()
	footerTextW := lipgloss.Width(footerText)
	sendIconW := lipgloss.Width(sendIcon)
	pad := w - 1 - footerTextW - sendIconW
	if pad < 0 {
		pad = 0
	}
	footer := lipgloss.NewStyle().
		Foreground(components.ColorMuted()).
		Width(w).
		Render(" " + footerText + strings.Repeat(" ", pad) + sendIcon)

	return sep + "\n" + taView + "\n" + sep + "\n" + footer
}

func (m Model) renderOverlay() string {
	panelContent := m.getPanelContent()
	if panelContent == "" {
		return ""
	}

	boxed := components.OverlayBoxStyle().Render(panelContent)

	vpW := m.viewport.Width()
	vpH := m.viewport.Height()

	return lipgloss.Place(
		vpW, vpH,
		lipgloss.Center, lipgloss.Center,
		boxed,
	)
}

func (m Model) getPanelContent() string {
	switch m.overlay.Current {
	case overlays.OverlayCommand:
		return m.cmdSheet.Render()
	case overlays.OverlaySession:
		return m.sessPicker.Render()
	case overlays.OverlayHelp:
		return m.helpPanel.Render()
	case overlays.OverlaySkills:
		return m.skillsPanel.Render()
	case overlays.OverlayMCP:
		return m.mcpPanel.Render()
	case overlays.OverlayMemory:
		return m.memoryPanel.Render()
	case overlays.OverlayWorkspace:
		return m.wsPanel.Render()
	case overlays.OverlayNewSession:
		return m.newSessModal.Render()
	case overlays.OverlayRename:
		return m.renameModal.Render()
	case overlays.OverlayRollback:
		return m.rollback.Render()
	case overlays.OverlayApproval:
		return m.approval.Render()
	case overlays.OverlayStatus:
		return m.statusPanel.Render()
	case overlays.OverlayVersion:
		return m.versionPanel.Render()
	case overlays.OverlayBackground:
		return m.backgroundPanel.Render()
	case overlays.OverlayDashboard:
		return m.dashboardPanel.Render()
	case overlays.OverlaySettings:
		return m.settingsPanel.Render()
	case overlays.OverlayReasoning:
		return m.reasoningPicker.Render()
	default:
		return ""
	}
}
