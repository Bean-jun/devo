package main

import "github.com/charmbracelet/lipgloss"

// ─── 主题色（对齐 design_spec.md §2） ───

type Theme struct {
	Name          string
	BgPrimary     lipgloss.Color
	TextPrimary   lipgloss.Color
	TextSecondary lipgloss.Color
	TextTertiary  lipgloss.Color
	Accent        lipgloss.Color
	Success       lipgloss.Color
	Warning       lipgloss.Color
	Error         lipgloss.Color
	Border        lipgloss.Color
}

var Dark = Theme{
	Name:          "dark",
	BgPrimary:     "#0d1117",
	TextPrimary:   "#e6edf3",
	TextSecondary: "#8b949e",
	TextTertiary:  "#6e7681",
	Accent:        "#58a6ff",
	Success:       "#3fb950",
	Warning:       "#d29922",
	Error:         "#f85149",
	Border:        "#30363d",
}

var Light = Theme{
	Name:          "light",
	BgPrimary:     "#ffffff",
	TextPrimary:   "#1f2328",
	TextSecondary: "#656d76",
	TextTertiary:  "#8b949e",
	Accent:        "#0969da",
	Success:       "#1a7f37",
	Warning:       "#9a6700",
	Error:         "#cf222e",
	Border:        "#d0d7de",
}

var currentTheme = Dark

func toggleTheme() {
	if currentTheme.Name == "dark" {
		currentTheme = Light
	} else {
		currentTheme = Dark
	}
}

// ─── 全局样式 ───

var (
	StatusBarBg = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextPrimary).
			Padding(0, 1)
	}

	InputBoxStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(currentTheme.Border).
			Padding(0, 1)
	}
)

// ─── 消息前缀样式（Claude Code 风格） ───

var (
	UserPrefix = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.Accent).Bold(true)
	}

	AsstPrefix = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextPrimary).Bold(true)
	}

	ThinkStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextSecondary).Italic(true)
	}

	SysStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextSecondary).Italic(true)
	}

	TimeStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextSecondary)
	}

	DiffStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.Border)
	}
)

// ─── 工具调用样式 ───

var (
	ToolExec = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(currentTheme.Accent)
	}
	ToolOK = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(currentTheme.Success)
	}
	ToolFail = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(currentTheme.Error)
	}
	ToolWait = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(currentTheme.Warning)
	}
	ToolNameStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().Bold(true)
	}
	ToolDetail = func() lipgloss.Style {
		return lipgloss.NewStyle().Foreground(currentTheme.TextSecondary)
	}
)

// ─── 覆盖层样式 ───

var (
	OverlayStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Background(currentTheme.BgPrimary).
			Foreground(currentTheme.TextPrimary)
	}

	OverlayBoxStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(currentTheme.Border).
			Padding(1, 2)
	}

	OverlayTitleStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.Accent).Bold(true)
	}

	OverlayMutedStyle = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextSecondary)
	}

	// 面板分区样式：头部栏
	PanelHeaderStyle = func(w int) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextPrimary).
			Bold(true).
			Padding(0, 1).
			Width(w)
	}

	// 面板分区样式：底部栏
	PanelFooterStyle = func(w int) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.TextSecondary).
			Padding(0, 1).
			Width(w).
			Align(lipgloss.Center)
	}

	// 面板分隔线
	PanelSeparator = func(w int) lipgloss.Style {
		return lipgloss.NewStyle().
			Foreground(currentTheme.Border).
			Width(w)
	}

	// Toast 样式
	ToastError = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Background(currentTheme.Error).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 2)
	}
	ToastInfo = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Background(currentTheme.Accent).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 2)
	}
	ToastSuccess = func() lipgloss.Style {
		return lipgloss.NewStyle().
			Background(currentTheme.Success).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 2)
	}
)

// ─── 颜色快捷函数 ───

func colorAccent() lipgloss.Color  { return currentTheme.Accent }
func colorSuccess() lipgloss.Color { return currentTheme.Success }
func colorWarning() lipgloss.Color { return currentTheme.Warning }
func colorError() lipgloss.Color   { return currentTheme.Error }
func colorMuted() lipgloss.Color   { return currentTheme.TextSecondary }
func colorBorder() lipgloss.Color  { return currentTheme.Border }
func colorText() lipgloss.Color    { return currentTheme.TextPrimary }
func colorBg() lipgloss.Color      { return currentTheme.BgPrimary }

// dimBgColor 返回遮罩层背景色，比主背景略深以模拟半透明效果
func dimBgColor() lipgloss.Color {
	if currentTheme.Name == "light" {
		return lipgloss.Color("#e8e8ed")
	}
	return lipgloss.Color("#090d13")
}
