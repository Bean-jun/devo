package cli

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"

	"devo/internal/config"
	"devo/internal/pkg/logging"
)

func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func openBrowser(url string) {
	logging.Info(context.Background(), "opening browser",
		"url", url,
	)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		logging.Warn(context.Background(), "failed to open browser",
			"error", err,
		)
	}
}

func VerifyDevoDir() {
	devoDir := config.DevoDir()
	if err := os.MkdirAll(devoDir, 0755); err != nil {
		logging.Error(context.Background(), "failed to create .devo directory",
			"error", err,
		)
		os.Exit(1)
	}
}
