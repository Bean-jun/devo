package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ─── View ───

func (m model) View() string {
	if !m.ready {
		return "loading..."
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
		statusBar,
		centerArea,
		inputArea,
	)

	if toast := m.toast.Render(); toast != "" {
		main = toast + "\n" + main
	}

	return main
}

func (m model) renderInputArea() string {
	w := m.width

	// 顶部分隔线
	sep := lipgloss.NewStyle().
		Foreground(colorBorder()).
		Render(strings.Repeat("─", w))

	// 输入行
	taView := m.textarea.View()

	// 底部栏：左侧信息 + 右侧发送按钮
	sendIcon := lipgloss.NewStyle().
		Foreground(colorAccent()).
		Bold(true).
		Render(" ⏎ ")
	if m.statusBar.Processing {
		sendIcon = lipgloss.NewStyle().Foreground(colorError()).Bold(true).Render(" ■ ")
	}

	footerText := "Context: 12.5K  ·  Tokens: 3.2K  ·  /home/project"
	footerTextW := lipgloss.Width(footerText)
	sendIconW := lipgloss.Width(sendIcon)
	pad := w - 1 - footerTextW - sendIconW
	if pad < 0 {
		pad = 0
	}
	footer := lipgloss.NewStyle().
		Foreground(colorMuted()).
		Width(w).
		Render(" " + footerText + strings.Repeat(" ", pad) + sendIcon)

	return sep + "\n" + taView + "\n" + sep + "\n" + footer
}

// ─── 覆盖层渲染 ───

func (m model) renderOverlay() string {
	panelContent := m.getPanelContent()
	if panelContent == "" {
		return ""
	}

	boxed := OverlayBoxStyle().Render(panelContent)

	vpW := m.viewport.Width
	vpH := m.viewport.Height

	return lipgloss.Place(
		vpW, vpH,
		lipgloss.Center, lipgloss.Center,
		boxed,
		lipgloss.WithWhitespaceBackground(dimBgColor()),
	)
}

// getPanelContent 根据当前覆盖层类型返回面板内容
func (m model) getPanelContent() string {
	switch m.overlay.current {
	case OverlayCommand:
		return m.cmdSheet.Render()
	case OverlaySession:
		return m.sessPicker.Render()
	case OverlayApproval:
		return m.approval.Render()
	case OverlayHelp:
		return m.helpPanel.Render()
	case OverlayFiles:
		return m.filesPanel.Render()
	case OverlaySkills:
		return m.skillsPanel.Render()
	case OverlayMCP:
		return m.mcpPanel.Render()
	case OverlayMemory:
		return m.memoryPanel.Render()
	case OverlayWorkspace:
		return m.wsPanel.Render()
	case OverlayNewSession:
		return m.newSessModal.Render()
	case OverlayRename:
		return m.renameModal.Render()
	case OverlayRollback:
		return m.rollback.Render()
	default:
		return ""
	}
}
