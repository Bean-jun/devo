package tui

import (
	"devo/internal/config"
	"io"
	"log"
	"os"
	"sync"
)

var (
	logger     *log.Logger
	logFile    *os.File
	loggerOnce sync.Once
)

func initLogger() {
	loggerOnce.Do(func() {
		logPath := os.Getenv("DEVO_LOG_PATH")
		if logPath == "" {
			devoDir := config.DevoDir()
			os.MkdirAll(devoDir, 0755)

			logPath = config.LogPath()
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			logger = log.New(io.Discard, "", 0)
			return
		}
		logFile = f

		logger = log.New(f, "", log.LstdFlags)
		logger.Printf("=== Devo TUI session started ===")
		logger.Printf("Log file: %s", logPath)
	})
}

func Log(format string, args ...interface{}) {
	initLogger()
	logger.Printf(format, args...)
}

func RedirectStdLog() {
	initLogger()
	if logFile != nil {
		log.SetOutput(logFile)
	} else {
		log.SetOutput(io.Discard)
	}
}

func LogFilePath() string {
	initLogger()
	if logFile != nil {
		return logFile.Name()
	}
	return ""
}
