package tools

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func isPythonAvailable() bool {
	tool := NewExecPythonTool(nil)
	return tool.pythonBin != ""
}

func TestExecPythonTool_SimpleExpression(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "print(1+1)",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result.Content, "Exit code: 0") {
		t.Errorf("expected exit code 0, got: %s", result.Content)
	}

	if !strings.Contains(result.Content, "2") {
		t.Errorf("expected output to contain '2', got: %s", result.Content)
	}
}

func TestExecPythonTool_JSONProcessing(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "import json; data={'a':1,'b':2}; print(json.dumps(data))",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result.Content, `{"a": 1, "b": 2}`) {
		t.Errorf("expected JSON output, got: %s", result.Content)
	}
}

func TestExecPythonTool_NonZeroExitCode(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "import sys; sys.exit(1)",
	})

	if err != nil {
		t.Fatalf("expected no error (non-zero exit is valid), got: %v", err)
	}

	if !strings.Contains(result.Content, "Exit code: 1") {
		t.Errorf("expected exit code 1, got: %s", result.Content)
	}
}

func TestExecPythonTool_SyntaxError(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "print(",
	})

	if err != nil {
		t.Fatalf("expected no error (syntax error is valid result), got: %v", err)
	}

	if !strings.Contains(result.Content, "Exit code: 1") {
		t.Errorf("expected non-zero exit code for syntax error, got: %s", result.Content)
	}
}

func TestExecPythonTool_RuntimeError(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "1/0",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result.Content, "Exit code: 1") {
		t.Errorf("expected non-zero exit code for runtime error, got: %s", result.Content)
	}
}

func TestExecPythonTool_Timeout(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code":            "import time; time.sleep(10)",
		"timeout_seconds": float64(1),
	})

	if err != nil {
		t.Fatalf("expected no error (timeout is valid result), got: %v", err)
	}

	if !strings.Contains(result.Content, "timed out") {
		t.Errorf("expected timeout message, got: %s", result.Content)
	}
}

func TestExecPythonTool_MissingCode(t *testing.T) {
	tool := &ExecPythonTool{}
	result, err := executeTool(t, tool, "", map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for missing code parameter")
	}

	if !strings.Contains(result.Error, "missing required parameter: code") {
		t.Errorf("expected error about missing code, got: %v", result.Error)
	}
}

func TestExecPythonTool_EmptyCode(t *testing.T) {
	tool := &ExecPythonTool{}
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for empty code")
	}
}

func TestExecPythonTool_RiskLevel(t *testing.T) {
	tool := &ExecPythonTool{}

	rl := tool.RiskLevel()
	if rl != RiskLevelHigh {
		t.Errorf("expected RiskLevelHigh, got %s", rl)
	}
}

func TestExecPythonTool_OperationType(t *testing.T) {
	tool := &ExecPythonTool{}

	opType := tool.OperationType("", nil)
	if opType != "exec_python" {
		t.Errorf("expected 'exec_python', got %s", opType)
	}
}

func TestExecPythonTool_Name(t *testing.T) {
	tool := &ExecPythonTool{}

	if tool.Name() != "exec_python" {
		t.Errorf("expected 'exec_python', got %s", tool.Name())
	}
}

func TestExecPythonTool_ParamsSchema(t *testing.T) {
	tool := &ExecPythonTool{}

	schema := tool.ParamsSchema()

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("expected required field in schema")
	}

	found := false
	for _, r := range required {
		if r == "code" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'code' in required parameters")
	}
}

func TestExecPythonTool_GetCommandContext_Sync(t *testing.T) {
	tool := NewExecPythonTool(nil)

	ctx := tool.GetCommandContext("/tmp/test", map[string]interface{}{
		"code":            "print('hello')",
		"timeout_seconds": float64(60),
	})

	if ctx["working_directory"] != "/tmp/test" {
		t.Errorf("expected working_directory /tmp/test, got %v", ctx["working_directory"])
	}
	if ctx["mode"] != "sync" {
		t.Errorf("expected mode sync, got %v", ctx["mode"])
	}
	if ctx["timeout_seconds"] != 60 {
		t.Errorf("expected timeout_seconds 60, got %v", ctx["timeout_seconds"])
	}
}

func TestExecPythonTool_GetCommandContext_Background(t *testing.T) {
	tool := NewExecPythonTool(nil)

	ctx := tool.GetCommandContext("/tmp/test", map[string]interface{}{
		"code": "print('hello')",
		"mode": "background",
	})

	if ctx["mode"] != "background" {
		t.Errorf("expected mode background, got %v", ctx["mode"])
	}
	if ctx["timeout_seconds"] != 30 {
		t.Errorf("expected default timeout 30 for background (unused but reported), got %v", ctx["timeout_seconds"])
	}
}

func TestExecPythonTool_SubprocessEcho(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	code := "import subprocess, sys\nr = subprocess.run(['echo', 'hello from subprocess'], capture_output=True, text=True, shell=True)\nprint(r.stdout.strip())\nsys.exit(r.returncode)"
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": code,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Content)
	}
}

func TestExecPythonTool_StderrCaptured(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "import sys; print('to stderr', file=sys.stderr); print('to stdout')",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Error("expected success")
	}
	if !strings.Contains(result.Content, "to stderr") {
		t.Errorf("expected stderr content in output, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "to stdout") {
		t.Errorf("expected stdout content in output, got: %s", result.Content)
	}
}

func TestExecPythonTool_Background_Success(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	mgr := NewBackgroundProcessManager()
	tool := NewExecPythonTool(mgr)
	ctx := WithSessionID(context.Background(), "test-session")

	result, err := executeToolWithCtx(t, ctx, tool, "", map[string]interface{}{
		"code": "import time; time.sleep(60)",
		"mode": "background",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "PID=") {
		t.Errorf("expected 'PID=' in output, got: %s", result.Content)
	}

	re := regexp.MustCompile(`PID=(\d+)`)
	m := re.FindStringSubmatch(result.Content)
	if len(m) < 2 {
		t.Fatalf("could not extract PID from output: %s", result.Content)
	}
	pid, err := strconv.Atoi(m[1])
	if err != nil || pid <= 0 {
		t.Fatalf("invalid PID extracted: %s", m[1])
	}

	if err := mgr.Stop(pid); err != nil {
		t.Errorf("Stop(%d) failed: %v", pid, err)
	}
}

func TestExecPythonTool_Background_NoManager(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := NewExecPythonTool(nil)
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "import time; time.sleep(60)",
		"mode": "background",
	})

	if err != nil {
		t.Fatalf("expected no error from Execute wrapper, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected failure when bgManager is not configured")
	}
	if !strings.Contains(result.Error, "background process manager not configured") {
		t.Errorf("expected 'not configured' error, got: %v", result.Error)
	}
}

type fakeForwarder struct {
	mu     sync.Mutex
	chunks []struct {
		sessionID string
		pid       int
		stream    string
		data      string
	}
}

func (f *fakeForwarder) ForwardBackgroundOutput(sessionID string, pid int, stream string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.chunks = append(f.chunks, struct {
		sessionID string
		pid       int
		stream    string
		data      string
	}{sessionID, pid, stream, string(data)})
}

func (f *fakeForwarder) snapshot() []struct {
	sessionID string
	pid       int
	stream    string
	data      string
} {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]struct {
		sessionID string
		pid       int
		stream    string
		data      string
	}, len(f.chunks))
	copy(out, f.chunks)
	return out
}

func TestExecPythonTool_Background_OutputForwarded(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	mgr := NewBackgroundProcessManager()
	fwd := &fakeForwarder{}
	mgr.SetOutputForwarder(fwd)
	tool := NewExecPythonTool(mgr)
	ctx := WithSessionID(context.Background(), "test-session-fwd")

	result, err := executeToolWithCtx(t, ctx, tool, "", map[string]interface{}{
		"code": "import time, sys\nprint('line-one', flush=True)\nprint('line-two', file=sys.stderr, flush=True)\ntime.sleep(60)",
		"mode": "background",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got: %s", result.Content)
	}

	re := regexp.MustCompile(`PID=(\d+)`)
	m := re.FindStringSubmatch(result.Content)
	if len(m) < 2 {
		t.Fatalf("could not extract PID: %s", result.Content)
	}
	pid, _ := strconv.Atoi(m[1])

	// Give the pipe reader a moment to flush. The throttle is 100ms.
	deadline := time.Now().Add(2 * time.Second)
	var chunks []struct {
		sessionID string
		pid       int
		stream    string
		data      string
	}
	for time.Now().Before(deadline) {
		chunks = fwd.snapshot()
		if len(chunks) >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if len(chunks) == 0 {
		t.Fatal("expected at least one forwarded chunk, got none")
	}

	var sawStdout, sawStderr bool
	for _, c := range chunks {
		if c.sessionID != "test-session-fwd" {
			t.Errorf("unexpected sessionID: %q", c.sessionID)
		}
		if c.pid != pid {
			t.Errorf("unexpected pid: got %d, want %d", c.pid, pid)
		}
		if strings.Contains(c.data, "line-one") {
			sawStdout = true
		}
		if strings.Contains(c.data, "line-two") {
			sawStderr = true
		}
	}
	if !sawStdout {
		t.Error("expected stdout 'line-one' to be forwarded")
	}
	if !sawStderr {
		t.Error("expected stderr 'line-two' to be forwarded")
	}

	if err := mgr.Stop(pid); err != nil {
		t.Errorf("Stop(%d) failed: %v", pid, err)
	}
}

func TestExecPythonTool_PreCheck_Blacklisted(t *testing.T) {
	tool := &ExecPythonTool{}
	err := tool.PreCheck(map[string]interface{}{"code": `subprocess.run(["rm", "-rf", "/"])`})
	if err == nil {
		t.Error("expected error for blacklisted code")
	}
}

func TestExecPythonTool_PreCheck_SafeCode(t *testing.T) {
	tool := &ExecPythonTool{}
	err := tool.PreCheck(map[string]interface{}{"code": "print('hello')"})
	if err != nil {
		t.Errorf("expected no error for safe code, got: %v", err)
	}
}

// TestExecPythonTool_PreCheck_BackgroundPopenRejected covers the regression
// where the agent used `subprocess.Popen + readline + exit` in background
// mode, leaving the actual server as an orphaned grandchild the manager
// could not stop. PreCheck now rejects any Popen usage in background mode.
func TestExecPythonTool_PreCheck_BackgroundPopenRejected(t *testing.T) {
	cases := []struct {
		name string
		code string
	}{
		{
			name: "classic spawn-and-exit",
			code: `import subprocess
p = subprocess.Popen(["npm", "run", "dev"], start_new_session=True)
print(p.pid)`,
		},
		{
			name: "readline loop variant",
			code: `import subprocess, time
p = subprocess.Popen(["npm", "run", "dev"], stdout=subprocess.PIPE)
while time.time() < t:
    line = p.stdout.readline()
    if "ready" in line: break`,
		},
		{
			name: "bare Popen from import",
			code: `from subprocess import Popen
p = Popen(["npm", "run", "dev"])`,
		},
		{
			name: "sleep + poll variant",
			code: `import subprocess, time
p = subprocess.Popen(["npm", "run", "dev"])
time.sleep(5)
print(p.poll())`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tool := &ExecPythonTool{}
			err := tool.PreCheck(map[string]interface{}{
				"code": tc.code,
				"mode": "background",
			})
			if err == nil {
				t.Errorf("expected error for background mode Popen usage, got nil")
			}
		})
	}
}

// TestExecPythonTool_PreCheck_BackgroundSubprocessRunAllowed ensures the
// correct primitive (subprocess.run blocking) is not rejected in background
// mode - only Popen is forbidden.
func TestExecPythonTool_PreCheck_BackgroundSubprocessRunAllowed(t *testing.T) {
	tool := &ExecPythonTool{}
	err := tool.PreCheck(map[string]interface{}{
		"code": `import subprocess
subprocess.run(["npm", "run", "dev"])`,
		"mode": "background",
	})
	if err != nil {
		t.Errorf("expected no error for subprocess.run in background mode, got: %v", err)
	}
}

// TestExecPythonTool_PreCheck_SyncPopenAllowed verifies Popen is still
// permitted in sync mode (the check is background-only). Sync mode with
// Popen has its own pitfalls (pipe inheritance causing hangs) but those are
// surfaced at execution time, not PreCheck.
func TestExecPythonTool_PreCheck_SyncPopenAllowed(t *testing.T) {
	tool := &ExecPythonTool{}
	err := tool.PreCheck(map[string]interface{}{
		"code": `import subprocess
p = subprocess.Popen(["echo", "hi"], stdout=subprocess.PIPE)
print(p.communicate())`,
	})
	if err != nil {
		t.Errorf("expected no error for Popen in sync mode, got: %v", err)
	}
}

func TestExecPythonTool_CommandContextProvider_Interface(t *testing.T) {
	var _ CommandContextProvider = (*ExecPythonTool)(nil)
}

func TestExecPythonTool_PreChecker_Interface(t *testing.T) {
	var _ PreChecker = (*ExecPythonTool)(nil)
}
