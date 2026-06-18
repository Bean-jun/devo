package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileTool_CreateNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &WriteFileTool{}

	result, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "newfile.txt",
		"content": "Hello, World!",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result, "created") {
		t.Errorf("expected result to contain 'created', got: %s", result)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "newfile.txt"))
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	if string(data) != "Hello, World!" {
		t.Errorf("expected file content 'Hello, World!', got %q", string(data))
	}
}

func TestWriteFileTool_OverwriteExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(testFile, []byte("old content"), 0644)

	tool := &WriteFileTool{}

	result, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "existing.txt",
		"content": "new content",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result, "updated") {
		t.Errorf("expected result to contain 'updated', got: %s", result)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != "new content" {
		t.Errorf("expected file content 'new content', got %q", string(data))
	}
}

func TestWriteFileTool_OperationType(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(existingFile, []byte("old"), 0644)

	tool := &WriteFileTool{}

	opType := tool.OperationType(tmpDir, map[string]interface{}{
		"path": "newfile.txt",
	})
	if opType != OpFileWriteNew {
		t.Errorf("expected OpFileWriteNew, got %s", opType)
	}

	opType = tool.OperationType(tmpDir, map[string]interface{}{
		"path": "existing.txt",
	})
	if opType != OpFileWriteOverwrite {
		t.Errorf("expected OpFileWriteOverwrite, got %s", opType)
	}
}

func TestWriteFileTool_PathOutsideWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &WriteFileTool{}
	_, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "../../etc/passwd",
		"content": "malicious",
	})

	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestWriteFileTool_MissingParams(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &WriteFileTool{}

	_, err := tool.Execute(tmpDir, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing path parameter")
	}

	_, err = tool.Execute(tmpDir, map[string]interface{}{
		"path": "test.txt",
	})
	if err == nil {
		t.Fatal("expected error for missing content parameter")
	}
}

func TestWriteFileTool_NestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &WriteFileTool{}

	_, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "subdir/nested/file.txt",
		"content": "nested content",
	})

	if err != nil {
		t.Fatalf("expected no error for nested dir creation, got: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "subdir", "nested", "file.txt"))
	if err != nil {
		t.Fatalf("failed to read nested file: %v", err)
	}
	if string(data) != "nested content" {
		t.Errorf("expected 'nested content', got %q", string(data))
	}
}

func TestEditFileTool_ReplaceSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "app.go")
	os.WriteFile(testFile, []byte("package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"), 0644)

	tool := &EditFileTool{}

	result, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "app.go",
		"mode":    "replace",
		"old_str": `fmt.Println("Hello")`,
		"new_str": `fmt.Println("World")`,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result, "successfully edited") {
		t.Errorf("expected result to contain 'successfully edited', got: %s", result)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !strings.Contains(string(data), `fmt.Println("World")`) {
		t.Errorf("expected file to contain 'World', got: %s", string(data))
	}
	if strings.Contains(string(data), `fmt.Println("Hello")`) {
		t.Error("file should not contain the old string 'Hello'")
	}
}

func TestEditFileTool_ReplaceNotUnique(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "duplicate.txt")
	os.WriteFile(testFile, []byte("hello\nhello\nworld"), 0644)

	tool := &EditFileTool{}

	_, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "duplicate.txt",
		"mode":    "replace",
		"old_str": "hello",
		"new_str": "hi",
	})

	if err == nil {
		t.Fatal("expected error for non-unique old_str")
	}

	if !strings.Contains(err.Error(), "matches") {
		t.Errorf("expected error to mention 'matches', got: %v", err)
	}
}

func TestEditFileTool_ReplaceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("some content"), 0644)

	tool := &EditFileTool{}

	_, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "test.txt",
		"mode":    "replace",
		"old_str": "nonexistent",
		"new_str": "new",
	})

	if err == nil {
		t.Fatal("expected error for old_str not found")
	}
}

func TestEditFileTool_PatchSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "file.txt")
	originalContent := "line1\nline2\nline3\n"
	os.WriteFile(testFile, []byte(originalContent), 0644)

	tool := &EditFileTool{}

	patch := `@@ -1,3 +1,3 @@
 line1
-line2
+updated_line2
 line3`

	result, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":  "file.txt",
		"mode":  "patch",
		"patch": patch,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(result, "successfully patched") {
		t.Errorf("expected result to contain 'successfully patched', got: %s", result)
	}

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !strings.Contains(string(data), "updated_line2") {
		t.Errorf("expected file to contain 'updated_line2', got: %s", string(data))
	}
	if strings.Contains(string(data), "-line2") {
		t.Error("file should not contain diff markers")
	}
}

func TestEditFileTool_PatchAddLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "file.txt")
	originalContent := "line1\nline2\n"
	os.WriteFile(testFile, []byte(originalContent), 0644)

	tool := &EditFileTool{}

	patch := `@@ -1,2 +1,3 @@
 line1
 line2
+line3`

	result, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":  "file.txt",
		"mode":  "patch",
		"patch": patch,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	t.Logf("patch result: %s", result)

	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if !strings.Contains(string(data), "line3") {
		t.Errorf("expected file to contain 'line3', got: %s", string(data))
	}
}

func TestEditFileTool_PathOutsideWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &EditFileTool{}

	_, err := tool.Execute(tmpDir, map[string]interface{}{
		"path":    "../../etc/passwd",
		"mode":    "replace",
		"old_str": "test",
		"new_str": "hacked",
	})

	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestEditFileTool_UnknownMode(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &EditFileTool{}

	_, err := tool.Execute(tmpDir, map[string]interface{}{
		"path": "test.txt",
		"mode": "unknown",
	})

	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestEditFileTool_MissingParams(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &EditFileTool{}

	_, err := tool.Execute(tmpDir, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing path")
	}

	_, err = tool.Execute(tmpDir, map[string]interface{}{
		"path": "test.txt",
	})
	if err == nil {
		t.Fatal("expected error for missing mode")
	}
}

func TestRegistry_WriteFileTool(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&WriteFileTool{})

	tmpDir := t.TempDir()

	result, err := reg.Execute(tmpDir, "write_file", map[string]interface{}{
		"path":    "test.txt",
		"content": "hello from registry",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Error("expected successful result")
	}
	if !strings.Contains(result.Content, "created") {
		t.Errorf("expected 'created' in result, got: %s", result.Content)
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if string(data) != "hello from registry" {
		t.Errorf("expected 'hello from registry', got %q", string(data))
	}
}

func TestRegistry_EditFileTool(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&EditFileTool{})

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("original"), 0644)

	result, err := reg.Execute(tmpDir, "edit_file", map[string]interface{}{
		"path":    "test.txt",
		"mode":    "replace",
		"old_str": "original",
		"new_str": "modified",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Error("expected successful result")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "test.txt"))
	if string(data) != "modified" {
		t.Errorf("expected 'modified', got %q", string(data))
	}
}
