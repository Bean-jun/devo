package overlays

import (
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
)

type ReasoningOption struct {
	Value  string
	Label  string
	Effort string
}

var ReasoningOptions = []ReasoningOption{
	{Value: "off", Label: "关闭", Effort: ""},
	{Value: "low", Label: "低", Effort: "low"},
	{Value: "medium", Label: "中", Effort: "medium"},
	{Value: "high", Label: "高", Effort: "high"},
}

type ReasoningPicker struct {
	Width    int
	Height   int
	Selected int
	Enabled  bool
	Effort   string
}

func NewReasoningPicker(enabled bool, effort string) ReasoningPicker {
	rp := ReasoningPicker{
		Enabled: enabled,
		Effort:  effort,
	}
	if !enabled {
		rp.Selected = 0
	} else {
		for i, opt := range ReasoningOptions {
			if opt.Effort == effort {
				rp.Selected = i
				break
			}
		}
	}
	return rp
}

func (rp *ReasoningPicker) CursorUp() {
	if rp.Selected > 0 {
		rp.Selected--
	}
}

func (rp *ReasoningPicker) CursorDown() {
	if rp.Selected < len(ReasoningOptions)-1 {
		rp.Selected++
	}
}

func (rp *ReasoningPicker) SelectedOption() ReasoningOption {
	if rp.Selected >= 0 && rp.Selected < len(ReasoningOptions) {
		return ReasoningOptions[rp.Selected]
	}
	return ReasoningOptions[0]
}

func (rp *ReasoningPicker) Render() string {
	w := rp.Width
	if w < 30 {
		w = 30
	}
	innerW := w - 4

	accent := lipgloss.NewStyle().Foreground(components.ColorAccent())
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" \U0001f9e0 思维链"))

	currentStatus := "关闭"
	if rp.Enabled {
		currentStatus = rp.Effort
	}
	lines = append(lines, " "+muted.Render("当前: "+currentStatus))

	for i, opt := range ReasoningOptions {
		marker := "  "
		nameStyle := muted
		if i == rp.Selected {
			marker = accent.Render("\u25b8 ")
			nameStyle = accent
		}
		line := marker + nameStyle.Render(opt.Label)
		pad := innerW - 2 - lipgloss.Width(line)
		if pad < 1 {
			pad = 1
		}
		lines = append(lines, " "+line+strings.Repeat(" ", pad))
	}

	lines = append(lines, components.PanelFooterStyle(innerW).Render("[\u2191\u2193/Tab] 切换  [Enter] 确认  [Esc] 取消"))
	return strings.Join(lines, "\n")
}
