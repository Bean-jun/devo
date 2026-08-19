package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestDefaultDirTreeConfig(t *testing.T) {
	cfg := DefaultDirTreeConfig()
	if cfg.MaxDepth != 3 {
		t.Errorf("expected MaxDepth 3, got %d", cfg.MaxDepth)
	}
	if cfg.MaxFiles != 200 {
		t.Errorf("expected MaxFiles 200, got %d", cfg.MaxFiles)
	}
}

func TestGenerateDirTree_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Test"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "helper.go"), []byte("package main"), 0644)

	cfg := DefaultDirTreeConfig()
	result, err := GenerateDirTree(tmpDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "main.go") {
		t.Error("expected main.go in tree")
	}
	if !strings.Contains(result, "README.md") {
		t.Error("expected README.md in tree")
	}
	if !strings.Contains(result, "subdir/") {
		t.Error("expected subdir/ in tree")
	}
	if !strings.Contains(result, "helper.go") {
		t.Error("expected helper.go in tree")
	}
}

func TestGenerateDirTree_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultDirTreeConfig()
	result, err := GenerateDirTree(tmpDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "工作目录文件结构") {
		t.Error("expected header in result")
	}
}

func TestGenerateDirTree_ZeroValues(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main"), 0644)

	cfg := DirTreeConfig{MaxDepth: 0, MaxFiles: 0}
	result, err := GenerateDirTree(tmpDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "test.go") {
		t.Error("expected test.go in tree")
	}
}

func TestGenerateDirTree_NegativeValues(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.go"), []byte("package main"), 0644)

	cfg := DirTreeConfig{MaxDepth: -1, MaxFiles: -5}
	result, err := GenerateDirTree(tmpDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "test.go") {
		t.Error("expected test.go in tree")
	}
}

func TestGenerateDirTree_WithDotFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".hidden"), []byte("hidden"), 0644)
	os.Mkdir(filepath.Join(tmpDir, ".devo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".devo", "rules.md"), []byte("rules"), 0644)
	os.Mkdir(filepath.Join(tmpDir, ".git"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("config"), 0644)

	cfg := DefaultDirTreeConfig()
	result, err := GenerateDirTree(tmpDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, ".hidden") {
		t.Error("should not show .hidden files")
	}
	if !strings.Contains(result, ".devo/") {
		t.Error("should show .devo directory")
	}
	if strings.Contains(result, ".git") {
		t.Error("should not show .git directory")
	}
	if !strings.Contains(result, "main.go") {
		t.Error("expected main.go in tree")
	}
}

func TestGenerateDirTree_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "level1", "level2", "level3", "level4"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "level1", "level2", "level3", "level4", "deep.go"), []byte("package main"), 0644)

	cfg := DirTreeConfig{MaxDepth: 2, MaxFiles: 200}
	result, err := GenerateDirTree(tmpDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "level3") {
		t.Error("should not show level3 with MaxDepth 2")
	}
}

func TestGenerateDirTree_MaxFiles(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 10; i++ {
		name := string(rune('a'+i)) + ".go"
		os.WriteFile(filepath.Join(tmpDir, name), []byte("package main"), 0644)
	}

	cfg := DirTreeConfig{MaxDepth: 3, MaxFiles: 5}
	result, err := GenerateDirTree(tmpDir, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "输出被截断") {
		t.Error("expected truncation message")
	}
}

func TestGenerateDirTree_InvalidDir(t *testing.T) {
	cfg := DefaultDirTreeConfig()
	_, err := GenerateDirTree("/nonexistent/path/12345", cfg)
	if err != nil {
		t.Errorf("expected no error (walkDirTree handles missing dir), got %v", err)
	}
}

func TestIsDirTreeChanged_NilCache(t *testing.T) {
	tmpDir := t.TempDir()
	changed, err := IsDirTreeChanged(tmpDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when cache is nil")
	}
}

func TestIsDirTreeChanged_InvalidCache(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &session.DirectorySummary{Valid: false}
	changed, err := IsDirTreeChanged(tmpDir, cache)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when cache is invalid")
	}
}

func TestIsDirTreeChanged_NotChanged(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &session.DirectorySummary{
		Content:     "old content",
		GeneratedAt: time.Now().Add(1 * time.Hour),
		Valid:       true,
	}
	changed, err := IsDirTreeChanged(tmpDir, cache)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Error("expected changed=false when cache is newer than dir")
	}
}

func TestIsDirTreeChanged_Changed(t *testing.T) {
	tmpDir := t.TempDir()
	cache := &session.DirectorySummary{
		Content:     "old content",
		GeneratedAt: time.Now().Add(-1 * time.Hour),
		Valid:       true,
	}
	changed, err := IsDirTreeChanged(tmpDir, cache)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when cache is older than dir")
	}
}

func TestIsDirTreeChanged_InvalidDir(t *testing.T) {
	cache := &session.DirectorySummary{
		Content:     "old content",
		GeneratedAt: time.Now().Add(1 * time.Hour),
		Valid:       true,
	}
	changed, err := IsDirTreeChanged("/nonexistent/path/12345", cache)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Error("expected changed=true for invalid dir")
	}
}

func TestNewDirectorySummary(t *testing.T) {
	summary := NewDirectorySummary("test content")
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.Content != "test content" {
		t.Errorf("expected 'test content', got %q", summary.Content)
	}
	if !summary.Valid {
		t.Error("expected Valid=true")
	}
	if summary.GeneratedAt.IsZero() {
		t.Error("expected non-zero GeneratedAt")
	}
}
