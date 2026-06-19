package tui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	logger     *log.Logger
	loggerOnce sync.Once
)

func initLogger() {
	loggerOnce.Do(func() {
		logPath := os.Getenv("DEVO_LOG_PATH")
		if logPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				homeDir = os.TempDir()
			}
			devoDir := filepath.Join(homeDir, ".devo")
			os.MkdirAll(devoDir, 0755)

			logPath = filepath.Join(devoDir, "devo.log")
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logger = log.New(os.Stderr, "[devo] ", log.LstdFlags)
			return
		}

		logger = log.New(f, "", log.LstdFlags)
		logger.Printf("=== Devo TUI session started ===")
		logger.Printf("Log file: %s", logPath)

		fmt.Fprintf(os.Stderr, "[devo] Logging to: %s\n", logPath)
	})
}

func Log(format string, args ...interface{}) {
	initLogger()
	logger.Printf(format, args...)
}
