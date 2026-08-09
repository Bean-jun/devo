package tui

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func Launch(baseURL string, version string) {
	model := NewModel(baseURL, version)
	model.applyTermSize()

	opts := append([]tea.ProgramOption{tea.WithOutput(os.Stderr)}, programOptions()...)
	p := tea.NewProgram(&model, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}
}
