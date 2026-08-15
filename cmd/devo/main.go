package main

import (
	"os"

	"devo/internal/cli/commands"
)

var Version = "dev"

func main() {
	commands.SetVersion(Version)
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
