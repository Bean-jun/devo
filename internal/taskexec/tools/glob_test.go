package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGlobTool_AllGoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "utils.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# readme"), 0644)

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "*.go",
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
	if !strings.Contains(result.Content, "utils.go") {
		t.Error("expected output to contain utils.go")
	}
	if strings.Contains(result.Content, "README.md") {
		t.Error("expected output to NOT contain README.md")
	}
}

func TestGlobTool_RecursivePattern(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "src"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "src", "sub"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "src", "sub", "helper.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "root.go"), []byte(""), 0644)

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "**/*.go",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "root.go") {
		t.Error("expected output to contain root.go")
	}
	if !strings.Contains(result.Content, "main.go") {
		t.Error("expected output to contain src/main.go")
	}
	if !strings.Contains(result.Content, "helper.go") {
		t.Error("expected output to contain src/sub/helper.go")
	}
}

func TestGlobTool_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(""), 0644)

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "*.py",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "No files found") {
		t.Errorf("expected 'No files found', got: %q", result.Content)
	}
}

func TestGlobTool_SpecificPath(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "src"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "lib"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "src", "main.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "lib", "utils.go"), []byte(""), 0644)

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "src/*.go",
		"path":    "src",
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
	if strings.Contains(result.Content, "utils.go") {
		t.Error("expected output to NOT contain utils.go (not in src/)")
	}
}

func TestGlobTool_MissingPattern(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for missing pattern parameter")
	}
}

func TestGlobTool_PathOutsideWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "*",
		"path":    "../../etc",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for path traversal")
	}
}

func TestGlobTool_TestFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "foo_test.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "bar_test.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "foo.go"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "bar.go"), []byte(""), 0644)

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "*_test.go",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if !strings.Contains(result.Content, "foo_test.go") {
		t.Error("expected output to contain foo_test.go")
	}
	if !strings.Contains(result.Content, "bar_test.go") {
		t.Error("expected output to contain bar_test.go")
	}
	if strings.Contains(result.Content, "foo.go") && !strings.Contains(result.Content, "foo_test.go") {
		t.Error("expected output to NOT contain foo.go")
	}
}

func TestGlobTool_SkipsHiddenDirs(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, ".git"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(""), 0644)

	tool := &GlobTool{}
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"pattern": "**/*",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	if strings.Contains(result.Content, ".git") {
		t.Error("expected output to NOT contain .git directory")
	}
	if !strings.Contains(result.Content, "main.go") {
		t.Error("expected output to contain main.go")
	}
}
