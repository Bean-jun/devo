//go:build windows

package tools

import (
	"fmt"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

type windowsProcHandle struct {
	pid          int
	creationTime windows.Filetime
}

func newProcHandle(pid int) (procHandle, error) {
	ct, err := getCreationTime(pid)
	if err != nil {
		return nil, err
	}
	return &windowsProcHandle{pid: pid, creationTime: ct}, nil
}

func getCreationTime(pid int) (windows.Filetime, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return windows.Filetime{}, fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	defer windows.CloseHandle(h)

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(h, &creation, &exit, &kernel, &user); err != nil {
		return windows.Filetime{}, fmt.Errorf("GetProcessTimes(%d): %w", pid, err)
	}
	return creation, nil
}

func (h *windowsProcHandle) isAlive() (bool, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(h.pid))
	if err != nil {
		return false, nil
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false, err
	}
	if exitCode != uint32(259) /* STILL_ACTIVE */ {
		return false, nil
	}

	currentCreation, err := getCreationTime(h.pid)
	if err != nil {
		return false, nil
	}
	if currentCreation != h.creationTime {
		return false, nil
	}
	return true, nil
}

func (h *windowsProcHandle) terminateGraceful() error {
	err := windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(h.pid))
	if err != nil {
		return fmt.Errorf("GenerateConsoleCtrlEvent(%d): %w", h.pid, err)
	}
	return nil
}

func (h *windowsProcHandle) killForce() error {
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", h.pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 128 {
			return nil
		}
		return fmt.Errorf("taskkill /F /T /PID %d failed: %w (output: %s)", h.pid, err, string(out))
	}
	return nil
}

func recommendedCreationFlagsHint() string {
	return "for graceful shutdown support, start background processes with " +
		"creationflags=subprocess.CREATE_NEW_PROCESS_GROUP | subprocess.DETACHED_PROCESS"
}
