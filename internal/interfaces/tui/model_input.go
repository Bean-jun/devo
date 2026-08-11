package tui

import (
	"fmt"
	"strings"

	"devo/internal/interfaces/tui/types"
)

func (m *Model) getActiveSessionState() types.SessionState {
	for _, s := range m.sessions {
		if s.ID == m.activeSessionID {
			return s.State
		}
	}
	return types.SessionStateIdle
}

func (m *Model) pushInputHistory(text string) {
	if text == "" {
		return
	}
	if len(m.inputHistory) > 0 && m.inputHistory[len(m.inputHistory)-1] == text {
		return
	}
	m.inputHistory = append(m.inputHistory, text)
}

func (m *Model) historyPrev(currentValue string) {
	if len(m.inputHistory) == 0 {
		return
	}
	if m.historyIndex == -1 {
		m.historyDraft = currentValue
		m.historyIndex = 0
	} else if m.historyIndex < len(m.inputHistory)-1 {
		m.historyIndex++
	}
	m.textarea.SetValue(m.inputHistory[len(m.inputHistory)-1-m.historyIndex])
	m.textarea.CursorEnd()
}

func (m *Model) historyNext() {
	if m.historyIndex == -1 {
		return
	}
	if m.historyIndex > 0 {
		m.historyIndex--
		m.textarea.SetValue(m.inputHistory[len(m.inputHistory)-1-m.historyIndex])
	} else {
		m.historyIndex = -1
		m.textarea.SetValue(m.historyDraft)
		m.historyDraft = ""
	}
	m.textarea.CursorEnd()
}

func (m *Model) insertNewline() {
	m.textarea.InsertString("\n")
	m.lastTextareaValue = m.textarea.Value()
	m.autoResizeTextarea()
}

func (m *Model) autoExpandPaste() {
	if m.pasteFolded && m.pasteBuffer != "" {
		currentVal := m.textarea.Value()
		expanded := replacePasteMarker(currentVal, m.pasteBuffer)
		m.textarea.SetValue(expanded)
		m.pasteFolded = false
		m.lastTextareaValue = expanded
		m.pasteBuffer = ""
	}
}

func (m *Model) shouldFoldPaste(oldVal, newVal string) bool {
	oldRunes := []rune(oldVal)
	newRunes := []rune(newVal)
	if len(newRunes)-len(oldRunes) > 200 {
		return true
	}
	oldLines := strings.Count(oldVal, "\n")
	newLines := strings.Count(newVal, "\n")
	if newLines-oldLines > 4 {
		return true
	}
	return false
}

func findPasteContent(oldVal, newVal string) (prefix, paste, suffix string) {
	oldRunes := []rune(oldVal)
	newRunes := []rune(newVal)

	i := 0
	for i < len(oldRunes) && i < len(newRunes) && oldRunes[i] == newRunes[i] {
		i++
	}
	prefix = string(newRunes[:i])

	j := 0
	oldLen := len(oldRunes)
	newLen := len(newRunes)
	for j < oldLen-i && j < newLen-i && oldRunes[oldLen-1-j] == newRunes[newLen-1-j] {
		j++
	}
	suffix = string(newRunes[newLen-j:])
	paste = string(newRunes[i : newLen-j])
	return
}

func pasteMarker(charCount, lineCount int) string {
	return fmt.Sprintf("[Pasted text +%d chars, %d lines]", charCount, lineCount)
}

func replacePasteMarker(value, pasteContent string) string {
	start := strings.Index(value, "[Pasted text +")
	if start == -1 {
		return value
	}
	end := strings.Index(value[start:], "]")
	if end == -1 {
		return value
	}
	return value[:start] + pasteContent + value[start+end+1:]
}

func (m *Model) autoResizeTextarea() {
	lines := strings.Count(m.textarea.Value(), "\n") + 1
	taHeight := lines + 1
	if taHeight < 3 {
		taHeight = 3
	}
	if taHeight > 10 {
		taHeight = 10
	}
	m.textarea.SetHeight(taHeight)

	headerH := 3
	footerH := 3 + taHeight
	vpHeight := m.height - headerH - footerH
	if vpHeight < 5 {
		vpHeight = 5
	}
	m.viewport.SetHeight(vpHeight)
}
