package main

import (
	"flag"
	"log"
	"os"

	"devo/internal/taskexec/pathsec"
)

var Version = "dev"

func main() {
	tuiMode := flag.Bool("tui", false, "Launch TUI mode")
	webMode := flag.Bool("web", false, "Auto-open browser on startup")
	portFlag := flag.Int("port", 0, "Port for web server (0 = auto-assign)")
	workspaceFlag := flag.String("workspace", "", "Default working directory (uses current directory if not set)")
	flag.Parse()

	if *workspaceFlag != "" {
		normalized := pathsec.NormalizePath(*workspaceFlag)
		if err := os.Chdir(normalized); err != nil {
			log.Fatalf("[devo] Failed to change working directory to %s: %v", *workspaceFlag, err)
		}
		log.Printf("[devo] Working directory set to: %s", normalized)
	}

	app, err := NewApp(*tuiMode, *webMode, *portFlag)
	if err != nil {
		log.Fatalf("[devo] %v", err)
	}
	defer app.Shutdown()

	app.Run()
}
