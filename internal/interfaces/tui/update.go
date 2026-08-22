package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/overlays"
	"devo/internal/interfaces/tui/renderer"
	"devo/internal/interfaces/tui/types"
)

type resizeTickMsg struct{}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return types.TickMsg(t)
		}),
		tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return resizeTickMsg{}
		}),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var taCmd tea.Cmd

	switch msg := msg.(type) {
	case apiResponseMsg:
		return m.handleAPIResponse(msg)

	case sseEventMsg:
		return m.handleSSEEvent(msg)

	case resizeTickMsg:
		m.applyTermSize()
		return m, tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
			return resizeTickMsg{}
		})

	case tea.WindowSizeMsg:
		if !m.initialized {
			m.initialized = true
			m.applyTermSize()
			if m.width == 80 && m.height == 24 {
				m.applySize(msg.Width, msg.Height)
			}
			return m, m.initFromAPI()
		}
		m.applySize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyboardEnhancementsMsg:
		m.keyboardEnhancements = msg.Flags
		return m, nil

	case tea.PasteMsg:
		if m.overlay.IsOpen() {
			return m, nil
		}

		m.autoExpandPaste()

		currentVal := m.textarea.Value()
		oldLen := len([]rune(currentVal))

		m.textarea, taCmd = m.textarea.Update(msg)
		cmds = append(cmds, taCmd)

		newVal := m.textarea.Value()
		newLen := len([]rune(newVal))
		addedLen := newLen - oldLen
		addedLines := strings.Count(newVal, "\n") - strings.Count(currentVal, "\n")

		if addedLen > 200 || addedLines > 4 {
			prefix, paste, suffix := findPasteContent(currentVal, newVal)
			m.pasteBuffer = paste
			m.pasteFolded = true
			charCount := len([]rune(paste))
			lineCount := strings.Count(paste, "\n") + 1
			marker := pasteMarker(charCount, lineCount)
			m.textarea.SetValue(prefix + marker + suffix)
		}
		m.lastTextareaValue = m.textarea.Value()
		m.autoResizeTextarea()
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		if m.overlay.IsOpen() {
			return m.handleOverlayKey(msg)
		}

		key := msg.String()
		keyCode := msg.Key().Code
		keyMod := msg.Key().Mod

		switch {
		case key == "ctrl+c" || key == "ctrl+q":
			return m, tea.Quit

		case key == "esc" || keyCode == tea.KeyEsc:
			return m.handleEscNoOverlay()

		case key == "ctrl+u":
			if m.textarea.Value() == "" {
				m.jumpToPrevUserMessage()
				return m, nil
			}

		case key == "ctrl+d":
			if m.textarea.Value() == "" {
				m.jumpToNextUserMessage()
				return m, nil
			}

		case key == "ctrl+b":
			m.viewport.GotoBottom()
			return m, nil

		case key == "?":
			m.overlay.Open(overlays.OverlayHelp)
			m.refreshViewport()
			return m, nil

		case key == "/":
			if m.textarea.Value() == "" {
				m.overlay.Open(overlays.OverlayCommand)
				m.refreshViewport()
				return m, nil
			}

		case key == "ctrl+y":
			m.statusBar.Yolo = !m.statusBar.Yolo
			yoloMsg := "YOLO 模式已开启"
			if !m.statusBar.Yolo {
				yoloMsg = "YOLO 模式已关闭"
			}
			m.toast.Show(yoloMsg, false)
			m.refreshViewport()
			if m.activeSessionID != "" {
				level := "normal"
				if m.statusBar.Yolo {
					level = "elevated"
				}
				return m, m.setTrustLevelFromAPI(m.activeSessionID, level)
			}
			return m, nil

		case key == "ctrl+e":
			m.statusBar.TeamMode = !m.statusBar.TeamMode
			m.settingsPanel.TeamMode = m.statusBar.TeamMode
			teamMsg := "Team Mode 已开启"
			if !m.statusBar.TeamMode {
				teamMsg = "Team Mode 已关闭"
			}
			m.toast.Show(teamMsg, false)
			m.refreshViewport()
			return m, m.setTeamModeFromAPI(m.statusBar.TeamMode)

		case key == "ctrl+t":
			components.ToggleTheme()
			m.renderer = renderer.New(m.width)
			m.refreshViewport()
			m.toast.Show("主题已切换为 "+components.CurrentTheme.Name, false)
			return m, nil

		case key == "ctrl+s":
			m.sessPicker.Sessions = m.sessions
			m.overlay.Open(overlays.OverlaySession)
			return m, m.fetchSessionsFromAPI()

		case key == "ctrl+n":
			m.overlay.Open(overlays.OverlayNewSession)
			return m, m.fetchAgentsFromAPI()

		case key == "ctrl+p":
			m.statusBar.Paused = !m.statusBar.Paused
			m.statusBar.Processing = !m.statusBar.Paused
			if m.statusBar.Paused {
				m.toast.Show("已暂停", false)
			} else {
				m.toast.Show("已恢复", false)
			}
			m.refreshViewport()
			return m, nil

		case key == "f2":
			m.overlay.Open(overlays.OverlayRename)
			return m, nil

		case keyMod.Contains(tea.ModCtrl) && keyCode == tea.KeyTab,
			keyCode == tea.KeyF3:
			img, ok := getImageFromClipboard()
			if ok {
				m.pastedImages = append(m.pastedImages, img)
				m.toast.Show(fmt.Sprintf("图片已附加 (%d 张)", len(m.pastedImages)), false)
			} else {
				m.toast.Show("剪贴板中没有图片", true)
			}
			return m, nil

		case key == "tab":
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == "assistant" && len(m.messages[i].ToolCalls) > 0 {
					for j := range m.messages[i].ToolCalls {
						m.messages[i].ToolCalls[j].Expanded = !m.messages[i].ToolCalls[j].Expanded
					}
					m.renderer.Invalidate(i)
					m.refreshViewport()
					break
				}
			}
			return m, nil

		case (keyMod.Contains(tea.ModCtrl) && keyCode == tea.KeyEnter) || (keyMod.Contains(tea.ModCtrl) && keyMod.Contains(tea.ModShift) && keyCode == tea.KeyEnter):
			m.insertNewline()
			return m, m.textarea.Focus()

		case key == "ctrl+up" || (keyMod.Contains(tea.ModCtrl) && keyCode == tea.KeyUp):
			m.historyPrev(m.textarea.Value())
			m.lastTextareaValue = m.textarea.Value()
			return m, nil

		case key == "ctrl+down" || (keyMod.Contains(tea.ModCtrl) && keyCode == tea.KeyDown):
			m.historyNext()
			m.lastTextareaValue = m.textarea.Value()
			return m, nil

		case key == "enter":
			content := m.resolvePasteBuffer(m.textarea.Value())
			images := m.pastedImages
			if content == "" && len(images) == 0 {
				return m, nil
			}

			m.pushInputHistory(content)
			m.historyIndex = -1
			m.historyDraft = ""
			m.pasteBuffer = ""
			m.pastedImages = nil

			userMsg := types.Message{
				Role:    "user",
				Content: content,
			}
			m.messages = append(m.messages, userMsg)
			m.renderer.Invalidate(len(m.messages) - 1)
			m.textarea.Reset()
			m.lastTextareaValue = ""
			m.statusBar.Processing = true
			m.refreshViewportToBottom()

			if m.activeSessionID != "" {
				cmds = append(cmds, m.sendMessageToAPI(m.activeSessionID, content, images))
			} else {
				assistantMsg := types.Message{
					Role:    "assistant",
					Content: "未连接到会话。请先创建或切换会话。",
				}
				m.messages = append(m.messages, assistantMsg)
				m.statusBar.Processing = false
				m.refreshViewportToBottom()
			}
			cmds = append(cmds, m.textarea.Focus())
			return m, tea.Batch(cmds...)
		}

	case types.TickMsg:
		if m.toast.Duration > 0 {
			m.toast.Tick()
		}
		return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return types.TickMsg(t)
		})
	}

	m.textarea, taCmd = m.textarea.Update(msg)
	cmds = append(cmds, taCmd)

	currentVal := m.textarea.Value()
	if m.lastTextareaValue != "" && m.shouldFoldPaste(m.lastTextareaValue, currentVal) {
		prefix, paste, suffix := findPasteContent(m.lastTextareaValue, currentVal)
		m.pasteBuffer = paste
		m.pasteFolded = true
		charCount := len([]rune(paste))
		lineCount := strings.Count(paste, "\n") + 1
		marker := pasteMarker(charCount, lineCount)
		m.textarea.SetValue(prefix + marker + suffix)
	}
	m.lastTextareaValue = m.textarea.Value()
	m.autoResizeTextarea()

	var vpCmd tea.Cmd
	// Only pass to viewport if not a printable character (j/k vim keys cause unwanted scroll)
	passToViewport := true
	if kp, isKey := msg.(tea.KeyPressMsg); isKey {
		code := kp.Key().Code
		if code >= ' ' && code <= '~' {
			passToViewport = false
		}
	}
	if passToViewport {
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleEscNoOverlay() (tea.Model, tea.Cmd) {
	state := m.getActiveSessionState()

	switch state {
	case types.SessionStateToolExecuting:
		if m.activeSessionID != "" {
			m.toast.Show("已暂停工具执行", false)
			m.statusBar.Paused = true
			m.statusBar.Processing = false
			m.refreshViewport()
			return m, m.pauseSessionFromAPI(m.activeSessionID)
		}
		return m, nil

	case types.SessionStatePaused:
		if m.activeSessionID != "" {
			m.toast.Show("已取消会话", false)
			m.statusBar.Paused = false
			m.statusBar.Processing = false
			m.refreshViewport()
			return m, m.cancelSessionFromAPI(m.activeSessionID)
		}
		return m, nil

	case types.SessionStateThinking, types.SessionStateProcessing, types.SessionStateAwaitingApproval:
		if m.activeSessionID != "" {
			m.toast.Show("已取消", false)
			m.statusBar.Processing = false
			m.refreshViewport()
			return m, m.cancelSessionFromAPI(m.activeSessionID)
		}
		return m, nil

	case types.SessionStateIdle, types.SessionStateCancelled, types.SessionStateCompleted:
		now := time.Now()
		if now.Sub(m.lastEscAt) < 500*time.Millisecond {
			m.lastEscAt = time.Time{}
			var items []overlays.RollbackItem
			for i := len(m.messages) - 1; i >= 0; i-- {
				msg := m.messages[i]
				if msg.Role == "user" {
					items = append(items, overlays.RollbackItem{
						Content:  msg.Content,
						Role:     msg.Role,
						Time:     formatRollbackTime(msg.CreatedAt),
						MsgIndex: i,
					})
				}
			}
			picker := overlays.NewRollbackPicker(items)
			picker.TotalMessages = len(m.messages)
			m.rollback = picker
			m.overlay.Open(overlays.OverlayRollback)
			m.refreshViewport()
			return m, nil
		}
		m.lastEscAt = now
		m.toast.Show("再按一次 ESC 打开回滚", false)
		return m, nil

	default:
		return m, nil
	}
}

func (m *Model) resolvePasteBuffer(value string) string {
	if m.pasteBuffer != "" {
		return replacePasteMarker(value, m.pasteBuffer)
	}
	return value
}

func truncateActivity(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if len([]rune(s)) > 40 {
		return string([]rune(s)[:40]) + "..."
	}
	return s
}
