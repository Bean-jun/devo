//go:build windows

package tools

import (
	"fmt"
	"os/exec"
	"syscall"
)

// killProcess kills the process tree rooted at pid via taskkill /F /T. On
// Windows there is no Unix-style process group; CREATE_NEW_PROCESS_GROUP only
// affects console Ctrl+C routing, not kill semantics. We rely on the process
// tree (parent-child) instead, which works because exec_python starts python
// as a direct devo child and any long-running task is a direct python child
// (no start_new_session=True detachment anymore).
func killProcess(pid int) error {
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Exit code 128 means "process not found" - treat as already-exited.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return nil
		}
		return fmt.Errorf("taskkill /F /T /PID %d failed: %w (output: %s)", pid, err, string(out))
	}
	return nil
}
