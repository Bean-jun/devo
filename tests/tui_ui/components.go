package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

// ─── StatusBar (§4.1) ───

type StatusBar struct {
	AppName    string
	Session    string
	Processing bool
	Paused     bool
	Yolo       bool
	Connected  bool
	Width      int
}

func (s *StatusBar) Render() string {
	w := s.Width

	// 左侧：App 名称 + 会话 + 状态 + YOLO
	app := lipgloss.NewStyle().Foreground(colorAccent()).Bold(true).Render("🚀 " + s.AppName)
	divider := lipgloss.NewStyle().Foreground(colorBorder()).Render(" · ")
	session := lipgloss.NewStyle().Foreground(colorText()).Render(s.Session)

	statusColor := colorSuccess()
	statusText := "● idle"
	if s.Processing {
		statusColor = colorAccent()
		statusText = "● Processing"
	}
	if s.Paused {
		statusColor = colorMuted()
		statusText = "● Paused"
	}
	statusDot := lipgloss.NewStyle().Foreground(statusColor).Render(statusText)

	yolo := ""
	if s.Yolo {
		yolo = lipgloss.NewStyle().
			Background(colorWarning()).Foreground(lipgloss.Color("#000000")).Bold(true).
			Render(" YOLO ")
	}

	left := app + divider + session + "  " + statusDot
	if yolo != "" {
		left += "  " + yolo
	}

	// 中间：活动信息
	var center string
	if s.Processing {
		spinner := lipgloss.NewStyle().Foreground(colorAccent()).Render("⏳")
		activity := lipgloss.NewStyle().Foreground(colorMuted()).Render("Processing...")
		center = spinner + " " + activity
	}

	// 右侧：连接状态 + 主题指示
	conn := lipgloss.NewStyle().Foreground(colorSuccess()).Render("✓")
	if !s.Connected {
		conn = lipgloss.NewStyle().Foreground(colorError()).Render("✗")
	}
	themeIcon := lipgloss.NewStyle().Foreground(colorMuted()).Render("🌙")

	right := conn + "  " + themeIcon

	// 三栏布局计算
	leftW := lipgloss.Width(left)
	centerW := lipgloss.Width(center)
	rightW := lipgloss.Width(right)

	var content string
	if centerW > 0 {
		midPad := w - 2 - leftW - centerW - rightW
		if midPad < 2 {
			midPad = 2
		}
		mid := midPad / 2
		content = left + strings.Repeat(" ", mid) + center + strings.Repeat(" ", midPad-mid) + right
	} else {
		pad := w - 2 - leftW - rightW
		if pad < 1 {
			pad = 1
		}
		content = left + strings.Repeat(" ", pad) + right
	}

	headerLine := StatusBarBg().Width(w).Render(content)

	// 底部分隔线
	sep := lipgloss.NewStyle().
		Foreground(colorBorder()).
		Render(strings.Repeat("─", w))

	return headerLine + "\n" + sep
}

// ─── 输入区 InputArea (§4.5) ───

type InputArea struct {
	textarea   textarea.Model
	width      int
	processing bool
}

func NewInputArea() InputArea {
	ta := textarea.New()
	ta.Placeholder = "输入消息..."
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(colorText())
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(colorMuted())
	ta.BlurredStyle = ta.FocusedStyle

	return InputArea{textarea: ta}
}

func (ia *InputArea) SetWidth(w int) {
	ia.width = w
	ia.textarea.SetWidth(w - 6)
}

func (ia *InputArea) SetProcessing(p bool) {
	ia.processing = p
}

func (ia *InputArea) Focus() {
	ia.textarea.Focus()
}

func (ia *InputArea) Blur() {
	ia.textarea.Blur()
}

func (ia *InputArea) Value() string {
	return ia.textarea.Value()
}

func (ia *InputArea) Reset() {
	ia.textarea.Reset()
}

func (ia *InputArea) Update(msg teaMsg) (teaCmd, teaCmd) {
	// simplified - just return nil for the demo
	return nil, nil
}

func (ia *InputArea) Render() string {
	slash := lipgloss.NewStyle().Foreground(colorAccent()).Bold(true).Render("/")
	taView := ia.textarea.View()

	sendIcon := "[⏎]"
	if ia.processing {
		sendIcon = lipgloss.NewStyle().Foreground(colorError()).Render("[■]")
	}

	footer := fmt.Sprintf("Context: 12.5K  ·  Tokens: 3.2K  ·  /home/project  %s", sendIcon)

	inner := slash + " " + taView + "\n" +
		lipgloss.NewStyle().Foreground(colorMuted()).Render(footer)

	return InputBoxStyle().Width(ia.width - 2).Render(inner)
}

// ─── Toast ─── 类型别名（避免循环依赖） ───

type teaMsg = interface{}
type teaCmd = interface{}
