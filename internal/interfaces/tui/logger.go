package tui

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"sync"

	"devo/internal/config"
	"devo/internal/pkg/logging"
)

var (
	tuiLogFile    *os.File
	tuiLoggerOnce sync.Once
	tuiSlogMu     sync.Mutex
)

func initLogger() {
	tuiLoggerOnce.Do(func() {
		logPath := os.Getenv("DEVO_LOG_PATH")
		if logPath == "" {
			devoDir := config.DevoDir()
			os.MkdirAll(devoDir, 0755)
			logPath = config.LogPath()
		}
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return
		}
		tuiLogFile = f

		logging.Info(context.Background(), "tui session started",
			"log_file", logPath,
		)
	})
}

func Log(format string, args ...interface{}) {
	initLogger()
	logging.Debug(context.Background(), fmt.Sprintf(format, args...))
}

func RedirectStdLog() {
	initLogger()

	var w io.Writer = io.Discard
	if tuiLogFile != nil {
		w = tuiLogFile
	}

	log.SetOutput(w)

	tuiSlogMu.Lock()
	defer tuiSlogMu.Unlock()

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logging.Logger = slog.New(handler)
	slog.SetDefault(logging.Logger)
}

func LogFilePath() string {
	initLogger()
	if tuiLogFile != nil {
		return tuiLogFile.Name()
	}
	return ""
}
