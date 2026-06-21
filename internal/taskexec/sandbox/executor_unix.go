//go:build !windows

package sandbox

import (
	"os/exec"
	"syscall"
)

func (e *NativeExecutor) setPlatformAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}