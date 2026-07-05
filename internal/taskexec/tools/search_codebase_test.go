package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCodebaseTool_FilesWithMatches(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "foo.go"), []byte("package main\nfunc foo() {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "bar.go"), []byte("package main\nfunc bar() {}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("no match here"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":     "func",
		"output_mode": "files_with_matches",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "foo.go") {
		t.Error("expected output to contain foo.go")
	}
	if !strings.Contains(result.Content, "bar.go") {
		t.Error("expected output to contain bar.go")
	}
	if strings.Contains(result.Content, "README.md") {
		t.Error("expected output to NOT contain README.md")
	}
	if strings.Contains(result.Content, "func foo") {
		t.Error("files_with_matches should not contain line content")
	}
}

func TestSearchCodebaseTool_Count(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("func foo()\nfunc bar()"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":     "func",
		"output_mode": "count",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "2 matches") {
		t.Errorf("expected count output '2 matches', got: %s", result.Content)
	}
}

func TestSearchCodebaseTool_CaseInsensitive(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("Func FOO()"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":        "func",
		"case_sensitive": false,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "main.go") {
		t.Error("expected output to contain main.go (case insensitive)")
	}
}

func TestSearchCodebaseTool_CaseSensitive_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("Func FOO()"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":        "func",
		"case_sensitive": true,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if strings.Contains(result.Content, "main.go") {
		t.Error("expected no match with case-sensitive search for 'func' vs 'Func'")
	}
}

func TestSearchCodebaseTool_FilePattern(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("func foo()"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "utils.js"), []byte("function foo()"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":      "foo",
		"file_pattern": "*.go",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "main.go") {
		t.Error("expected output to contain main.go")
	}
	if strings.Contains(result.Content, "utils.js") {
		t.Error("expected output to NOT contain utils.js (filtered by file_pattern)")
	}
}

func TestSearchCodebaseTool_ContextLines(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("line 1\nline 2\nfunc foo()\nline 4\nline 5"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":       "func",
		"context_lines": float64(1),
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "line 2") {
		t.Error("expected context to include line 2 (before match)")
	}
	if !strings.Contains(result.Content, "line 4") {
		t.Error("expected context to include line 4 (after match)")
	}
	if !strings.Contains(result.Content, "> func foo") {
		t.Error("expected match line to have '>' marker")
	}
	if strings.Contains(result.Content, "line 1") {
		t.Error("expected context NOT to include line 1 (out of range)")
	}
}

func TestSearchCodebaseTool_HeadLimit(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("func a()\nfunc b()\nfunc c()\nfunc d()\nfunc e()"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":    "func",
		"head_limit": float64(2),
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if strings.Count(result.Content, "\n") > 2 {
		t.Errorf("expected at most 2 lines of output, got: %s", result.Content)
	}
}

func TestSearchCodebaseTool_MaxResults(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("func a()\nfunc b()\nfunc c()\nfunc d()\nfunc e()"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern":     "func",
		"max_results": float64(2),
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if strings.Count(result.Content, "func") > 2 {
		t.Errorf("expected at most 2 results, got: %s", result.Content)
	}
}

func TestSearchCodebaseTool_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("func a()"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "nonexistent123",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "No matches") {
		t.Errorf("expected 'No matches found', got: %q", result.Content)
	}
}
