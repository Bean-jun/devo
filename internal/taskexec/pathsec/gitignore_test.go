package pathsec

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMatchExact_PatternLongerThanPath is the regression test for the panic
// "slice bounds out of range [-N:]" that occurred when a multi-segment
// .gitignore pattern (e.g. "tests/mock_server/requests") was checked against
// a single-segment path (e.g. "node_modules"). The panic was unreachable
// until CheckPath was fixed to allow listing the workspace root itself.
func TestMatchExact_PatternLongerThanPath(t *testing.T) {
	cases := []struct {
		name     string
		parts    []string
		pathPart []string
	}{
		{"pattern 3 parts, path 1 part", []string{"tests", "mock_server", "requests"}, []string{"node_modules"}},
		{"pattern 3 parts, path 2 parts", []string{"a", "b", "c"}, []string{"x", "y"}},
		{"pattern 2 parts, empty path", []string{"foo", "bar"}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Must not panic and must return false (a longer anchored pattern
			// can never match a shorter path).
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("matchExact panicked: %v", r)
				}
			}()
			if got := matchExact(tc.parts, tc.pathPart); got {
				t.Errorf("matchExact(%v, %v) = true, want false", tc.parts, tc.pathPart)
			}
		})
	}
}

func TestMatchExact_EqualLengthMatch(t *testing.T) {
	if !matchExact([]string{"foo", "bar"}, []string{"foo", "bar"}) {
		t.Errorf("exact equal path should match")
	}
	if matchExact([]string{"foo", "bar"}, []string{"foo", "baz"}) {
		t.Errorf("differing last segment should not match")
	}
}

func TestMatchExact_PatternShorterThanPath(t *testing.T) {
	// Anchored pattern with "/" should not match a deeper path.
	if matchExact([]string{"foo", "bar"}, []string{"x", "foo", "bar"}) {
		t.Errorf("shorter pattern should not match deeper path")
	}
}

// TestGitignoreIsIgnored_RealWorldPatterns exercises the actual code path
// that panicked: a project .gitignore with multi-segment directory rules
// being queried with single-segment sibling paths during a root listing.
func TestGitignoreIsIgnored_RealWorldPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, ".gitignore"),
		[]byte("node_modules/\ntests/mock_server/requests/\n*.log\nbuild/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	gi := LoadGitignore(tmpDir)

	cases := []struct {
		relPath string
		isDir   bool
		want    bool
	}{
		// Single-segment path vs multi-segment pattern: must not panic, must not match.
		{"node_modules", true, true},
		{"tests", true, false},
		{"build", true, true},
		// Multi-segment exact match.
		{"tests/mock_server/requests", true, true},
		// Glob at any depth (single-segment pattern, no slash).
		{"foo/bar/debug.log", false, true},
		{"foo/bar/keep.txt", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.relPath, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("IsIgnored(%q) panicked: %v", tc.relPath, r)
				}
			}()
			if got := gi.IsIgnored(tc.relPath, tc.isDir); got != tc.want {
				t.Errorf("IsIgnored(%q, isDir=%v) = %v, want %v", tc.relPath, tc.isDir, got, tc.want)
			}
		})
	}
}
