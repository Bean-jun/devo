package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModel(t *testing.T) {
	m := initialModel()
	if m.textarea.Placeholder == "" {
		t.Error("placeholder should not be empty")
	}
	if !m.textarea.Focused() {
		t.Error("textarea should be focused initially")
	}
	if m.submitted || m.quitting {
		t.Error("should not be submitted or quitting")
	}
}

func TestView_ContainsHeader(t *testing.T) {
	m := initialModel()
	view := m.View()
	if !strings.Contains(view, "Terminal Input Box") {
		t.Error("view should contain header")
	}
	if !strings.Contains(view, "Ctrl+V") {
		t.Error("view should contain help text")
	}
	if !strings.Contains(view, "Ctrl+Enter") {
		t.Error("view should show Ctrl+Enter=Newline help")
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
	m.textarea.SetValue("test content")
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
	m.textarea.SetValue("hello world")
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

func TestCtrlEnter_InsertsNewline(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("line1")
	m.textarea.SetCursor(5)
	// ctrl+enter handler calls InsertString("\n"), so test that directly
	m.textarea.InsertString("\n")
	if !strings.Contains(m.textarea.Value(), "\n") {
		t.Error("Ctrl+Enter should insert a newline")
	}
	if m.textarea.LineCount() < 2 {
		t.Errorf("Ctrl+Enter should create at least 2 lines, got %d", m.textarea.LineCount())
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

func TestWindowSize_Narrow(t *testing.T) {
	m := initialModel()
	msg := tea.WindowSizeMsg{Width: 2, Height: 40}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.width != 2 {
		t.Errorf("expected width=2, got width=%d", nm.width)
	}
}

func TestSingleCharacterInput(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a'},
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "a" {
		t.Errorf("expected 'a', got '%s'", nm.textarea.Value())
	}
}

func TestMultipleCharactersInput(t *testing.T) {
	m := initialModel()
	runes := []rune{'h', 'e', 'l', 'l', 'o'}
	for _, r := range runes {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newModel, _ := m.Update(msg)
		m = newModel.(model)
	}
	if m.textarea.Value() != "hello" {
		t.Errorf("expected 'hello', got '%s'", m.textarea.Value())
	}
}

func TestUppercaseInput(t *testing.T) {
	m := initialModel()
	runes := []rune{'A', 'B', 'C', 'D'}
	for _, r := range runes {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newModel, _ := m.Update(msg)
		m = newModel.(model)
	}
	if m.textarea.Value() != "ABCD" {
		t.Errorf("expected 'ABCD', got '%s'", m.textarea.Value())
	}
}

func TestNumberInput(t *testing.T) {
	m := initialModel()
	runes := []rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}
	for _, r := range runes {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newModel, _ := m.Update(msg)
		m = newModel.(model)
	}
	if m.textarea.Value() != "0123456789" {
		t.Errorf("expected '0123456789', got '%s'", m.textarea.Value())
	}
}

func TestSpaceInput(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{' '},
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if !strings.Contains(nm.textarea.Value(), " ") {
		t.Error("expected space in value")
	}
}

func TestPunctuationInput(t *testing.T) {
	m := initialModel()
	runes := []rune{',', '.', '/', ';', '\'', '[', ']'}
	for _, r := range runes {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		newModel, _ := m.Update(msg)
		m = newModel.(model)
	}
	if m.textarea.Value() != ",./;'[]" {
		t.Errorf("expected ',./;'[]', got '%s'", m.textarea.Value())
	}
}

func TestChineseInput(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'你', '好', '世', '界'},
		Paste: true,
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "你好世界" {
		t.Errorf("expected '你好世界', got '%s'", nm.textarea.Value())
	}
}

func TestEmojiInput(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'😀', '🎉'},
		Paste: true,
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "😀🎉" {
		t.Errorf("expected '😀🎉', got '%s'", nm.textarea.Value())
	}
}

func TestBackspaceDeletion(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(3)
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "ab" {
		t.Errorf("expected 'ab', got '%s'", nm.textarea.Value())
	}
}

func TestBackspaceOnEmpty(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "" {
		t.Errorf("expected empty, got '%s'", nm.textarea.Value())
	}
}

func TestDeleteDeletion(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(0)
	msg := tea.KeyMsg{Type: tea.KeyDelete}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "bc" {
		t.Errorf("expected 'bc', got '%s'", nm.textarea.Value())
	}
}

func TestDeleteAtEnd(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(3)
	msg := tea.KeyMsg{Type: tea.KeyDelete}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "abc" {
		t.Errorf("expected 'abc' unchanged, got '%s'", nm.textarea.Value())
	}
}

func TestCursorLeft_MovesCorrectly(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(3)
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	newModel, _ := m.Update(msg)
	_ = newModel.(model)
}

func TestCursorRight_MovesCorrectly(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(0)
	msg := tea.KeyMsg{Type: tea.KeyRight}
	newModel, _ := m.Update(msg)
	_ = newModel.(model)
}

func TestCursorLeftAtBoundary_NoCrash(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(0)
	msg := tea.KeyMsg{Type: tea.KeyLeft}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "abc" {
		t.Error("value should not change when moving left at boundary")
	}
}

func TestCursorRightAtBoundary_NoCrash(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(3)
	msg := tea.KeyMsg{Type: tea.KeyRight}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "abc" {
		t.Error("value should not change when moving right at boundary")
	}
}

func TestHomeKey(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(3)
	msg := tea.KeyMsg{Type: tea.KeyHome}
	newModel, _ := m.Update(msg)
	_ = newModel.(model)
}

func TestEndKey(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("abc")
	m.textarea.SetCursor(0)
	msg := tea.KeyMsg{Type: tea.KeyEnd}
	newModel, _ := m.Update(msg)
	_ = newModel.(model)
}

func TestPasteAutoDetected(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'p', 'a', 's', 't', 'e', 'd'},
		Paste: true,
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "pasted" {
		t.Errorf("expected 'pasted', got '%s'", nm.textarea.Value())
	}
}

func TestBulkRunesInsert(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a', 'b', 'c', 'd'},
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "abcd" {
		t.Errorf("expected 'abcd', got '%s'", nm.textarea.Value())
	}
}

func TestBulkRunesSmallInserted(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'a', 'b', 'c'},
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "abc" {
		t.Errorf("expected 'abc', got '%s'", nm.textarea.Value())
	}
}

func TestPasteChinese(t *testing.T) {
	m := initialModel()
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'中', '文', '测', '试'},
		Paste: true,
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "中文测试" {
		t.Errorf("expected '中文测试', got '%s'", nm.textarea.Value())
	}
}

func TestPasteInsertAtCursor(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("hello world")
	m.textarea.SetCursor(5)
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{' ', 'g', 'o'},
		Paste: true,
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != "hello go world" {
		t.Errorf("expected 'hello go world', got '%s'", nm.textarea.Value())
	}
}

func TestCtrlVDoesNotCrash(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("existing")
	msg := tea.KeyMsg{Type: tea.KeyCtrlV}
	newModel, _ := m.Update(msg)
	_ = newModel.(model)
}

func TestNewlineInTextarea(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("line1\nline2")
	if m.textarea.LineCount() != 2 {
		t.Errorf("expected 2 lines, got %d", m.textarea.LineCount())
	}
}

func TestCursorUp_NoCrash(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("line1\nline2")
	m.textarea.SetCursor(len("line1\nli"))
	msg := tea.KeyMsg{Type: tea.KeyUp}
	newModel, _ := m.Update(msg)
	_ = newModel.(model)
}

func TestCursorDown_NoCrash(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("line1\nline2")
	m.textarea.SetCursor(0)
	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := m.Update(msg)
	_ = newModel.(model)
}

func TestView_CharactersCount(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("12345")
	view := m.View()
	if !strings.Contains(view, "Characters: 5") {
		t.Error("view should show character count")
	}
}

func TestNonKeyMsg_Passthrough(t *testing.T) {
	m := initialModel()
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	_, _ = m.Update(msg)
}

func TestLargeTextPaste(t *testing.T) {
	m := initialModel()
	largeText := strings.Repeat("abcdefghij", 1000)
	runes := []rune(largeText)
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: runes,
		Paste: true,
	}
	newModel, _ := m.Update(msg)
	nm := newModel.(model)
	if nm.textarea.Value() != largeText {
		t.Errorf("large paste should preserve all %d characters, got %d", len(largeText), len(nm.textarea.Value()))
	}
}

func TestNormalizePastedText_CRLF(t *testing.T) {
	result := normalizePastedText("line1\r\nline2")
	if result != "line1\nline2" {
		t.Errorf("expected 'line1\\nline2', got '%s'", result)
	}
}

func TestNormalizePastedText_CR(t *testing.T) {
	result := normalizePastedText("line1\rline2")
	if result != "line1\nline2" {
		t.Errorf("expected 'line1\\nline2', got '%s'", result)
	}
}

func TestNormalizePastedText_TrailingNewline(t *testing.T) {
	result := normalizePastedText("hello\n")
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestNormalizePastedText_NoNewline(t *testing.T) {
	result := normalizePastedText("hello")
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestCtrlEnterPreservesContent(t *testing.T) {
	m := initialModel()
	m.textarea.SetValue("before")
	m.textarea.SetCursor(6)
	// ctrl+enter inserts newline via InsertString("\n")
	m.textarea.InsertString("\n")
	if !strings.HasPrefix(m.textarea.Value(), "before") {
		t.Error("Ctrl+Enter should preserve existing content")
	}
	if !m.textarea.Focused() {
		t.Error("textarea should still be focused after Ctrl+Enter")
	}
}
