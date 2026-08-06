package process

func KillProcessGroup(pid int) {
	if pid <= 0 {
		return
	}
	killProcessGroup(pid)
}
