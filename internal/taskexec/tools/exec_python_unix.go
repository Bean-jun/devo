//go:build !windows

package tools

import (
	"os/exec"
	"syscall"
)

func getPythonSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func hideWindowForPython(cmd *exec.Cmd) {}
