//go:build !windows

package tools

import "syscall"

// killProcess kills the process group rooted at pid. exec_python on Unix
// starts python with Setpgid:true, so pid is both the python pid and its
// process group id. Killing -pid takes down python and any children that
// haven't detached (which is what we want - background processes are devo
// children, not detached grandchildren).
func killProcess(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		// ESRCH means the process already exited - treat as success.
		if err == syscall.ESRCH {
			return nil
		}
		return syscall.Kill(pid, syscall.SIGKILL)
	}
	return nil
}
