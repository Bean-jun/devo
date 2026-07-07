//go:build !windows

package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func programOptions() []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithAltScreen(),
	}
}
