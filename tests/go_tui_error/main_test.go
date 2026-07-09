package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewInputArea(t *testing.T) {
	ia := NewInputArea()
	if ia.textarea.Placeholder == "" {
		t.Error("placeholder should not be empty")
	}
}

func TestInputArea_Focus(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	if !ia.Focused() {
		t.Error("should be focused after Focus()")
	}
}

func TestInputArea_Blur(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.Blur()
	if ia.Focused() {
		t.Error("should not be focused after Blur()")
	}
}

func TestInputArea_Value(t *testing.T) {
	ia := NewInputArea()
	ia.SetValue("hello")
	if ia.Value() != "hello" {
		t.Errorf("expected 'hello', got '%s'", ia.Value())
	}
}

func TestInputArea_Reset(t *testing.T) {
	ia := NewInputArea()
	ia.SetValue("hello")
	ia.Reset()
	if ia.Value() != "" {
		t.Errorf("expected empty after Reset, got '%s'", ia.Value())
	}
}

func TestInputArea_View(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	view := ia.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestInputArea_FooterShowsContextUsage(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	ia.ContextUsage = "context 12.5K"
	view := ia.View()
	if !strings.Contains(view, "context 12.5K") {
		t.Error("view should contain context usage")
	}
}

func TestInputArea_FooterShowsTokenUsage(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	ia.TokenUsage = "Tokens ↑1.2K ↓3.4K (4.6K)"
	view := ia.View()
	if !strings.Contains(view, "Tokens") {
		t.Error("view should contain token usage")
	}
}

func TestInputArea_FooterShowsWorkingDir(t *testing.T) {
	ia := NewInputArea()
	ia.Width = 80
	ia.WorkingDir = "/home/user/project"
	view := ia.View()
	if !strings.Contains(view, "/home/user/project") {
		t.Error("view should contain working directory")
	}
}

func TestInputArea_PasteMessage(t *testing.T) {
	ia := NewInputArea()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h', 'e', 'l', 'l', 'o'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "hello" {
		t.Errorf("expected 'hello', got '%s'", ia.Value())
	}
}

func TestInputArea_BulkRunesInsert(t *testing.T) {
	ia := NewInputArea()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a', 'b', 'c', 'd'},
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "abcd" {
		t.Errorf("expected 'abcd', got '%s'", ia.Value())
	}
}

func TestInputArea_SingleRune(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'x'},
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "x" {
		t.Errorf("expected 'x', got '%s'", ia.Value())
	}
}

func TestInputArea_Backspace(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.SetValue("abc")
	ia.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if ia.Value() != "ab" {
		t.Errorf("expected 'ab', got '%s'", ia.Value())
	}
}

func TestInputArea_BulkRunesNormalizesCRLF(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	runes := []rune{'a', 'b', '\r', '\n', 'c', 'd'}
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: runes,
	}
	_, _ = ia.Update(msg)
	if strings.Contains(ia.Value(), "\r") {
		t.Error("bulk runes should not contain \\r")
	}
	if !strings.Contains(ia.Value(), "ab\ncd") {
		t.Errorf("expected 'ab\\ncd', got '%s'", ia.Value())
	}
}

func TestInputArea_CtrlN_InsertsNewline(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.SetValue("line1")
	msg := tea.KeyMsg{Type: tea.KeyCtrlN}
	_, _ = ia.Update(msg)
	if !strings.Contains(ia.Value(), "\n") {
		t.Error("Ctrl+N should insert a newline")
	}
	if ia.textarea.LineCount() < 2 {
		t.Errorf("Ctrl+N should create at least 2 lines, got %d", ia.textarea.LineCount())
	}
}

func TestInputArea_PasteNormalizesCRLF(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h', 'e', 'l', 'l', 'o', '\r', '\n', 'w', 'o', 'r', 'l', 'd'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if strings.Contains(ia.Value(), "\r") {
		t.Error("pasted text should not contain \\r")
	}
	if !strings.Contains(ia.Value(), "hello\nworld") {
		t.Errorf("expected 'hello\\nworld', got '%s'", ia.Value())
	}
}

func TestInputArea_PasteNormalizesCR(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h', 'e', 'l', 'l', 'o', '\r', 'w', 'o', 'r', 'l', 'd'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if strings.Contains(ia.Value(), "\r") {
		t.Error("pasted text should not contain \\r")
	}
	if !strings.Contains(ia.Value(), "hello\nworld") {
		t.Errorf("expected 'hello\\nworld', got '%s'", ia.Value())
	}
}

func TestNormalizePastedText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "crlf", input: "hello\r\nworld", expected: "hello\nworld"},
		{name: "cr only", input: "hello\rworld", expected: "hello\nworld"},
		{name: "trailing newline", input: "hello\n", expected: "hello"},
		{name: "normal text", input: "hello world", expected: "hello world"},
		{name: "mixed", input: "line1\r\nline2\rline3", expected: "line1\nline2\nline3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePastedText(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestInputArea_ChineseInput(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'你', '好', '世', '界'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "你好世界" {
		t.Errorf("expected '你好世界', got '%s'", ia.Value())
	}
}

func TestInputArea_ChineseSingleRune(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'中'},
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "中" {
		t.Errorf("expected '中', got '%s'", ia.Value())
	}
}

func TestInputArea_MixedChineseEnglish(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	runes := []rune{'h', 'e', 'l', 'l', 'o', '你', '好'}
	for _, r := range runes {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		_, _ = ia.Update(msg)
	}
	if ia.Value() != "hello你好" {
		t.Errorf("expected 'hello你好', got '%s'", ia.Value())
	}
}

func TestInputArea_ChineseBackspace(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	ia.SetValue("你好世界")
	ia.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if ia.Value() != "你好世" {
		t.Errorf("expected '你好世', got '%s'", ia.Value())
	}
}

func TestInputArea_ChinesePasteWithCRLF(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'你', '好', '\r', '\n', '世', '界'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if strings.Contains(ia.Value(), "\r") {
		t.Error("Chinese pasted text should not contain \\r")
	}
	if !strings.Contains(ia.Value(), "你好\n世界") {
		t.Errorf("expected '你好\\n世界', got '%s'", ia.Value())
	}
}

func TestInitialModel(t *testing.T) {
	m := initialModel()
	if m.chatView.InputArea.textarea.Placeholder == "" {
		t.Error("placeholder should not be empty")
	}
	if !m.chatView.InputArea.Focused() {
		t.Error("inputArea should be focused initially")
	}
	if m.submitted || m.quitting {
		t.Error("should not be submitted or quitting")
	}
}

func TestView_ContainsHelp(t *testing.T) {
	m := initialModel()
	view := m.View()
	if !strings.Contains(view, "Ctrl+V") {
		t.Error("view should contain help text")
	}
}

func TestView_Quitting(t *testing.T) {
	m := initialModel()
	m.quitting = true
	view := m.View()
	if !strings.Contains(view, "Goodbye!") {
		t.Error("view should show goodbye message")
	}
}

func TestView_Submitted(t *testing.T) {
	m := initialModel()
	m.lastValue = "test content"
	m.submitted = true
	view := m.View()
	if !strings.Contains(view, "Submitted Content") {
		t.Error("view should show submitted header")
	}
	if !strings.Contains(view, "test content") {
		t.Error("view should show submitted content")
	}
}

func TestView_SubmittedEmpty(t *testing.T) {
	m := initialModel()
	m.submitted = true
	view := m.View()
	if !strings.Contains(view, "(empty)") {
		t.Error("view should show (empty) for empty submission")
	}
}

func TestInit_ReturnsBlinkCmd(t *testing.T) {
	m := initialModel()
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return a non-nil command")
	}
}

func TestQuit_Esc(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd := m.Update(msg)
	nm := newModel.(model)
	if !nm.quitting {
		t.Error("model should be quitting after Esc")
	}
	if cmd == nil {
		t.Error("Esc should return Quit command")
	}
}

func TestQuit_CtrlC(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := m.Update(msg)
	nm := newModel.(model)
	if !nm.quitting {
		t.Error("model should be quitting after Ctrl+C")
	}
	if cmd == nil {
		t.Error("Ctrl+C should return Quit command")
	}
}

func TestSubmit_Enter(t *testing.T) {
	m := initialModel()
	m.chatView.InputArea.SetValue("hello world")
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	nm := newModel.(model)
	if !nm.submitted {
		t.Error("model should be submitted after Enter")
	}
	if cmd == nil {
		t.Error("Enter should return Quit command")
	}
}

func TestSubmit_EnterEmpty(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, cmd := m.Update(msg)
	nm := newModel.(model)
	if !nm.submitted {
		t.Error("model should be submitted even with empty content")
	}
	if cmd == nil {
		t.Error("Enter should return Quit command")
	}
}

func TestWindowSize(t *testing.T) {
	m := initialModel()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.width != 120 || nm.height != 40 {
		t.Errorf("expected width=120, height=40, got width=%d, height=%d", nm.width, nm.height)
	}
}

func TestDebugInfo(t *testing.T) {
	info := buildDebugInfo()
	if info == "" {
		t.Error("debug info should not be empty")
	}
	if !strings.Contains(info, "OS:") {
		t.Error("debug info should contain OS")
	}
}

func TestView_ContainsDebugInfo(t *testing.T) {
	m := initialModel()
	view := m.View()
	if !strings.Contains(view, "Environment Debug Info") {
		t.Error("view should contain debug info section")
	}
	if !strings.Contains(view, "OS:") {
		t.Error("view should contain OS info")
	}
}

func TestView_ContainsCharCount(t *testing.T) {
	m := initialModel()
	m.chatView.InputArea.SetValue("你好世界")
	view := m.View()
	if !strings.Contains(view, "Input chars:") {
		t.Error("view should contain character count")
	}
}

func TestInputArea_EmojiInput(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'😀', '🎉', '✅'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "😀🎉✅" {
		t.Errorf("expected '😀🎉✅', got '%s'", ia.Value())
	}
}

func TestInputArea_JapaneseInput(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'こ', 'ん', 'に', 'ち', 'は'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "こんにちは" {
		t.Errorf("expected 'こんにちは', got '%s'", ia.Value())
	}
}

func TestInputArea_KoreanInput(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'안', '녕', '하', '세', '요'},
		Paste: true,
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "안녕하세요" {
		t.Errorf("expected '안녕하세요', got '%s'", ia.Value())
	}
}

func TestChatView_Composition(t *testing.T) {
	cv := NewChatView()
	cv.SetSize(80, 24)
	view := cv.View()
	if view == "" {
		t.Error("ChatView should not return empty view")
	}
	if !strings.Contains(view, "Type / to see commands") {
		t.Error("ChatView should contain empty state banner")
	}
}

func TestChatView_CompositionWithChineseInput(t *testing.T) {
	cv := NewChatView()
	cv.SetSize(80, 24)
	cv.InputArea.SetValue("测试中文输入")
	view := cv.View()
	if !strings.Contains(view, "测试中文输入") {
		t.Error("ChatView should display Chinese input content")
	}
}

func TestChatView_CompositionWithMixedContent(t *testing.T) {
	cv := NewChatView()
	cv.SetSize(80, 24)
	cv.InputArea.SetValue("hello 你好 world 世界")
	view := cv.View()
	if !strings.Contains(view, "hello 你好 world 世界") {
		t.Error("ChatView should display mixed Chinese/English content")
	}
}

func TestChatView_FooterInComposition(t *testing.T) {
	cv := NewChatView()
	cv.SetSize(80, 24)
	cv.InputArea.ContextUsage = "context 12.5K"
	cv.InputArea.TokenUsage = "Tokens ↑1.2K ↓3.4K"
	cv.InputArea.WorkingDir = "/home/user/project"
	view := cv.View()
	if !strings.Contains(view, "context 12.5K") {
		t.Error("ChatView should show context usage in footer")
	}
	if !strings.Contains(view, "/home/user/project") {
		t.Error("ChatView should show working directory in footer")
	}
}

func TestChatView_ProcessingMode(t *testing.T) {
	cv := NewChatView()
	cv.SetSize(80, 24)
	cv.Processing = true
	view := cv.View()
	if !strings.Contains(view, "Processing...") {
		t.Error("ChatView should show Processing... when Processing is true")
	}
}

func TestMessageViewport_New(t *testing.T) {
	mv := NewMessageViewport()
	view := mv.View()
	if view == "" {
		t.Error("MessageViewport view should not be empty")
	}
}

func TestMessageViewport_AddMessage(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	mv.AddMessage(Message{Role: "user", Content: "你好世界"})
	mv.AddMessage(Message{Role: "assistant", Content: "欢迎使用 Devo"})
	mv.Refresh()
	view := mv.View()
	if !strings.Contains(view, "你好世界") {
		t.Error("MessageViewport should contain user message content")
	}
	if !strings.Contains(view, "欢迎使用") {
		t.Error("MessageViewport should contain assistant message content")
	}
}

func TestMessageViewport_AddToolCard(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	mv.AddToolCard(ToolCardData{
		ToolName: "read_file",
		Params:   `{"path": "/home/main.go"}`,
		Result:   "中国語テスト",
		Success:  true,
		Duration: "0.5s",
		Expanded: true,
	})
	mv.Refresh()
	view := mv.View()
	if !strings.Contains(view, "read_file") {
		t.Error("MessageViewport should contain tool card name")
	}
	if !strings.Contains(view, "中国語テスト") {
		t.Error("MessageViewport should contain tool card result with Chinese")
	}
}

func TestMessageViewport_SetMessages(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	mv.SetMessages([]Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
	})
	view := mv.View()
	if !strings.Contains(view, "msg1") {
		t.Error("MessageViewport should contain msg1")
	}
	if !strings.Contains(view, "msg2") {
		t.Error("MessageViewport should contain msg2")
	}
}

func TestMessageViewport_EmptyState(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	view := mv.View()
	if !strings.Contains(view, "Type / to see commands") {
		t.Error("Empty state should show banner with instructions")
	}
}

func TestMessageViewport_SystemMessage(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	mv.AddMessage(Message{Role: "system", Content: "系统通知：操作成功"})
	mv.Refresh()
	view := mv.View()
	if !strings.Contains(view, "系统通知") {
		t.Error("MessageViewport should contain system message with Chinese")
	}
}

func TestMessageViewport_ToolMessage(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	mv.AddMessage(Message{Role: "tool", Content: "工具执行结果"})
	mv.Refresh()
	view := mv.View()
	if !strings.Contains(view, "[Tool Result]") {
		t.Error("MessageViewport should show tool result header")
	}
}

func TestMessageViewport_MultipleToolCards(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	mv.AddToolCard(ToolCardData{
		ToolName: "read_file",
		Result:   "result1",
		Success:  true,
		Expanded: false,
	})
	mv.AddToolCard(ToolCardData{
		ToolName: "write_file",
		Result:   "result2",
		Success:  true,
		Expanded: false,
	})
	mv.Refresh()
	view := mv.View()
	if !strings.Contains(view, "read_file") {
		t.Error("MessageViewport should contain first tool card")
	}
	if !strings.Contains(view, "write_file") {
		t.Error("MessageViewport should contain second tool card")
	}
}

func TestMessageViewport_StreamingContent(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 20)
	mv.StreamingActive = true
	mv.StreamingBuffer.WriteString("流式输出中...")
	mv.Refresh()
	view := mv.View()
	if !strings.Contains(view, "流式输出中") {
		t.Error("MessageViewport should show streaming content")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{name: "short", input: "hello", maxLen: 10, expected: "hello"},
		{name: "exact", input: "hello", maxLen: 5, expected: "hello"},
		{name: "long", input: "hello world", maxLen: 8, expected: "hello w…"},
		{name: "chinese short", input: "你好世界", maxLen: 20, expected: "你好世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestModel_ContainsMessagesAndToolCards(t *testing.T) {
	m := initialModel()
	view := m.View()
	if !strings.Contains(view, "read_file") {
		t.Error("ChatView should contain tool card")
	}
	if !strings.Contains(view, "中国語のテスト") {
		t.Error("ChatView should contain tool card with CJK result")
	}
	if !strings.Contains(view, "exec_python") {
		t.Error("ChatView should contain second tool card")
	}
	if !strings.Contains(view, "中文乱码") {
		t.Error("ChatView should contain last assistant message about encoding")
	}
}

func TestIsOSCLeak(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "bg color with bracket", input: "]11;rgb:0000/0000/0000", expected: true},
		{name: "bg color no bracket", input: "11;rgb:0000/0000/0000", expected: true},
		{name: "fg color no bracket", input: "10;rgb:aaaa/bbbb/cccc", expected: true},
		{name: "palette no bracket", input: "4;0;rgb:1111/2222/3333", expected: true},
		{name: "partial osc leak", input: "11;rgb:0000/00", expected: true},
		{name: "normal text", input: "hello world", expected: false},
		{name: "bracket but not OSC", input: "]hello", expected: false},
		{name: "short text", input: "11;r", expected: false},
		{name: "empty", input: "", expected: false},
		{name: "chinese text", input: "你好世界", expected: false},
		{name: "code block", input: "[]", expected: false},
		{name: "numbers only", input: "12345", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOSCLeak(tt.input)
			if result != tt.expected {
				t.Errorf("isOSCLeak(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInputArea_OSCLeakFiltered(t *testing.T) {
	tests := []struct {
		name  string
		runes []rune
	}{
		{name: "OSC background color with bracket", runes: []rune("]11;rgb:2727/2828/2222")},
		{name: "OSC foreground color with bracket", runes: []rune("]10;rgb:aaaa/bbbb/cccc")},
		{name: "OSC palette with bracket", runes: []rune("]4;0;rgb:1111/2222/3333")},
		{name: "OSC background color no bracket", runes: []rune("11;rgb:0000/0000/0000")},
		{name: "partial OSC leak", runes: []rune("11;rgb:0000/00")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ia := NewInputArea()
			ia.Focus()
			msg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: tt.runes,
			}
			_, _ = ia.Update(msg)
			if ia.Value() != "" {
				t.Errorf("OSC leak should be filtered, got '%s'", ia.Value())
			}
		})
	}
}

func TestInputArea_OSCLeakSingleRuneReset(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()

	ia.SetValue("]11;rgb:0000/0000/0000")

	ia.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if strings.Contains(ia.Value(), "rgb:") {
		t.Errorf("OSC leak should be cleared after next update, got '%s'", ia.Value())
	}
}

func TestInputArea_OSCLeakSingleRuneAccumulation(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()

	leakChars := []rune("11;rgb:0000/0000/0000")
	for _, r := range leakChars {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		_, _ = ia.Update(msg)
	}

	if strings.Contains(ia.Value(), "rgb:") {
		t.Errorf("OSC leak accumulated from single runes should be cleared, got '%s'", ia.Value())
	}
}

func TestInputArea_OSCLeakNoBracketAccumulation(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()

	leakChars := []rune("11;rgb:0000/00")
	for _, r := range leakChars {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		_, _ = ia.Update(msg)
	}

	if strings.Contains(ia.Value(), "rgb:") {
		t.Errorf("OSC leak (no bracket) accumulated from single runes should be cleared, got '%s'", ia.Value())
	}
}

func TestInputArea_NormalBracketNotFiltered(t *testing.T) {
	ia := NewInputArea()
	ia.Focus()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{']', 'h', 'e', 'l', 'l', 'o'},
	}
	_, _ = ia.Update(msg)
	if ia.Value() != "]hello" {
		t.Errorf("expected ']hello', got '%s'", ia.Value())
	}
}
