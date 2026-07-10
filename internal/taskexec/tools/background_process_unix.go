//go:build !windows

package tools

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type unixProcHandle struct {
	pid       int
	startTime uint64
}

func newProcHandle(pid int) (procHandle, error) {
	st, err := getStartTime(pid)
	if err != nil {
		return nil, err
	}
	return &unixProcHandle{pid: pid, startTime: st}, nil
}

func getStartTime(pid int) (uint64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, fmt.Errorf("read /proc/%d/stat: %w", pid, err)
	}
	content := string(data)
	closeParen := strings.LastIndex(content, ")")
	if closeParen < 0 {
		return 0, fmt.Errorf("unexpected /proc/%d/stat format", pid)
	}
	fields := strings.Fields(content[closeParen+2:])
	if len(fields) < 20 {
		return 0, fmt.Errorf("unexpected /proc/%d/stat field count", pid)
	}
	startTime, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse starttime for pid %d: %w", pid, err)
	}
	return startTime, nil
}

func (h *unixProcHandle) isAlive() (bool, error) {
	currentStart, err := getStartTime(h.pid)
	if err != nil {
		return false, nil
	}
	if currentStart != h.startTime {
		return false, nil
	}
	return true, nil
}

func (h *unixProcHandle) terminateGraceful() error {
	return syscall.Kill(-h.pid, syscall.SIGTERM)
}

func (h *unixProcHandle) killForce() error {
	// 先尝试 SIGKILL 到进程组，失败后只杀单个进程
	if err := syscall.Kill(-h.pid, syscall.SIGKILL); err != nil {
		return syscall.Kill(h.pid, syscall.SIGKILL)
	}
	return nil
}

func recommendedCreationFlagsHint() string {
	return ""
}

func init() {
	_ = time.Now
}
