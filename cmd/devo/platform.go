package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func defaultDBPath() string {
	devoDir := defaultDevoDir()
	return filepath.Join(devoDir, "devo.db")
}

func defaultDevoDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("[devo] Failed to get home directory: %v", err)
	}
	devoDir := filepath.Join(homeDir, ".devo")
	if err := os.MkdirAll(devoDir, 0755); err != nil {
		log.Fatalf("[devo] Failed to create .devo directory: %v", err)
	}
	return devoDir
}

func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func openBrowser(url string) {
	log.Printf("[devo] Opening browser: %s", url)
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
		log.Printf("[devo] Failed to open browser: %v", err)
	}
}
