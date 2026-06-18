package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"syscall"
)

const pythonScript = `
import os, sys, json, subprocess, signal, platform

command = os.environ.get("DEVO_COMMAND", "")
workdir = os.environ.get("DEVO_WORKDIR", "")
timeout_str = os.environ.get("DEVO_TIMEOUT", "30")

if not command:
    print(json.dumps({"exit_code": -1, "stdout": "", "stderr": "DEVO_COMMAND environment variable not set", "timed_out": False}))
    sys.exit(0)

try:
    timeout = int(timeout_str)
except ValueError:
    timeout = 30

try:
    if workdir:
        os.chdir(workdir)
except Exception as e:
    print(json.dumps({"exit_code": -1, "stdout": "", "stderr": "Failed to chdir to " + workdir + ": " + str(e), "timed_out": False}))
    sys.exit(0)

def set_resource_limits():
    if platform.system() == "Windows":
        return
    try:
        import resource
        resource.setrlimit(resource.RLIMIT_AS, (512 * 1024 * 1024, 1024 * 1024 * 1024))
        resource.setrlimit(resource.RLIMIT_CPU, (timeout + 5, timeout + 10))
        resource.setrlimit(resource.RLIMIT_NPROC, (50, 100))
    except Exception:
        pass

set_resource_limits()

try:
    proc = subprocess.Popen(
        command,
        shell=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        preexec_fn=None if platform.system() == "Windows" else os.setsid,
    )
    
    try:
        stdout, stderr = proc.communicate(timeout=timeout)
        exit_code = proc.returncode
        timed_out = False
    except subprocess.TimeoutExpired:
        if platform.system() != "Windows":
            try:
                os.killpg(os.getpgid(proc.pid), signal.SIGKILL)
            except Exception:
                proc.kill()
        else:
            proc.kill()
        stdout, stderr = proc.communicate()
        exit_code = -1
        timed_out = True
    
    result = {
        "exit_code": exit_code,
        "stdout": stdout.decode("utf-8", errors="replace") if stdout else "",
        "stderr": stderr.decode("utf-8", errors="replace") if stderr else "",
        "timed_out": timed_out,
    }
    print(json.dumps(result))
    
except Exception as e:
    print(json.dumps({"exit_code": -1, "stdout": "", "stderr": str(e), "timed_out": False}))

sys.exit(0)
`

type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timed_out"`
}

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(workingDir, command string, timeoutSeconds int) (*ExecResult, int, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	cmd := exec.Command("python", "-c", pythonScript)
	cmd.Env = append(os.Environ(),
		"DEVO_COMMAND="+command,
		"DEVO_WORKDIR="+workingDir,
		fmt.Sprintf("DEVO_TIMEOUT=%d", timeoutSeconds),
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{}

	stdout, err := cmd.Output()

	pid := cmd.Process.Pid

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderrOutput := string(exitErr.Stderr)
			if len(stdout) > 0 {
				var result ExecResult
				if jsonErr := json.Unmarshal(stdout, &result); jsonErr == nil {
					return &result, pid, nil
				}
			}
			return &ExecResult{
				ExitCode: -1,
				Stdout:   "",
				Stderr:   stderrOutput,
				TimedOut: false,
			}, pid, nil
		}
		return &ExecResult{
			ExitCode: -1,
			Stdout:   "",
			Stderr:   err.Error(),
			TimedOut: false,
		}, 0, err
	}

	var result ExecResult
	if err := json.Unmarshal(stdout, &result); err != nil {
		return &ExecResult{
			ExitCode: -1,
			Stdout:   string(stdout),
			Stderr:   "",
			TimedOut: false,
		}, pid, nil
	}

	return &result, pid, nil
}

func IsPythonAvailable() bool {
	_, err := exec.LookPath("python")
	if err != nil {
		_, err = exec.LookPath("python3")
	}
	return err == nil
}

func PlatformResourceLimitsNote() string {
	if runtime.GOOS == "windows" {
		return "resource limits (setrlimit) not supported on Windows"
	}
	return "resource limits active on " + runtime.GOOS
}
