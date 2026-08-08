package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ─── Update ───

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.Width = m.width

		headerH := 2
		footerH := 6
		vpHeight := m.height - headerH - footerH
		if vpHeight < 5 {
			vpHeight = 5
		}
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
		m.textarea.SetWidth(m.width - 4)
		m.renderer.setWidth(m.width)
		m.toast.Width = m.width

		// 面板使用浮动宽度，不占满终端
		pw := m.overlayPanelWidth()
		m.cmdSheet.Width = pw
		m.sessPicker.Width = pw
		m.approval.Width = pw
		m.approval.Height = m.height
		m.helpPanel.Width = pw
		m.helpPanel.Height = m.height
		m.filesPanel.Width = pw
		m.skillsPanel.Width = pw
		m.mcpPanel.Width = pw
		m.memoryPanel.Width = pw
		m.wsPanel.Width = pw
		m.renameModal.Width = pw
		m.rollback.Width = pw
		m.newSessModal.Width = pw

		m.ready = true
		m.refreshViewport()
		return m, nil

	case tea.KeyMsg:
		if m.overlay.IsOpen() {
			return m.handleOverlayKey(msg)
		}

		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			return m, tea.Quit

		case "esc":
			if !m.overlay.Close() {
			}
			m.refreshViewport()
			return m, nil

		// 跳转到上一条/下一条用户消息（输入框为空时生效）
		case "ctrl+u":
			if m.textarea.Value() == "" {
				m.jumpToPrevUserMessage()
				return m, nil
			}
		case "ctrl+d":
			if m.textarea.Value() == "" {
				m.jumpToNextUserMessage()
				return m, nil
			}

		case "?":
			m.overlay.Open(OverlayHelp)
			m.refreshViewport()
			return m, nil

		case "/":
			if m.textarea.Value() == "" {
				m.overlay.Open(OverlayCommand)
				m.refreshViewport()
				return m, nil
			}

		case "ctrl+y":
			m.statusBar.Yolo = !m.statusBar.Yolo
			m.showToast("YOLO 模式已"+map[bool]string{true: "开启", false: "关闭"}[m.statusBar.Yolo], "info")
			m.refreshViewport()
			return m, nil

		case "ctrl+t":
			toggleTheme()
			m.renderer = newMsgRenderer()
			m.renderer.setWidth(m.width)
			m.refreshViewport()
			m.showToast("主题已切换为 "+currentTheme.Name, "info")
			return m, nil

		case "ctrl+s":
			m.overlay.Open(OverlaySession)
			m.refreshViewport()
			return m, nil

		case "ctrl+n":
			m.showToast("模拟：创建新会话", "success")
			m.refreshViewport()
			return m, nil

		case "ctrl+p":
			m.statusBar.Paused = !m.statusBar.Paused
			m.statusBar.Processing = !m.statusBar.Paused
			if m.statusBar.Paused {
				m.showToast("已暂停", "info")
			} else {
				m.showToast("已恢复", "success")
			}
			m.refreshViewport()
			return m, nil

		case "f2":
			m.showToast("模拟：重命名会话", "info")
			return m, nil

		case "tab":
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == RoleAssistant && len(m.messages[i].ToolCalls) > 0 {
					for j := range m.messages[i].ToolCalls {
						m.messages[i].ToolCalls[j].Expanded = !m.messages[i].ToolCalls[j].Expanded
					}
					m.renderer.cache.invalidate(i)
					m.refreshViewport()
					break
				}
			}
			return m, nil

		case "enter":
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}
			m.textarea.Reset()
			now := time.Now().Format("15:04")
			m.messages = append(m.messages, Message{
				Role: RoleUser, Content: input, Time: now,
			})
			m.statusBar.Processing = true
			m.renderer.cache.invalidate(len(m.messages) - 1)
			m.refreshViewport()
			cmds = append(cmds, mockRespond(input, now))
			return m, tea.Batch(cmds...)
		}

	case mockResponseMsg:
		m.messages = append(m.messages, msg.msg)
		m.statusBar.Processing = false
		m.renderer.cache.invalidate(len(m.messages) - 1)
		m.refreshViewport()
		return m, nil

	case toastTickMsg:
		if m.toast.Duration > 0 {
			m.toast.Duration--
			if m.toast.Duration > 0 {
				cmds = append(cmds, tickToast())
			}
		}
		m.refreshViewport()
		return m, tea.Batch(cmds...)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.textarea, cmd = m.textarea.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ─── 覆盖层按键处理 ───

func (m model) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.overlay.Close()
		m.refreshViewport()
		return m, nil

	case "up", "k":
		switch m.overlay.current {
		case OverlayCommand:
			m.cmdSheet.CursorUp()
		case OverlaySession:
			m.sessPicker.CursorUp()
		case OverlayFiles:
			m.filesPanel.CursorUp()
		case OverlaySkills:
			m.skillsPanel.CursorUp()
		case OverlayMCP:
			m.mcpPanel.CursorUp()
		case OverlayMemory:
			m.memoryPanel.CursorUp()
		case OverlayWorkspace:
			m.wsPanel.CursorUp()
		case OverlayRollback:
			m.rollback.CursorUp()
		}
		return m, nil

	case "down", "j":
		switch m.overlay.current {
		case OverlayCommand:
			m.cmdSheet.CursorDown()
		case OverlaySession:
			m.sessPicker.CursorDown()
		case OverlayFiles:
			m.filesPanel.CursorDown()
		case OverlaySkills:
			m.skillsPanel.CursorDown()
		case OverlayMCP:
			m.mcpPanel.CursorDown()
		case OverlayMemory:
			m.memoryPanel.CursorDown()
		case OverlayWorkspace:
			m.wsPanel.CursorDown()
		case OverlayRollback:
			m.rollback.CursorDown()
		}
		return m, nil

	case "enter":
		switch m.overlay.current {
		case OverlayCommand:
			sel := m.cmdSheet.SelectedCommand()
			m.overlay.Close()
			m.routeCommand(sel.Name)
			return m, nil

		case OverlaySession:
			sel := m.sessPicker.Sessions[m.sessPicker.Selected]
			m.overlay.Close()
			m.statusBar.Session = sel.Name
			m.showToast("已切换到: "+sel.Name, "success")
			m.refreshViewport()
			return m, nil

		case OverlaySkills:
			m.skillsPanel.Toggle()
			sk := m.skillsPanel.Skills[m.skillsPanel.Selected]
			status := "停用"
			if sk.Enabled {
				status = "启用"
			}
			m.showToast(sk.Name+" 已"+status, "success")
			return m, nil

		case OverlayFiles:
			f := m.filesPanel.Files[m.filesPanel.Selected]
			m.showToast("打开文件: "+f.Name, "info")
			return m, nil

		case OverlayMCP:
			srv := m.mcpPanel.Servers[m.mcpPanel.Selected]
			if srv.Status == "connected" {
				m.mcpPanel.Servers[m.mcpPanel.Selected].Status = "disconnected"
				m.showToast(srv.Name+" 已断开", "info")
			} else {
				m.mcpPanel.Servers[m.mcpPanel.Selected].Status = "connected"
				m.showToast(srv.Name+" 已连接", "success")
			}
			return m, nil

		case OverlayWorkspace:
			ws := m.wsPanel.Workspaces[m.wsPanel.Selected]
			for i := range m.wsPanel.Workspaces {
				m.wsPanel.Workspaces[i].Active = false
			}
			m.wsPanel.Workspaces[m.wsPanel.Selected].Active = true
			m.overlay.Close()
			m.showToast("已切换到工作区: "+ws.Name, "success")
			m.refreshViewport()
			return m, nil

		case OverlayNewSession:
			m.overlay.Close()
			m.messages = mockMessages()
			m.renderer = newMsgRenderer()
			m.renderer.setWidth(m.width)
			m.refreshViewport()
			m.showToast("新会话已创建", "success")
			return m, nil

		case OverlayRename:
			if m.renameModal.NewName != "" {
				m.statusBar.Session = m.renameModal.NewName
				m.showToast("已重命名为: "+m.renameModal.NewName, "success")
			}
			m.overlay.Close()
			m.refreshViewport()
			return m, nil

		case OverlayRollback:
			m.overlay.Close()
			m.showToast("模拟回滚到第 "+fmt.Sprintf("%d", m.rollback.Selected+1)+" 条消息", "success")
			m.refreshViewport()
			return m, nil
		}
		return m, nil

	case "y":
		if m.overlay.current == OverlayApproval {
			m.overlay.Close()
			m.showToast("操作已批准", "success")
			m.refreshViewport()
			return m, nil
		}

	case "n":
		if m.overlay.current == OverlayApproval {
			m.overlay.Close()
			m.showToast("操作已拒绝", "error")
			m.refreshViewport()
			return m, nil
		}

	default:
		if m.overlay.current == OverlayRename {
			if len(msg.String()) == 1 {
				m.renameModal.NewName += msg.String()
				return m, nil
			}
			if msg.String() == "backspace" {
				if len(m.renameModal.NewName) > 0 {
					runes := []rune(m.renameModal.NewName)
					m.renameModal.NewName = string(runes[:len(runes)-1])
				}
				return m, nil
			}
			if msg.String() == " " {
				m.renameModal.NewName += " "
				return m, nil
			}
		}
	}

	return m, nil
}

// ─── 命令路由 ───

func (m *model) routeCommand(cmd string) {
	switch cmd {
	case "/new":
		m.overlay.Open(OverlayNewSession)
	case "/switch":
		m.overlay.Open(OverlaySession)
	case "/rename":
		m.renameModal.NewName = ""
		m.overlay.Open(OverlayRename)
	case "/export":
		m.showToast("会话已导出到 ~/devo-exports/demo-session.json", "success")
	case "/rollback":
		m.rollback = NewRollbackPicker(m.messages)
		m.overlay.Open(OverlayRollback)
	case "/files":
		m.overlay.Open(OverlayFiles)
	case "/skills":
		m.overlay.Open(OverlaySkills)
	case "/mcp":
		m.overlay.Open(OverlayMCP)
	case "/memory":
		m.overlay.Open(OverlayMemory)
	case "/workspace":
		m.overlay.Open(OverlayWorkspace)
	case "/w-create":
		m.showToast("模拟：新建工作区已创建", "success")
	case "/yolo":
		m.statusBar.Yolo = !m.statusBar.Yolo
		m.showToast("YOLO 模式已"+map[bool]string{true: "开启", false: "关闭"}[m.statusBar.Yolo], "info")
	case "/theme":
		toggleTheme()
		m.renderer = newMsgRenderer()
		m.renderer.setWidth(m.width)
		m.showToast("主题已切换为 "+currentTheme.Name, "info")
	case "/help":
		m.overlay.Open(OverlayHelp)
	case "/quit":
		m.showToast("请使用 Ctrl+C 或 Ctrl+Q 退出", "info")
	}
	m.refreshViewport()
}
