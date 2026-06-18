package pathsec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckPath_ValidRelativePath(t *testing.T) {
	tmpDir := t.TempDir()

	absPath, err := CheckPath(tmpDir, "test.txt")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := filepath.Join(tmpDir, "test.txt")
	if absPath != expected {
		t.Errorf("expected %q, got %q", expected, absPath)
	}
}

func TestCheckPath_ValidSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	absPath, err := CheckPath(tmpDir, "subdir/file.txt")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := filepath.Join(tmpDir, "subdir", "file.txt")
	if absPath != expected {
		t.Errorf("expected %q, got %q", expected, absPath)
	}
}

func TestCheckPath_ValidAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	absPath, err := CheckPath(tmpDir, subDir)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if absPath != subDir {
		t.Errorf("expected %q, got %q", subDir, absPath)
	}
}

func TestCheckPath_TraversalAttempt(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := CheckPath(tmpDir, "../../etc/passwd")
	if err != ErrPathOutsideWorkDir {
		t.Errorf("expected ErrPathOutsideWorkDir, got: %v", err)
	}
}

func TestCheckPath_AbsolutePathOutsideWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	absPath := filepath.Join(tmpDir, "..", "..", "..", "..", "Windows", "System32", "drivers", "etc", "hosts")
	absPath, _ = filepath.Abs(absPath)

	_, err := CheckPath(tmpDir, absPath)
	if err != ErrPathOutsideWorkDir {
		t.Errorf("expected ErrPathOutsideWorkDir, got: %v", err)
	}
}

func TestCheckPath_DotDotInSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(subDir, 0755)

	_, err := CheckPath(tmpDir, "subdir/../../etc/passwd")
	if err != ErrPathOutsideWorkDir {
		t.Errorf("expected ErrPathOutsideWorkDir, got: %v", err)
	}
}

func TestCheckPath_CurrentDir(t *testing.T) {
	tmpDir := t.TempDir()

	absPath, err := CheckPath(tmpDir, ".")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if absPath != tmpDir {
		t.Errorf("expected %q, got %q", tmpDir, absPath)
	}
}

func TestCheckRelativePath_Valid(t *testing.T) {
	tmpDir := t.TempDir()

	rel, err := CheckRelativePath(tmpDir, "test.txt")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if rel != "test.txt" {
		t.Errorf("expected 'test.txt', got %q", rel)
	}
}

func TestCheckRelativePath_Traversal(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := CheckRelativePath(tmpDir, "../../etc/passwd")
	if err != ErrPathOutsideWorkDir {
		t.Errorf("expected ErrPathOutsideWorkDir, got: %v", err)
	}
}
