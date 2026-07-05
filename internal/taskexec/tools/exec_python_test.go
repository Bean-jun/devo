package tools

import (
	"strings"
	"testing"
)

func isPythonAvailable() bool {
	tool := NewExecPythonTool()
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
