//go:build !windows

package process

import "syscall"

func killProcessGroup(pid int) {
	syscall.Kill(-pid, syscall.SIGKILL)
}