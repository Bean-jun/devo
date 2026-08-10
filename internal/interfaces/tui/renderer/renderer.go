package renderer

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/types"
)

type MsgRenderer struct {
	width      int
	mdRenderer *glamour.TermRenderer
	cache      *RenderCache
	isDark     bool
}

type RenderCache struct {
	cache   []string
	dirty   int
	content string
	count   int
}

func (rc *RenderCache) invalidate(idx int) {
	if rc.dirty < 0 || idx < rc.dirty {
		rc.dirty = idx
	}
}

func (rc *RenderCache) invalidateAll() {
	rc.dirty = 0
	rc.count = 0
}

func (rc *RenderCache) isClean() bool {
	return rc.dirty < 0
}

func New(width int) *MsgRenderer {
	r := &MsgRenderer{
		width:  width,
		cache:  &RenderCache{},
		isDark: components.CurrentTheme.IsDark,
	}
	r.buildMdRenderer()
	return r
}

func (r *MsgRenderer) Width() int {
	return r.width
}

func (r *MsgRenderer) buildMdRenderer() {
	style := "dark"
	if !r.isDark {
		style = "light"
	}
	mdRenderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(0),
	)
	if err == nil {
		r.mdRenderer = mdRenderer
	}
}

func (r *MsgRenderer) SetWidth(w int) {
	if r.width == w {
		return
	}
	r.width = w
	r.cache.invalidateAll()
	r.buildMdRenderer()
}

func (r *MsgRenderer) Render(messages []types.Message) string {
	if r.cache.isClean() && r.cache.count == len(messages) {
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
		r.cache.count = len(messages)
	} else if r.cache.count != len(messages) {
		r.cache.cache = nil
		for i := 0; i < len(messages); i++ {
			r.cache.cache = append(r.cache.cache, r.renderMessage(messages[i]))
		}
		r.cache.count = len(messages)
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

func (r *MsgRenderer) Invalidate(idx int) {
	if r.cache != nil {
		r.cache.invalidate(idx)
	}
}

func (r *MsgRenderer) renderMessage(msg types.Message) string {
	switch msg.Role {
	case "user":
		return r.renderUser(msg)
	case "assistant":
		return r.renderAssistant(msg)
	case "tool":
		return r.renderToolMessage(msg)
	default:
		return r.renderSystem(msg)
	}
}

func (r *MsgRenderer) renderUser(msg types.Message) string {
	prefix := components.UserPrefix().Render("\u25b6")
	prefixW := lipgloss.Width(prefix)
	bodyW := r.width - prefixW - 1
	if bodyW < 10 {
		bodyW = 10
	}

	wrapped := WrapCJK(msg.Content, bodyW)
	lines := strings.Split(wrapped, "\n")

	var b strings.Builder
	b.WriteString(prefix + " " + lines[0])
	indent := strings.Repeat(" ", prefixW+1)
	for _, line := range lines[1:] {
		b.WriteString("\n" + indent + line)
	}

	return b.String()
}

func (r *MsgRenderer) renderAssistant(msg types.Message) string {
	var b strings.Builder

	prefix := components.AsstPrefix().Render("\u25cf")
	prefixW := lipgloss.Width(prefix)
	bodyW := r.width - prefixW - 1
	if bodyW < 10 {
		bodyW = 10
	}

	preWrapped := WrapCJK(msg.Content, bodyW)
	var body string
	if r.mdRenderer != nil {
		rendered, err := r.mdRenderer.Render(preWrapped)
		if err == nil {
			body = stripBlockChars(strings.TrimSpace(rendered))
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

	if msg.Thinking != "" {
		b.WriteString("\n")
		for _, line := range strings.Split(msg.Thinking, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			b.WriteString(components.ThinkStyle().Render("  - ") + components.ThinkStyle().Render(line) + "\n")
		}
	}

	if len(msg.ToolCalls) > 0 {
		b.WriteString("\n")
		for _, tc := range msg.ToolCalls {
			b.WriteString(r.renderToolCall(tc))
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func (r *MsgRenderer) renderToolCall(tc types.ToolCall) string {
	var prefix string
	var styleFn func() lipgloss.Style
	switch tc.Status {
	case "success", "completed":
		prefix = "\u2713"
		styleFn = components.ToolOK
	case "error", "failed":
		prefix = "\u2717"
		styleFn = components.ToolFail
	case "pending", "awaiting_approval":
		prefix = "\u25cb"
		styleFn = components.ToolWait
	default:
		prefix = "\u25cb"
		styleFn = components.ToolExec
	}

	name := tc.ToolName
	if name == "" {
		name = tc.Name
	}
	summary := tc.Summary
	if summary == "" {
		summary = tc.Status
	}
	duration := tc.Duration
	if duration == "" {
		duration = "..."
	}

	line := styleFn().Render("  "+prefix) +
		" " + components.ToolNameStyle().Render(name) +
		"  " + components.ToolDetail().Render(summary+"  -  "+tc.Status+"  -  "+duration)

	if tc.Expanded {
		if tc.Diff != "" {
			for _, dl := range strings.Split(tc.Diff, "\n") {
				dl = strings.TrimSpace(dl)
				if dl == "" {
					continue
				}
				line += "\n" + components.DiffStyle().Render("  \u2502  "+dl)
			}
		}
		if tc.Output != "" {
			line += "\n" + components.DiffStyle().Render("  \u2502  "+strings.Repeat("\u2500", 8)+" output "+strings.Repeat("\u2500", 8))
			for _, ol := range strings.Split(tc.Output, "\n") {
				ol = strings.TrimSpace(ol)
				if ol == "" {
					continue
				}
				line += "\n" + components.DiffStyle().Render("  \u2502  "+ol)
			}
		}
	}

	return line
}

func (r *MsgRenderer) renderToolMessage(msg types.Message) string {
	if len(msg.ToolCalls) == 0 {
		if msg.Content != "" {
			return r.renderSystem(msg)
		}
		return ""
	}
	var b strings.Builder
	for _, tc := range msg.ToolCalls {
		b.WriteString(r.renderToolCall(tc))
	}
	return b.String()
}

func (r *MsgRenderer) renderSystem(msg types.Message) string {
	return components.SysStyle().Render("  " + msg.Content)
}

func (r *MsgRenderer) FindUserMessageYOffsets(messages []types.Message) []int {
	if r.cache == nil || r.cache.cache == nil {
		return nil
	}

	var offsets []int
	lineNum := 0

	for i, msg := range messages {
		if msg.Role == "user" {
			offsets = append(offsets, lineNum)
		}
		if i < len(r.cache.cache) {
			lineNum += strings.Count(r.cache.cache[i], "\n") + 1
		}
	}
	return offsets
}

func WrapCJK(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result []string
	var currentLine strings.Builder
	currentLineLen := 0

	for _, r := range text {
		ch := string(r)
		chLen := 1

		if r == '\n' {
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentLineLen = 0
			continue
		}

		if utf8.RuneLen(r) > 1 {
			chLen = 2
		}

		if currentLineLen+chLen > width {
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentLineLen = 0
		}

		currentLine.WriteString(ch)
		currentLineLen += chLen
	}

	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

	return strings.Join(result, "\n")
}

func (r *MsgRenderer) MessageCountInCache() int {
	if r.cache == nil {
		return 0
	}
	return len(r.cache.cache)
}

func (r *MsgRenderer) GetCacheLineCount(index int) int {
	if r.cache == nil || index >= len(r.cache.cache) {
		return 0
	}
	return strings.Count(r.cache.cache[index], "\n") + 1
}

var reAnsiBg = regexp.MustCompile(`\x1b\[(?:4[0-7]|10[0-7]|48(?:;[0-9]+)*|49)m`)

func stripBlockChars(s string) string {
	s = reAnsiBg.ReplaceAllString(s, "")
	var b strings.Builder
	for _, r := range s {
		if (r >= 0x2500 && r <= 0x27BF) || (r >= 0x2580 && r <= 0x259F) {
			b.WriteRune(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
