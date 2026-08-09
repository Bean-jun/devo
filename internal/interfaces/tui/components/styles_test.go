package components

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestToggleTheme(t *testing.T) {
	CurrentTheme = Dark
	if CurrentTheme.Name != "dark" {
		t.Error("默认主题应为 dark")
	}

	ToggleTheme()
	if CurrentTheme.Name != "light" {
		t.Error("ToggleTheme() 应切换到 light")
	}

	ToggleTheme()
	if CurrentTheme.Name != "dark" {
		t.Error("再次 ToggleTheme() 应切回 dark")
	}

	CurrentTheme = Dark
}

func TestColorFunctions(t *testing.T) {
	CurrentTheme = Dark
	defer func() { CurrentTheme = Dark }()

	noColor := lipgloss.NoColor{}

	if ColorAccent() == noColor {
		t.Error("ColorAccent() 不应为空")
	}
	if ColorSuccess() == noColor {
		t.Error("ColorSuccess() 不应为空")
	}
	if ColorError() == noColor {
		t.Error("ColorError() 不应为空")
	}
	if ColorMuted() == noColor {
		t.Error("ColorMuted() 不应为空")
	}
	if ColorBorder() == noColor {
		t.Error("ColorBorder() 不应为空")
	}
	if ColorText() == noColor {
		t.Error("ColorText() 不应为空")
	}
	if ColorBg() == noColor {
		t.Error("ColorBg() 不应为空")
	}
}

func TestDimBgColor(t *testing.T) {
	noColor := lipgloss.NoColor{}

	CurrentTheme = Dark
	if DimBgColor() == noColor {
		t.Error("DimBgColor() 在暗色主题下不应为空")
	}

	CurrentTheme = Light
	if DimBgColor() == noColor {
		t.Error("DimBgColor() 在亮色主题下不应为空")
	}

	CurrentTheme = Dark
}

func TestStyleFunctions(t *testing.T) {
	CurrentTheme = Dark
	defer func() { CurrentTheme = Dark }()

	_ = StatusBarBg()
	_ = UserPrefix()
	_ = AsstPrefix()
	_ = SysStyle()
	_ = OverlayBoxStyle()
	_ = PanelHeaderStyle(40)
	_ = PanelFooterStyle(40)
	_ = ToastError()
	_ = ToastSuccess()
	_ = ToastInfo()
}
