package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateUnifiedDiff_SingleLineChange(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nline2_changed\nline3\n"

	diff := generateUnifiedDiff(old, new)

	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ header")
	}
	if !strings.Contains(diff, "-line2") {
		t.Error("diff should contain removed line")
	}
	if !strings.Contains(diff, "+line2_changed") {
		t.Error("diff should contain added line")
	}
}

func TestGenerateUnifiedDiff_MultipleChanges(t *testing.T) {
	old := "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n"
	new := "package main\n\nfunc main() {\n\tfmt.Println(\"World\")\n\tfmt.Println(\"Done\")\n}\n"

	diff := generateUnifiedDiff(old, new)

	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ header")
	}
	if !strings.Contains(diff, "Hello") {
		t.Error("diff should mention the removed content")
	}
	if !strings.Contains(diff, "World") {
		t.Error("diff should mention the added content")
	}
}

func TestGenerateUnifiedDiff_NoChange(t *testing.T) {
	content := "same\ncontent\n"
	diff := generateUnifiedDiff(content, content)

	if diff != "" {
		t.Errorf("expected empty diff for identical content, got: %s", diff)
	}
}

func TestGenerateUnifiedDiff_AddLine(t *testing.T) {
	old := "line1\nline2\n"
	new := "line1\nline2\nline3\n"

	diff := generateUnifiedDiff(old, new)

	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ header")
	}
	if !strings.Contains(diff, "+line3") {
		t.Error("diff should contain added line3")
	}
}

func TestGenerateUnifiedDiff_RemoveLine(t *testing.T) {
	old := "line1\nline2\nline3\n"
	new := "line1\nline3\n"

	diff := generateUnifiedDiff(old, new)

	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ header")
	}
	if !strings.Contains(diff, "-line2") {
		t.Error("diff should contain removed line2")
	}
}

func TestWriteFileTool_PreviewDiff_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	oldContent := "line1\nline2\nline3\n"
	os.WriteFile(existingFile, []byte(oldContent), 0644)

	tool := &WriteFileTool{}

	newContent := "line1\nline2_modified\nline3\n"
	diff, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":    "existing.txt",
		"content": newContent,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if diff == "" {
		t.Error("expected non-empty diff for overwrite")
	}

	if !strings.Contains(diff, "@@") {
		t.Error("diff should contain @@ header")
	}

	if !strings.Contains(diff, "-line2") {
		t.Error("diff should contain removed line")
	}

	if !strings.Contains(diff, "+line2_modified") {
		t.Error("diff should contain added line")
	}

	data, _ := os.ReadFile(existingFile)
	if string(data) != oldContent {
		t.Error("PreviewDiff should not modify the file on disk")
	}
}

func TestWriteFileTool_PreviewDiff_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &WriteFileTool{}

	diff, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":    "newfile.txt",
		"content": "new content",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if diff != "" {
		t.Errorf("expected empty diff for new file, got: %s", diff)
	}
}

func TestWriteFileTool_PreviewDiff_SameContent(t *testing.T) {
	tmpDir := t.TempDir()
	content := "same content"
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(content), 0644)

	tool := &WriteFileTool{}

	diff, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":    "file.txt",
		"content": content,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if diff != "" {
		t.Errorf("expected empty diff when content is unchanged, got: %s", diff)
	}
}

func TestEditFileTool_PreviewDiff_Replace(t *testing.T) {
	tmpDir := t.TempDir()
	oldContent := "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n"
	os.WriteFile(filepath.Join(tmpDir, "app.go"), []byte(oldContent), 0644)

	tool := &EditFileTool{}

	diff, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":    "app.go",
		"mode":    "replace",
		"old_str": `fmt.Println("Hello")`,
		"new_str": `fmt.Println("World")`,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if diff == "" {
		t.Error("expected non-empty diff for replace")
	}

	if !strings.Contains(diff, "Hello") {
		t.Error("diff should mention the old content")
	}
	if !strings.Contains(diff, "World") {
		t.Error("diff should mention the new content")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "app.go"))
	if string(data) != oldContent {
		t.Error("PreviewDiff should not modify the file on disk")
	}
}

func TestEditFileTool_PreviewDiff_ReplaceNotUnique(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "dup.txt"), []byte("hello\nhello\n"), 0644)

	tool := &EditFileTool{}

	_, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":    "dup.txt",
		"mode":    "replace",
		"old_str": "hello",
		"new_str": "hi",
	})

	if err == nil {
		t.Fatal("expected error for non-unique old_str in PreviewDiff")
	}

	if !strings.Contains(err.Error(), "matches") {
		t.Errorf("expected error to mention 'matches', got: %v", err)
	}
}

func TestEditFileTool_PreviewDiff_ReplaceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("some content"), 0644)

	tool := &EditFileTool{}

	_, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":    "test.txt",
		"mode":    "replace",
		"old_str": "nonexistent",
		"new_str": "new",
	})

	if err == nil {
		t.Fatal("expected error for old_str not found in PreviewDiff")
	}
}

func TestEditFileTool_PreviewDiff_Patch(t *testing.T) {
	tmpDir := t.TempDir()
	oldContent := "line1\nline2\nline3\n"
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte(oldContent), 0644)

	tool := &EditFileTool{}

	patch := `@@ -1,3 +1,3 @@
 line1
-line2
+updated_line2
 line3`

	diff, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":  "file.txt",
		"mode":  "patch",
		"patch": patch,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if diff == "" {
		t.Error("expected non-empty diff for patch")
	}

	if !strings.Contains(diff, "-line2") {
		t.Error("diff should contain removed line")
	}
	if !strings.Contains(diff, "+updated_line2") {
		t.Error("diff should contain added line")
	}

	data, _ := os.ReadFile(filepath.Join(tmpDir, "file.txt"))
	if string(data) != oldContent {
		t.Error("PreviewDiff should not modify the file on disk")
	}
}

func TestEditFileTool_PreviewDiff_PatchFailure(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("line1\nline2\n"), 0644)

	tool := &EditFileTool{}

	patch := `@@ -1,2 +1,2 @@
 line1
-wrong_line
+new_line`

	_, err := tool.PreviewDiff(tmpDir, map[string]interface{}{
		"path":  "file.txt",
		"mode":  "patch",
		"patch": patch,
	})

	if err == nil {
		t.Fatal("expected error when patch cannot be applied")
	}
}

func TestEditFileTool_PreCheck_MissingParams(t *testing.T) {
	tool := &EditFileTool{}

	err := tool.PreCheck(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing path")
	}

	err = tool.PreCheck(map[string]interface{}{
		"path": "test.txt",
	})
	if err == nil {
		t.Fatal("expected error for missing mode")
	}

	err = tool.PreCheck(map[string]interface{}{
		"path": "test.txt",
		"mode": "replace",
	})
	if err == nil {
		t.Fatal("expected error for missing old_str in replace mode")
	}

	err = tool.PreCheck(map[string]interface{}{
		"path": "test.txt",
		"mode": "patch",
	})
	if err == nil {
		t.Fatal("expected error for missing patch in patch mode")
	}
}

func TestEditFileTool_PreCheck_UnknownMode(t *testing.T) {
	tool := &EditFileTool{}

	err := tool.PreCheck(map[string]interface{}{
		"path": "test.txt",
		"mode": "unknown",
	})
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
}

func TestEditFileTool_PreCheck_Valid(t *testing.T) {
	tool := &EditFileTool{}

	err := tool.PreCheck(map[string]interface{}{
		"path":    "test.txt",
		"mode":    "replace",
		"old_str": "hello",
		"new_str": "world",
	})
	if err != nil {
		t.Fatalf("expected no error for valid replace params, got: %v", err)
	}

	err = tool.PreCheck(map[string]interface{}{
		"path":  "test.txt",
		"mode":  "patch",
		"patch": "@@ -1 +1 @@\n-old\n+new\n",
	})
	if err != nil {
		t.Fatalf("expected no error for valid patch params, got: %v", err)
	}
}

func TestExecPythonTool_GetCommandContext(t *testing.T) {
	tool := NewExecPythonTool()

	ctx := tool.GetCommandContext("/tmp/test", map[string]interface{}{
		"code":            "print('hello')",
		"timeout_seconds": float64(60),
	})

	if ctx["working_directory"] != "/tmp/test" {
		t.Errorf("expected working_directory /tmp/test, got %v", ctx["working_directory"])
	}

	invocation, ok := ctx["invocation"].(string)
	if !ok {
		t.Fatal("expected invocation to be a string")
	}
	if !strings.Contains(invocation, "python") {
		t.Errorf("expected invocation to mention python, got %v", invocation)
	}

	if ctx["timeout_seconds"] != 60 {
		t.Errorf("expected timeout_seconds 60, got %v", ctx["timeout_seconds"])
	}
}

func TestExecPythonTool_GetCommandContext_DefaultTimeout(t *testing.T) {
	tool := NewExecPythonTool()

	ctx := tool.GetCommandContext("/tmp/test", map[string]interface{}{
		"code": "print('hello')",
	})

	if ctx["timeout_seconds"] != 30 {
		t.Errorf("expected default timeout 30, got %v", ctx["timeout_seconds"])
	}
}

func TestDiffPreviewer_Interface(t *testing.T) {
	var _ DiffPreviewer = (*WriteFileTool)(nil)
	var _ DiffPreviewer = (*EditFileTool)(nil)
}

func TestCommandContextProvider_Interface(t *testing.T) {
	var _ CommandContextProvider = (*ExecPythonTool)(nil)
}

func TestPreChecker_EditFileTool(t *testing.T) {
	var _ PreChecker = (*EditFileTool)(nil)
}