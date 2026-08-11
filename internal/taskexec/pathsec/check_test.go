package pathsec

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsRealAbsPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "Windows drive letter path",
			path:     `D:\devo\projects\devo\build`,
			expected: runtime.GOOS == "windows",
		},
		{
			name:     "Windows lowercase drive letter",
			path:     `c:\Users\test`,
			expected: runtime.GOOS == "windows",
		},
		{
			name:     "UNC path",
			path:     `\\server\share\file`,
			expected: runtime.GOOS == "windows",
		},
		{
			name:     "Unix-style absolute path (slash start)",
			path:     `/devo-windows-amd64.exe`,
			expected: false,
		},
		{
			name:     "relative path",
			path:     `devo-windows-amd64.exe`,
			expected: false,
		},
		{
			name:     "relative path with dir",
			path:     `subdir/file.txt`,
			expected: false,
		},
		{
			name:     "Unix-style absolute with subdir",
			path:     `/usr/local/bin`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRealAbsPath(tt.path)
			if result != tt.expected {
				t.Errorf("isRealAbsPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestCheckPath_SlashPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "build")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(workDir, "devo-windows-amd64.exe")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := CheckPath(workDir, "/devo-windows-amd64.exe")
	if err != nil {
		t.Fatalf("CheckPath with slash prefix should succeed: %v", err)
	}

	expected := filepath.Join(workDir, "devo-windows-amd64.exe")
	resultAbs, _ := filepath.Abs(result)
	expectedAbs, _ := filepath.Abs(expected)
	if resultAbs != expectedAbs {
		t.Errorf("CheckPath result = %q, want %q", resultAbs, expectedAbs)
	}
}

func TestCheckPath_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "build")
	subDir := filepath.Join(workDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := CheckPath(workDir, "subdir/file.txt")
	if err != nil {
		t.Fatalf("CheckPath with relative path should succeed: %v", err)
	}

	resultAbs, _ := filepath.Abs(result)
	expectedAbs, _ := filepath.Abs(testFile)
	if resultAbs != expectedAbs {
		t.Errorf("CheckPath result = %q, want %q", resultAbs, expectedAbs)
	}
}

func TestCheckPath_AbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(workDir, "main.go")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := CheckPath(workDir, testFile)
	if err != nil {
		t.Fatalf("CheckPath with absolute path should succeed: %v", err)
	}

	resultAbs, _ := filepath.Abs(result)
	expectedAbs, _ := filepath.Abs(testFile)
	if runtime.GOOS == "windows" {
		if resultAbs != expectedAbs {
			t.Errorf("CheckPath result = %q, want %q", resultAbs, expectedAbs)
		}
	} else {
		expectedJoin, _ := filepath.Abs(filepath.Join(workDir, testFile))
		if resultAbs != expectedJoin {
			t.Errorf("CheckPath result = %q, want %q", resultAbs, expectedJoin)
		}
	}
}

func TestCheckPath_SiblingDir(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "devo")
	siblingDir := filepath.Join(tmpDir, "devo2")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(siblingDir, 0755); err != nil {
		t.Fatal(err)
	}

	siblingFile := filepath.Join(siblingDir, "main.go")
	if err := os.WriteFile(siblingFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := CheckPath(workDir, siblingFile)
	if runtime.GOOS == "windows" {
		if err != ErrPathOutsideWorkDir {
			t.Errorf("CheckPath with sibling dir should return ErrPathOutsideWorkDir, got: %v", err)
		}
	} else {
		if err != nil {
			t.Errorf("CheckPath with sibling dir should succeed on non-Windows, got: %v", err)
		}
	}
}

func TestCheckPath_ParentDir(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "project", "subdir")
	parentFile := filepath.Join(tmpDir, "project", "main.go")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parentFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := CheckPath(workDir, parentFile)
	if runtime.GOOS == "windows" {
		if err != ErrPathOutsideWorkDir {
			t.Errorf("CheckPath with parent dir should return ErrPathOutsideWorkDir, got: %v", err)
		}
	} else {
		if err != nil {
			t.Errorf("CheckPath with parent dir should succeed on non-Windows, got: %v", err)
		}
	}
}

// TestCheckPath_WorkDirRoot covers the regression where requesting the
// workspace directory itself (via "." or the absolute path) was rejected.
// The trailing-separator HasPrefix check used to fail because the requested
// path has no trailing separator - this guards against that bug coming back.
func TestCheckPath_WorkDirRoot(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "project")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		path string
	}{
		{"dot", "."},
		{"dotSlash", "./"},
		{"absoluteWorkDir", workDir},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := CheckPath(workDir, tc.path)
			if err != nil {
				t.Fatalf("CheckPath(%q) should succeed: %v", tc.path, err)
			}
			resultAbs, _ := filepath.Abs(result)
			workDirAbs, _ := filepath.Abs(workDir)
			if runtime.GOOS == "windows" {
				if resultAbs != workDirAbs {
					t.Errorf("CheckPath(%q) = %q, want %q", tc.path, resultAbs, workDirAbs)
				}
			} else {
				expectedJoin, _ := filepath.Abs(filepath.Join(workDir, tc.path))
				if resultAbs != expectedJoin {
					t.Errorf("CheckPath(%q) = %q, want %q", tc.path, resultAbs, expectedJoin)
				}
			}
		})
	}
}

func TestCheckPath_WindowsRelativePaths(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "devo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		relPath    string
		createFile string
	}{
		{
			name:       "forward slash relative path",
			relPath:    "docs/问题排查与修复方案.md",
			createFile: "docs/问题排查与修复方案.md",
		},
		{
			name:       "backslash relative path",
			relPath:    `tests\knowledge-base\resource-platform\TC-UPLOAD-011_1061.spec.js`,
			createFile: `tests\knowledge-base\resource-platform\TC-UPLOAD-011_1061.spec.js`,
		},
		{
			name:       "forward slash with subdirs",
			relPath:    "internal/taskexec/pathsec/check.go",
			createFile: "internal/taskexec/pathsec/check.go",
		},
		{
			name:       "simple forward slash filename",
			relPath:    "问题排查与修复方案.md",
			createFile: "问题排查与修复方案.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(workDir, tt.createFile)
			dir := filepath.Dir(fullPath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}

			result, err := CheckPath(workDir, tt.relPath)
			if err != nil {
				t.Fatalf("CheckPath(%q) should succeed: %v", tt.relPath, err)
			}

			resultAbs, _ := filepath.Abs(result)
			expectedAbs, _ := filepath.Abs(fullPath)
			if resultAbs != expectedAbs {
				t.Errorf("CheckPath(%q) = %q, want %q", tt.relPath, resultAbs, expectedAbs)
			}
		})
	}
}

func TestCheckPath_MixedSeparatorRelativePath(t *testing.T) {
	tmpDir := t.TempDir()

	workDir := filepath.Join(tmpDir, "devo")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(workDir, "src", "utils", "helper.go")
	dir := filepath.Dir(testFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := CheckPath(workDir, "src/utils/helper.go")
	if err != nil {
		t.Fatalf("CheckPath with mixed separators should succeed: %v", err)
	}

	resultAbs, _ := filepath.Abs(result)
	expectedAbs, _ := filepath.Abs(testFile)
	if resultAbs != expectedAbs {
		t.Errorf("CheckPath result = %q, want %q", resultAbs, expectedAbs)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase drive letter",
			input:    `d:\devo\project`,
			expected: `D:\devo\project`,
		},
		{
			name:     "already uppercase",
			input:    `D:\devo\project`,
			expected: `D:\devo\project`,
		},
		{
			name:  "unix path (Windows will convert to backslash)",
			input: `/usr/local`,
			expected: func() string {
				if runtime.GOOS == "windows" {
					return `\usr\local`
				}
				return `/usr/local`
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
