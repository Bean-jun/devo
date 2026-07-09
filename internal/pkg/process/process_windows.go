//go:build windows

package process

import (
	"fmt"
	"os/exec"
)

func killProcessGroup(pid int) {
	exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid)).Run()
}