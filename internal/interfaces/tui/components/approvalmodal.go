package components

import (
	"fmt"

	"devo/internal/interfaces/tui/types"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ApprovalModal struct {
	Visible  bool
	Request  *types.ApprovalRequest
	viewport viewport.Model
	Width    int
	Height   int
}

func NewApprovalModal() ApprovalModal {
	vp := viewport.New(60, 15)
	return ApprovalModal{
		viewport: vp,
	}
}

func (m *ApprovalModal) Show(req *types.ApprovalRequest, width, height int) {
	m.Visible = true
	m.Request = req
	m.Width = width
	m.Height = height

	modalWidth := width - 10
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 40 {
		modalWidth = 40
	}
	m.viewport.Width = modalWidth - 6
	m.viewport.Height = height - 14

	content := m.buildDiffContent()
	m.viewport.SetContent(content)
	m.viewport.GotoTop()
}

func (m *ApprovalModal) Hide() {
	m.Visible = false
	m.Request = nil
}

func (m *ApprovalModal) Update(msg tea.Msg) (ApprovalModal, tea.Cmd) {
	if !m.Visible {
		return *m, nil
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return *m, cmd
}

func (m *ApprovalModal) View() string {
	if !m.Visible || m.Request == nil {
		return ""
	}

	modalWidth := m.Width - 10
	if modalWidth > 80 {
		modalWidth = 80
	}
	if modalWidth < 40 {
		modalWidth = 40
	}

	riskStyle := RiskStyle(m.Request.RiskLevel)

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorWarning).Render("⚠ Approval Required")
	opType := fmt.Sprintf("Operation: %s", m.Request.OperationType)
	riskLevel := fmt.Sprintf("Risk: %s", riskStyle.Render(m.Request.RiskLevel))
	summary := ""
	if m.Request.Summary != "" {
		summary = "\n" + lipgloss.NewStyle().Foreground(ColorMuted).Render(m.Request.Summary)
	}

	diffView := m.viewport.View()

	actions := lipgloss.NewStyle().Foreground(ColorInfo).Render("[Y] Approve  [N] Reject  [D] Full Diff  [Esc] Reject")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		opType+"  "+riskLevel,
		summary,
		"",
		diffView,
		"",
		actions,
	)

	modalBox := ModalBoxStyle.Copy().
		Width(modalWidth).
		Render(content)

	overlayHeight := m.Height
	modalHeight := lipgloss.Height(modalBox)
	topPadding := (overlayHeight - modalHeight) / 2
	if topPadding < 0 {
		topPadding = 0
	}

	overlay := ModalOverlayStyle.Copy().
		Width(m.Width).
		Height(overlayHeight).
		Render(lipgloss.NewStyle().Height(topPadding).Render("") + modalBox)

	return overlay
}

func (m *ApprovalModal) buildDiffContent() string {
	if m.Request == nil {
		return ""
	}
	if m.Request.Diff != "" {
		return "Diff:\n" + m.Request.Diff
	}
	if m.Request.CommandPreview != "" {
		return "Command:\n" + m.Request.CommandPreview
	}
	return fmt.Sprintf("Params: %v", m.Request.Params)
}
