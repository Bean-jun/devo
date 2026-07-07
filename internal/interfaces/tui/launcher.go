package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func Launch(baseURL string, version string) {
	app, err := NewAppWithURL(baseURL, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize TUI: %v\n", err)
		os.Exit(1)
	}

	opts := append([]tea.ProgramOption{tea.WithOutput(os.Stderr)}, programOptions()...)
	p := tea.NewProgram(app, opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}
}
