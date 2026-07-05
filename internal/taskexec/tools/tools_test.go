package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executeTool(t *testing.T, tool Tool, workingDir string, params map[string]interface{}) (*ToolResult, error) {
	t.Helper()
	ctx := context.Background()
	ch := make(chan StreamEvent, 256)
	sw := NewChannelStreamWriter(ch)

	go func() {
		defer close(ch)
		if err := tool.Execute(ctx, workingDir, params, sw); err != nil {
			sw.WriteError(err)
		}
	}()

	return CollectToolResult(ch, nil), nil
}

func TestReadFileTool_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, Devo!"
	os.WriteFile(testFile, []byte(content), 0644)

	tool := &ReadFileTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"path": "test.txt",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Content != content {
		t.Errorf("expected %q, got %q", content, result.Content)
	}
}

func TestReadFileTool_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &ReadFileTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"path": "nonexistent.txt",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestReadFileTool_PathOutsideWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &ReadFileTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"path": "../../etc/passwd",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for path traversal")
	}
}

func TestReadFileTool_MissingPathParam(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &ReadFileTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for missing path parameter")
	}
}

func TestReadFileTool_PathIsDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &ReadFileTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"path": ".",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error when path is a directory")
	}
}

func TestListFilesTool_RootDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.txt"), []byte("b"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "c.txt"), []byte("c"), 0644)

	tool := &ListFilesTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "a.txt") {
		t.Error("expected output to contain a.txt")
	}
	if !strings.Contains(result.Content, "b.txt") {
		t.Error("expected output to contain b.txt")
	}
	if !strings.Contains(result.Content, "subdir/") {
		t.Error("expected output to contain subdir/")
	}
}

func TestListFilesTool_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "c.txt"), []byte("c"), 0644)

	tool := &ListFilesTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"max_depth": float64(0),
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if strings.Contains(result.Content, "c.txt") {
		t.Error("with max_depth=0, should not contain nested file c.txt")
	}
	if !strings.Contains(result.Content, "subdir/") {
		t.Error("expected output to contain subdir/")
	}
}

func TestListFilesTool_PathOutsideWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &ListFilesTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"path": "../../etc",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for path traversal")
	}
}

func TestSearchCodebaseTool_FindMatch(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "app.go"), []byte("package main\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "Hello",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "app.go") {
		t.Error("expected output to contain app.go")
	}
	if !strings.Contains(result.Content, "Hello") {
		t.Error("expected output to contain 'Hello'")
	}
}

func TestSearchCodebaseTool_PathOutsideWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "test",
		"path":    "../../etc",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for path traversal")
	}
}

func TestSearchCodebaseTool_SpecificPath(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "nested.txt"), []byte("world"), 0644)

	tool := &SearchCodebaseTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "world",
		"path":    "subdir",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if strings.Contains(result.Content, "root.txt") {
		t.Error("searching subdir should not find root.txt")
	}
	if !strings.Contains(result.Content, "nested.txt") {
		t.Error("expected output to contain nested.txt")
	}
}

func TestRegistry_ExecuteKnownTool(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&ReadFileTool{})

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("content"), 0644)

	eventCh, err := reg.Execute(context.Background(), tmpDir, "read_file", map[string]interface{}{
		"path": "test.txt",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	result := CollectToolResult(eventCh, nil)
	if !result.Success {
		t.Errorf("expected successful result, got error: %s", result.Error)
	}
	if result.Content != "content" {
		t.Errorf("expected 'content', got %q", result.Content)
	}
}

func TestRegistry_ExecuteUnknownTool(t *testing.T) {
	reg := NewRegistry()

	eventCh, err := reg.Execute(context.Background(), "/tmp", "unknown_tool", nil)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	result := CollectToolResult(eventCh, nil)
	if result.Success {
		t.Error("expected failure for unknown tool")
	}
}
