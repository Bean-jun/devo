//go:build !windows

package tools

import (
	"os/exec"
	"syscall"
)

func getPythonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func hideWindowForPython(cmd *exec.Cmd) {}
