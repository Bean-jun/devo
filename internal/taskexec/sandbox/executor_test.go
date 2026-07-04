package sandbox

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNativeExecutor_ExecuteSync(t *testing.T) {
	executor := NewExecutor()

	result, err := executor.Execute(context.Background(), "", "echo hello", 30, ExecModeSync)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got: %s", result.Stdout)
	}

	if result.Background {
		t.Error("expected Background=false for sync execution")
	}

	if result.PID <= 0 {
		t.Error("expected valid PID > 0")
	}

	if result.TimedOut {
		t.Error("expected TimedOut=false")
	}
}

func TestNativeExecutor_ExecuteSync_Timeout(t *testing.T) {
	executor := NewExecutor()

	cmd := "sleep 10"
	if runtime.GOOS == "windows" {
		cmd = "ping -n 10 127.0.0.1"
	}

	result, err := executor.Execute(context.Background(), "", cmd, 1, ExecModeSync)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.TimedOut {
		t.Error("expected TimedOut=true for timeout")
	}
}

func TestNativeExecutor_ExecuteSync_NonZeroExit(t *testing.T) {
	executor := NewExecutor()

	cmd := "exit 1"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c exit 1"
	}

	result, err := executor.Execute(context.Background(), "", cmd, 30, ExecModeSync)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
}

func TestNativeExecutor_ExecuteAsync_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific test")
	}

	executor := NewExecutor()

	result, err := executor.Execute(context.Background(), "", "start /B timeout 5", 30, ExecModeAsync)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.Background {
		t.Error("expected Background=true for async execution")
	}
}

func TestNativeExecutor_ExecuteAsync_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	executor := NewExecutor()

	result, err := executor.Execute(context.Background(), "", "sleep 30 &", 5, ExecModeAsync)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !result.Background {
		t.Error("expected Background=true for async execution")
	}
}

func TestNativeExecutor_ResolveMode_AutoSync(t *testing.T) {
	executor := NewExecutor()

	mode := executor.resolveMode(ExecModeAuto, "echo hello")
	if mode != ExecModeSync {
		t.Errorf("expected ExecModeSync for normal command, got %s", mode)
	}
}

func TestNativeExecutor_ResolveMode_AutoAsync_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific test")
	}

	executor := NewExecutor()

	mode := executor.resolveMode(ExecModeAuto, "start /B server.exe")
	if mode != ExecModeAsync {
		t.Errorf("expected ExecModeAsync for 'start' command, got %s", mode)
	}
}

func TestNativeExecutor_ResolveMode_AutoAsync_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	executor := NewExecutor()

	mode := executor.resolveMode(ExecModeAuto, "sleep 30 &")
	if mode != ExecModeAsync {
		t.Errorf("expected ExecModeAsync for background command, got %s", mode)
	}

	mode = executor.resolveMode(ExecModeAuto, "nohup sleep 30")
	if mode != ExecModeAsync {
		t.Errorf("expected ExecModeAsync for nohup command, got %s", mode)
	}
}

func TestNativeExecutor_ResolveMode_ExplicitSync(t *testing.T) {
	executor := NewExecutor()

	mode := executor.resolveMode(ExecModeSync, "start /B server.exe")
	if mode != ExecModeSync {
		t.Errorf("expected ExecModeSync when explicitly requested, got %s", mode)
	}
}

func TestNativeExecutor_ResolveMode_ExplicitAsync(t *testing.T) {
	executor := NewExecutor()

	mode := executor.resolveMode(ExecModeAsync, "echo hello")
	if mode != ExecModeAsync {
		t.Errorf("expected ExecModeAsync when explicitly requested, got %s", mode)
	}
}

func TestDecodeOutput_UTF8(t *testing.T) {
	input := []byte("Hello, 世界")
	result := decodeOutput(input)

	if result != "Hello, 世界" {
		t.Errorf("expected 'Hello, 世界', got '%s'", result)
	}
}

func TestDecodeOutput_Empty(t *testing.T) {
	result := decodeOutput([]byte{})
	if result != "" {
		t.Errorf("expected empty string, got '%s'", result)
	}
}

func TestDecodeOutput_InvalidUTF8(t *testing.T) {
	invalid := []byte{0xFF, 0xFE, 0xFD}
	result := decodeOutput(invalid)

	if !utf8.ValidString(result) {
		t.Errorf("expected valid UTF-8 output, got invalid string")
	}
}

func TestDecodeOutput_GBK(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("GBK test only relevant on Windows")
	}

	gbkBytes := []byte{0xD6, 0xD0, 0xCE, 0xC4}
	result := decodeOutput(gbkBytes)

	if !utf8.ValidString(result) {
		t.Errorf("expected valid UTF-8 output from GBK, got: %q", result)
	}

	if len(result) == 0 {
		t.Error("expected non-empty decoded output")
	}
}

func TestIsBackgroundCommand_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific test")
	}

	tests := []struct {
		cmd      string
		expected bool
	}{
		{"start /B server.exe", true},
		{"START /B server.exe", true},
		{"start server.exe", true},
		{"echo hello", false},
		{"dir", false},
		{"python server.py", false},
	}

	for _, tt := range tests {
		result := isBackgroundCommand(tt.cmd)
		if result != tt.expected {
			t.Errorf("isBackgroundCommand(%q) = %v, expected %v", tt.cmd, result, tt.expected)
		}
	}
}

func TestIsBackgroundCommand_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-specific test")
	}

	tests := []struct {
		cmd      string
		expected bool
	}{
		{"sleep 30 &", true},
		{"python server.py &", true},
		{"nohup sleep 30", true},
		{"nohup python server.py &", true},
		{"echo hello", false},
		{"ls -la", false},
	}

	for _, tt := range tests {
		result := isBackgroundCommand(tt.cmd)
		if result != tt.expected {
			t.Errorf("isBackgroundCommand(%q) = %v, expected %v", tt.cmd, result, tt.expected)
		}
	}
}

func TestGetShellCommand(t *testing.T) {
	shell, args := getShellCommand("echo hello")

	if runtime.GOOS == "windows" {
		if shell != "cmd" {
			t.Errorf("expected 'cmd' on Windows, got '%s'", shell)
		}
		if len(args) != 2 || args[0] != "/c" || args[1] != "echo hello" {
			t.Errorf("expected ['/c', 'echo hello'], got %v", args)
		}
	} else {
		if shell != "sh" {
			t.Errorf("expected 'sh' on Unix, got '%s'", shell)
		}
		if len(args) != 2 || args[0] != "-c" || args[1] != "echo hello" {
			t.Errorf("expected ['-c', 'echo hello'], got %v", args)
		}
	}
}

func TestPlatformResourceLimitsNote(t *testing.T) {
	result := PlatformResourceLimitsNote()
	if result != "" {
		t.Errorf("expected empty string (no Python sandbox), got '%s'", result)
	}
}

func TestNewExecutor(t *testing.T) {
	executor := NewExecutor()
	if executor == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestNativeExecutor_StderrCapture(t *testing.T) {
	executor := NewExecutor()

	cmd := "ls nonexistent 2>&1"
	if runtime.GOOS == "windows" {
		cmd = "dir nonexistent 2>&1"
	}

	result, err := executor.Execute(context.Background(), "", cmd, 30, ExecModeSync)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	hasStderr := result.Stderr != ""
	hasStdout := result.Stdout != ""
	if !hasStderr && !hasStdout {
		t.Error("expected either stdout or stderr to contain error information")
	}
}

func TestNativeExecutor_WorkingDirectory(t *testing.T) {
	executor := NewExecutor()

	tmpDir := t.TempDir()

	cmd := "echo %cd%"
	if runtime.GOOS != "windows" {
		cmd = "pwd"
	}

	result, err := executor.Execute(context.Background(), tmpDir, cmd, 30, ExecModeSync)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result.Stdout, tmpDir) {
		t.Errorf("expected working directory '%s' in output, got: %s", tmpDir, result.Stdout)
	}
}
