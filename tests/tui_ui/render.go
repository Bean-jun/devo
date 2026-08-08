package main

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// ─── 渲染缓存（design_spec.md §9） ───

type renderCache struct {
	cache   []string // 每条消息的渲染结果
	dirty   int      // -1 = 全干净，>=0 = 从该索引起需重渲染
	content string   // 拼接后的完整内容
}

func newRenderCache() *renderCache {
	return &renderCache{dirty: 0}
}

func (rc *renderCache) invalidate(idx int) {
	if rc.dirty < 0 {
		rc.dirty = idx
	} else if idx < rc.dirty {
		rc.dirty = idx
	}
}

func (rc *renderCache) invalidateAll() {
	rc.dirty = 0
}

func (rc *renderCache) isClean() bool {
	return rc.dirty < 0
}

// ─── 消息渲染器 ───

type msgRenderer struct {
	width      int
	mdRenderer *glamour.TermRenderer
	cache      *renderCache
}

func newMsgRenderer() *msgRenderer {
	return &msgRenderer{
		cache: newRenderCache(),
	}
}

func (r *msgRenderer) setWidth(w int) {
	if r.width == w {
		return
	}
	r.width = w
	r.cache.invalidateAll()

	contentW := w - 10
	if contentW < 40 {
		contentW = 40
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(contentW),
	)
	if err == nil {
		r.mdRenderer = renderer
	}
}

func (r *msgRenderer) render(messages []Message) string {
	if r.cache.isClean() {
		return r.cache.content
	}

	if r.cache.dirty >= 0 {
		if r.cache.dirty >= len(r.cache.cache) {
			r.cache.cache = r.cache.cache[:r.cache.dirty]
		} else {
			r.cache.cache = r.cache.cache[:r.cache.dirty]
		}
		for i := r.cache.dirty; i < len(messages); i++ {
			r.cache.cache = append(r.cache.cache, r.renderMessage(messages[i]))
		}
		r.cache.dirty = -1
	}

	var parts []string
	for _, c := range r.cache.cache {
		if c != "" {
			parts = append(parts, c)
		}
	}
	r.cache.content = strings.Join(parts, "\n")
	return r.cache.content
}

func (r *msgRenderer) renderMessage(msg Message) string {
	var b strings.Builder
	b.WriteString("") // 每消息前加空行

	switch msg.Role {
	case RoleUser:
		b.WriteString(r.renderUser(msg))
	case RoleAssistant:
		b.WriteString(r.renderAssistant(msg))
	case RoleSystem:
		b.WriteString(r.renderSystem(msg))
	}
	return b.String()
}

// ─── 用户消息（⏺ 蓝色前缀，左对齐） ───

func (r *msgRenderer) renderUser(msg Message) string {
	prefix := UserPrefix().Render("⏺")
	prefixW := lipgloss.Width(prefix)
	bodyW := r.width - prefixW - 1 // -1 for space between prefix and body
	if bodyW < 10 {
		bodyW = 10
	}

	wrapped := wrapCJK(msg.Content, bodyW)
	lines := strings.Split(wrapped, "\n")

	var b strings.Builder
	b.WriteString(prefix + " " + lines[0])
	indent := strings.Repeat(" ", prefixW+1)
	for _, line := range lines[1:] {
		b.WriteString("\n" + indent + line)
	}

	ts := TimeStyle().Render("  " + msg.Time)
	b.WriteString("\n" + ts)
	return b.String()
}

// ─── 助手消息（⏺ 默认色前缀，左对齐） ───

func (r *msgRenderer) renderAssistant(msg Message) string {
	var b strings.Builder

	prefix := AsstPrefix().Render("⏺")
	prefixW := lipgloss.Width(prefix)
	bodyW := r.width - prefixW - 1
	if bodyW < 10 {
		bodyW = 10
	}

	preWrapped := wrapCJK(msg.Content, bodyW)
	var body string
	if r.mdRenderer != nil {
		rendered, err := r.mdRenderer.Render(preWrapped)
		if err == nil {
			body = strings.TrimSpace(rendered)
		}
	}
	if body == "" {
		body = preWrapped
	}

	lines := strings.Split(body, "\n")
	b.WriteString(prefix + " " + lines[0])
	indent := strings.Repeat(" ", prefixW+1)
	for _, line := range lines[1:] {
		b.WriteString("\n" + indent + line)
	}

	// 思考过程
	if msg.Thinking != "" {
		b.WriteString("\n")
		for _, line := range strings.Split(msg.Thinking, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			b.WriteString(ThinkStyle().Render("  · ") + ThinkStyle().Render(line) + "\n")
		}
	}

	// 工具调用
	if len(msg.ToolCalls) > 0 {
		b.WriteString("\n")
		for _, tc := range msg.ToolCalls {
			b.WriteString(r.renderToolCall(tc))
		}
	}

	// 时间戳
	b.WriteString("\n" + TimeStyle().Render("  "+msg.Time))

	return strings.TrimRight(b.String(), "\n")
}

// ─── 工具调用渲染 ───

func (r *msgRenderer) renderToolCall(tc ToolCall) string {
	var prefix string
	var styleFn func() lipgloss.Style
	switch tc.Status {
	case "success":
		prefix = "✓"
		styleFn = ToolOK
	case "error":
		prefix = "✗"
		styleFn = ToolFail
	case "pending":
		prefix = "⏺"
		styleFn = ToolWait
	default:
		prefix = "⏺"
		styleFn = ToolExec
	}

	line := styleFn().Render("  "+prefix) +
		" " + ToolNameStyle().Render(tc.Name) +
		"  " + ToolDetail().Render(tc.Summary+"  ·  "+tc.Status+"  ·  "+tc.Duration)

	if tc.Diff != "" {
		for _, dl := range strings.Split(tc.Diff, "\n") {
			dl = strings.TrimSpace(dl)
			if dl == "" {
				continue
			}
			line += "\n" + DiffStyle().Render("  │  "+dl)
		}
	}

	return line
}

// ─── 系统消息 ───

func (r *msgRenderer) renderSystem(msg Message) string {
	return SysStyle().Render("  " + msg.Content)
}

// ─── CJK/中英文混排自动换行 ───

func wrapCJK(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return s
	}

	var lines []string
	var curLine []rune
	curWidth := 0

	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, string(curLine))
			curLine = nil
			curWidth = 0
			continue
		}

		chW := runeWidth(ch)
		if curWidth+chW > maxWidth {
			lines = append(lines, string(curLine))
			curLine = nil
			curWidth = 0
		}
		curLine = append(curLine, ch)
		curWidth += chW
	}

	if len(curLine) > 0 {
		lines = append(lines, string(curLine))
	}
	return strings.Join(lines, "\n")
}

func runeWidth(r rune) int {
	if r == 0 {
		return 0
	}
	// CJK 字符（中日韩统一表意文字、全角符号等）
	if unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r) {
		return 2
	}
	// 全角符号
	if r >= 0xFF00 && r <= 0xFFEF {
		return 2
	}
	// 全角标点
	if r >= 0x3000 && r <= 0x303F {
		return 2
	}
	// 全角字母数字
	if r >= 0xFF01 && r <= 0xFF5E {
		return 2
	}
	return 1
}

// displayWidth 计算字符串的显示宽度
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}
