package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type ExecMode string

const (
	ExecModeSync  ExecMode = "sync"
	ExecModeAsync ExecMode = "async"
	ExecModeAuto  ExecMode = "auto"
)

type ExecResult struct {
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	TimedOut   bool   `json:"timed_out"`
	Background bool   `json:"background"`
	PID        int    `json:"pid"`
}

type OutputCallback func(line string, isStderr bool)

type NativeExecutor struct {
	mu sync.Mutex
}

func NewExecutor() *NativeExecutor {
	return &NativeExecutor{}
}

func (e *NativeExecutor) Execute(ctx context.Context, workingDir, command string, timeoutSeconds int, mode ExecMode) (*ExecResult, error) {
	return e.ExecuteStreaming(ctx, workingDir, command, timeoutSeconds, mode, nil)
}

func (e *NativeExecutor) ExecuteStreaming(ctx context.Context, workingDir, command string, timeoutSeconds int, mode ExecMode, onOutput OutputCallback) (*ExecResult, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}

	actualMode := e.resolveMode(mode, command)

	switch actualMode {
	case ExecModeAsync:
		return e.executeAsync(ctx, workingDir, command, timeoutSeconds)
	default:
		return e.executeSyncStreaming(ctx, workingDir, command, timeoutSeconds, onOutput)
	}
}

func (e *NativeExecutor) resolveMode(mode ExecMode, command string) ExecMode {
	if mode == ExecModeSync || mode == ExecModeAsync {
		return mode
	}
	if isBackgroundCommand(command) {
		return ExecModeAsync
	}
	return ExecModeSync
}

func (e *NativeExecutor) executeSyncStreaming(ctx context.Context, workingDir, command string, timeoutSeconds int, onOutput OutputCallback) (*ExecResult, error) {
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	shell, shellArgs := getShellCommand(command)

	cmd := exec.CommandContext(execCtx, shell, shellArgs...)
	cmd.Dir = workingDir

	e.setPlatformAttrs(cmd)

	var stdoutBuf, stderrBuf bytes.Buffer

	var runErr error

	if onOutput != nil {
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("create stdout pipe: %w", err)
		}
		stderrPipe, err := cmd.StderrPipe()
		if err != nil {
			return nil, fmt.Errorf("create stderr pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("start command: %w", err)
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			scanner := newLineScanner(stdoutPipe)
			for scanner.Scan() {
				line := scanner.Text()
				stdoutBuf.WriteString(line + "\n")
				onOutput(line, false)
			}
		}()

		go func() {
			defer wg.Done()
			scanner := newLineScanner(stderrPipe)
			for scanner.Scan() {
				line := scanner.Text()
				stderrBuf.WriteString(line + "\n")
				onOutput(line, true)
			}
		}()

		wg.Wait()
		runErr = cmd.Wait()
	} else {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		runErr = cmd.Run()
	}

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	timedOut := false
	exitCode := 0

	if runErr != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			timedOut = true
			exitCode = -1
		} else if ctx.Err() != nil {
			timedOut = false
			exitCode = -1
		} else if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("command execution failed: %w", runErr)
		}
	}

	return &ExecResult{
		ExitCode:   exitCode,
		Stdout:     decodeOutput(stdoutBuf.Bytes()),
		Stderr:     decodeOutput(stderrBuf.Bytes()),
		TimedOut:   timedOut,
		Background: false,
		PID:        pid,
	}, nil
}

func newLineScanner(r io.Reader) *bufioScanner {
	return bufio.NewScanner(r)
}

type bufioScanner = bufio.Scanner

func (e *NativeExecutor) executeAsync(ctx context.Context, workingDir, command string, timeoutSeconds int) (*ExecResult, error) {
	shell, shellArgs := getShellCommand(command)

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = workingDir

	e.setPlatformAttrs(cmd)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(&stdoutBuf, stdoutPipe)
	}()
	go func() {
		defer wg.Done()
		io.Copy(&stderrBuf, stderrPipe)
	}()

	collectionTimeout := time.Duration(min(timeoutSeconds, 5)) * time.Second
	startTime := time.Now()
	lastOutputLen := 0

	for {
		elapsed := time.Since(startTime)
		currentLen := stdoutBuf.Len() + stderrBuf.Len()

		if currentLen > lastOutputLen {
			lastOutputLen = currentLen
		}

		if currentLen > 0 && elapsed > 2*time.Second {
			stableDuration := time.Since(startTime)
			_ = stableDuration
			break
		}

		if elapsed >= collectionTimeout {
			break
		}

		if e.isProcessDone(cmd) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	_ = lastOutputLen

	exitCode := 0
	if e.isProcessDone(cmd) {
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}
	}

	return &ExecResult{
		ExitCode:   exitCode,
		Stdout:     decodeOutput(stdoutBuf.Bytes()),
		Stderr:     decodeOutput(stderrBuf.Bytes()),
		TimedOut:   false,
		Background: true,
		PID:        pid,
	}, nil
}

func (e *NativeExecutor) isProcessDone(cmd *exec.Cmd) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if cmd.ProcessState != nil {
		return true
	}
	return false
}

func decodeOutput(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	if utf8.Valid(data) {
		return string(data)
	}

	if runtime.GOOS == "windows" {
		reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
		decoded, err := io.ReadAll(reader)
		if err == nil {
			result := string(decoded)
			if utf8.ValidString(result) {
				return result
			}
		}
	}

	return strings.ToValidUTF8(string(data), "\uFFFD")
}

func isBackgroundCommand(cmd string) bool {
	trimmed := strings.TrimSpace(cmd)
	if runtime.GOOS == "windows" {
		lower := strings.ToLower(trimmed)
		return strings.HasPrefix(lower, "start ")
	}
	return strings.HasSuffix(trimmed, "&") || strings.Contains(trimmed, "nohup ")
}

func getShellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/c", command}
	}
	return "sh", []string{"-c", command}
}

func PlatformResourceLimitsNote() string {
	return ""
}
