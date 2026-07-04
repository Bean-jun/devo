package components

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary = lipgloss.Color("#A78BFA")
	ColorSuccess = lipgloss.Color("#34D399")
	ColorWarning = lipgloss.Color("#FBBF24")
	ColorDanger  = lipgloss.Color("#F87171")
	ColorInfo    = lipgloss.Color("#60A5FA")
	ColorMuted   = lipgloss.Color("#CBD5E1")
	ColorBg      = lipgloss.Color("#334155")
	ColorSurface = lipgloss.Color("#475569")
	ColorBorder  = lipgloss.Color("#64748B")
	ColorText    = lipgloss.Color("#F8FAFC")
	ColorWhite   = lipgloss.Color("#FFFFFF")
)

var StateColors = map[string]lipgloss.Color{
	"idle":              ColorSuccess,
	"thinking":          ColorInfo,
	"tool_executing":    ColorPrimary,
	"processing":        ColorInfo,
	"awaiting_approval": ColorWarning,
	"paused":            ColorMuted,
	"cancelled":         ColorDanger,
	"completed":         ColorSuccess,
	"archived":          ColorMuted,
}

var (
	StatusBarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorBorder).
			Background(ColorSurface).
			Foreground(ColorWhite).
			Padding(0, 1)

	UserPrefixStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	AssistantPrefixStyle = lipgloss.NewStyle().
				Foreground(ColorInfo).
				Bold(true)

	SystemNoticeStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Italic(true).
				Padding(0, 1).
				Margin(0, 0, 1, 0)

	ToolCardStyle = lipgloss.NewStyle().
			Padding(0, 0, 0, 2).
			Margin(0, 0, 1, 0)

	ToolCardExecutingStyle = lipgloss.NewStyle().
				Padding(0, 0, 0, 2).
				Margin(0, 0, 0, 0)

	ToolCardSuccess = lipgloss.NewStyle().
			Padding(0, 0, 0, 2).
			Margin(0, 0, 0, 0)

	ToolCardError = lipgloss.NewStyle().
			Padding(0, 0, 0, 2).
			Margin(0, 0, 0, 0)

	ToolCardFoldedStyle = lipgloss.NewStyle().
				Padding(0, 0, 0, 2).
				Margin(0, 0, 0, 2)

	ToolCardGroupBorder = lipgloss.NewStyle().
				Padding(0, 0, 0, 1)

	ToolCardSeparator = lipgloss.NewStyle().
				Foreground(ColorBorder)

	ModalOverlayStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#000000CC"))

	ModalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(2, 3).
			Background(ColorSurface)

	SidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, true, false, false).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			Width(30)

	SidebarItemStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Width(27)

	SidebarActiveStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Width(27).
				Foreground(ColorWhite).
				Background(ColorPrimary)

	InputAreaStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(ColorBorder).
			Background(ColorSurface).
			Padding(0, 1)

	ToastErrorStyle = lipgloss.NewStyle().
			Background(ColorDanger).
			Foreground(ColorWhite).
			Padding(1, 2)

	ToastInfoStyle = lipgloss.NewStyle().
			Background(ColorInfo).
			Foreground(ColorWhite).
			Padding(1, 2)

	ApprovalBadgeStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)

	AutoApprovedBadgeStyle = lipgloss.NewStyle().
				Foreground(ColorSuccess).
				Italic(true)

	RiskHighStyle = lipgloss.NewStyle().
			Foreground(ColorDanger).
			Bold(true)

	RiskMediumStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	RiskLowStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	ThinkingStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true)

	ThinkingCollapsedStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)

	YOLOStatusBarStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(ColorWarning).
				Background(ColorWarning).
				Foreground(ColorBg).
				Padding(0, 1)

	YOLOBadgeStyle = lipgloss.NewStyle().
			Background(ColorWarning).
			Foreground(ColorBg).
			Bold(true).
			Padding(0, 1)

	YOLOSmallBadgeStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)
)

func RiskStyle(level string) lipgloss.Style {
	switch level {
	case "high":
		return RiskHighStyle
	case "medium":
		return RiskMediumStyle
	case "low":
		return RiskLowStyle
	default:
		return RiskLowStyle
	}
}

func StateColor(state string) lipgloss.Color {
	if c, ok := StateColors[state]; ok {
		return c
	}
	return ColorMuted
}
