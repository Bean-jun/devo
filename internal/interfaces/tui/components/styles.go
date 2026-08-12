package components

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

type Theme struct {
	Name          string
	IsDark        bool
	BgPrimary     color.Color
	TextPrimary   color.Color
	TextSecondary color.Color
	TextTertiary  color.Color
	Accent        color.Color
	Success       color.Color
	Warning       color.Color
	Error         color.Color
	Border        color.Color
}

var Dark = Theme{
	Name:          "dark",
	IsDark:        true,
	BgPrimary:     lipgloss.Color("#0d1117"),
	TextPrimary:   lipgloss.Color("#e6edf3"),
	TextSecondary: lipgloss.Color("#8b949e"),
	TextTertiary:  lipgloss.Color("#6e7681"),
	Accent:        lipgloss.Color("#58a6ff"),
	Success:       lipgloss.Color("#3fb950"),
	Warning:       lipgloss.Color("#d29922"),
	Error:         lipgloss.Color("#f85149"),
	Border:        lipgloss.Color("#30363d"),
}

var Light = Theme{
	Name:          "light",
	IsDark:        false,
	BgPrimary:     lipgloss.Color("#ffffff"),
	TextPrimary:   lipgloss.Color("#1f2328"),
	TextSecondary: lipgloss.Color("#656d76"),
	TextTertiary:  lipgloss.Color("#8b949e"),
	Accent:        lipgloss.Color("#0969da"),
	Success:       lipgloss.Color("#1a7f37"),
	Warning:       lipgloss.Color("#9a6700"),
	Error:         lipgloss.Color("#cf222e"),
	Border:        lipgloss.Color("#d0d7de"),
}

var CurrentTheme = Dark

func ToggleTheme() {
	if CurrentTheme.Name == "dark" {
		CurrentTheme = Light
	} else {
		CurrentTheme = Dark
	}
}

func ColorAccent() color.Color  { return CurrentTheme.Accent }
func ColorSuccess() color.Color { return CurrentTheme.Success }
func ColorWarning() color.Color { return CurrentTheme.Warning }
func ColorError() color.Color   { return CurrentTheme.Error }
func ColorMuted() color.Color   { return CurrentTheme.TextSecondary }
func ColorBorder() color.Color  { return CurrentTheme.Border }
func ColorText() color.Color    { return CurrentTheme.TextPrimary }
func ColorBg() color.Color      { return CurrentTheme.BgPrimary }

func DimBgColor() color.Color {
	if CurrentTheme.IsDark {
		return lipgloss.Color("#090d13")
	}
	return lipgloss.Color("#e8e8ed")
}

var (
	StatusBarBg = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextPrimary).
			Padding(0, 1)
	}

	InputBoxStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(0, 1)
	}

	UserPrefix = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Accent).Bold(true)
	}

	AsstPrefix = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextPrimary).Bold(true)
	}

	ThinkStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextSecondary).Italic(true)
	}

	SysStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextSecondary).Italic(true)
	}

	TimeStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextSecondary)
	}

	DiffStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Border)
	}

	ToolExec = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(CurrentTheme.Accent)
	}
	ToolOK = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(CurrentTheme.Success)
	}
	ToolFail = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(CurrentTheme.Error)
	}
	ToolWait = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(CurrentTheme.Warning)
	}
	ToolNameStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Bold(true)
	}
	ToolDetail = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(CurrentTheme.TextSecondary)
	}

	OverlayStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextPrimary)
	}

	OverlayBoxStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(CurrentTheme.Border).
			Padding(1, 2)
	}

	OverlayTitleStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Accent).Bold(true)
	}

	OverlayMutedStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextSecondary)
	}

	PanelHeaderStyle = func(w int) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextPrimary).
			Bold(true).
			Padding(0, 1).
			Width(w)
	}

	PanelFooterStyle = func(w int) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.TextSecondary).
			Padding(0, 1).
			Width(w).
			Align(lipgloss.Center)
	}

	PanelSeparator = func(w int) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Border).
			Width(w)
	}

	ToastError = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Error).
			Padding(0, 2)
	}

	ToastSuccess = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Success).
			Padding(0, 2)
	}

	ToastInfo = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Accent).
			Padding(0, 2)
	}

	ToastUpdate = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(CurrentTheme.Success).
			Bold(true).
			Padding(0, 2)
	}
)
