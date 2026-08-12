package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"devo/internal/config"
	"devo/internal/pkg/logging"
	"devo/internal/taskexec/pathsec"
)

var Version = "dev"

func main() {
	subcommand := ""
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		subcommand = os.Args[1]
	}

	switch subcommand {
	case "config":
		if len(os.Args) > 2 && os.Args[2] == "models" {
			runConfigModels(os.Args[3:])
			return
		}
		if len(os.Args) > 2 && os.Args[2] == "onboard" {
			runConfigOnboard()
			return
		}
		fmt.Fprintln(os.Stderr, "用法: devo config models <子命令>")
		fmt.Fprintln(os.Stderr, "      devo config onboard")
		os.Exit(1)
	case "":
		runServer()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", subcommand)
		fmt.Fprintln(os.Stderr, "用法: devo [config]")
		os.Exit(1)
	}
}

func runServer() {
	tuiMode := flag.Bool("tui", false, "Launch TUI mode")
	webMode := flag.Bool("web", false, "Auto-open browser on startup")
	portFlag := flag.Int("port", 0, "Port for web server (0 = auto-assign)")
	workspaceFlag := flag.String("workspace", "", "Default working directory (uses current directory if not set)")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	flag.Parse()

	logging.Init(logging.Config{
		Level:   parseLogLevel(*logLevel),
		LogPath: config.LogPath(),
	})

	logging.Info(context.Background(), "devo starting",
		"version", Version,
		"log_level", *logLevel,
	)

	if *workspaceFlag != "" {
		normalized := pathsec.NormalizePath(*workspaceFlag)
		if err := os.Chdir(normalized); err != nil {
			logging.Error(context.Background(), "failed to change working directory",
				"path", *workspaceFlag,
				"error", err,
			)
			os.Exit(1)
		}
		logging.Info(context.Background(), "working directory set",
			"path", normalized,
		)
	}

	app, err := NewApp(*tuiMode, *webMode, *portFlag)
	if err != nil {
		logging.Error(context.Background(), "failed to create app",
			"error", err,
		)
		os.Exit(1)
	}
	defer app.Shutdown()

	app.Run()
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
