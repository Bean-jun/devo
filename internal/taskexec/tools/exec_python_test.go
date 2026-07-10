package tools

import (
	"devo/internal/pkg/process"
	"strings"
	"testing"
)

func isPythonAvailable() bool {
	tool := NewExecPythonTool(nil)
	return tool.pythonBin != ""
}

func TestExecPythonTool_SimpleExpression(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := &ExecPythonTool{}
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

	tool := &ExecPythonTool{}
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

	tool := &ExecPythonTool{}
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

	tool := &ExecPythonTool{}
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

	tool := &ExecPythonTool{}
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

	tool := &ExecPythonTool{}
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
	if ctx["timeout_seconds"] != 10 {
		t.Errorf("expected default timeout 10 for background, got %v", ctx["timeout_seconds"])
	}
}

func TestExecPythonTool_SubprocessEcho(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := &ExecPythonTool{}
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

	tool := &ExecPythonTool{}
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

	tool := &ExecPythonTool{}
	code := `import subprocess, time, sys
p = subprocess.Popen(["python", "-c", "import time; time.sleep(60)"], start_new_session=True)
time.sleep(0.5)
if p.poll() is not None:
    print("startup failed", file=sys.stderr)
    sys.exit(1)
print(f"__DEVO_BG_PID__={p.pid}")
print("background process started")
`
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": code,
		"mode": "background",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Errorf("expected success, got: %s", result.Content)
	}
	if !strings.Contains(result.Content, "__DEVO_BG_PID__") {
		t.Error("expected __DEVO_BG_PID__ tag in output")
	}

	bgPID := parseBGPID(result.Content)
	if bgPID > 0 {
		process.KillProcessGroup(bgPID)
	}
}

func TestExecPythonTool_Background_MissingPIDMarker(t *testing.T) {
	if !isPythonAvailable() {
		t.Skip("python not available, skipping test")
	}

	tool := &ExecPythonTool{}
	result, err := executeTool(t, tool, "", map[string]interface{}{
		"code": "print('hello')",
		"mode": "background",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Error("expected failure due to missing PID marker")
	}
	if !strings.Contains(result.Content, "__DEVO_BG_PID__") {
		t.Error("expected mention of __DEVO_BG_PID__ in error message")
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

func TestExecPythonTool_CommandContextProvider_Interface(t *testing.T) {
	var _ CommandContextProvider = (*ExecPythonTool)(nil)
}

func TestExecPythonTool_PreChecker_Interface(t *testing.T) {
	var _ PreChecker = (*ExecPythonTool)(nil)
}
