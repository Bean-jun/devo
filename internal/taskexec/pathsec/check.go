package pathsec

import (
	"errors"
	"path/filepath"
	"strings"
)

var (
	ErrPathOutsideWorkDir = errors.New("path is outside working directory")
	ErrPathNormalization  = errors.New("failed to normalize path")
)

// NormalizePath normalizes a path for consistent comparison.
// On Windows, this ensures the drive letter is uppercase (e.g., "C:\" instead of "c:\").
func NormalizePath(p string) string {
	p = filepath.Clean(p)
	if len(p) >= 2 && p[1] == ':' {
		p = strings.ToUpper(p[:1]) + p[1:]
	}
	return p
}

// isRealAbsPath checks if a path is a "real" absolute path.
// On Windows, paths starting with "/" (e.g., "/foo/bar") are considered absolute
// by filepath.IsAbs, but they are not truly absolute since they lack a drive letter.
// Such paths should be treated as relative and joined with the working directory.
func isRealAbsPath(p string) bool {
	if !filepath.IsAbs(p) {
		return false
	}
	if len(p) >= 2 && p[1] == ':' {
		return true
	}
	if len(p) >= 2 && p[0] == '\\' && p[1] == '\\' {
		return true
	}
	return false
}

func CheckPath(workingDir, requestedPath string) (string, error) {
	absWorkDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", ErrPathNormalization
	}
	absWorkDir = NormalizePath(absWorkDir)

	var absRequested string
	if isRealAbsPath(requestedPath) {
		absRequested = requestedPath
	} else {
		absRequested = filepath.Join(absWorkDir, requestedPath)
	}

	absRequested, err = filepath.Abs(absRequested)
	if err != nil {
		return "", ErrPathNormalization
	}
	absRequested = NormalizePath(absRequested)

	// A path is inside the workspace if it equals the workspace root or is a
	// descendant of it. Comparing against absWorkDir+separator (rather than
	// appending a separator to absWorkDir and using HasPrefix) ensures the
	// exact-root case (requestedPath == "." or the absolute workdir itself)
	// passes instead of being rejected for lacking a trailing separator.
	sep := string(filepath.Separator)
	if absRequested != absWorkDir && !strings.HasPrefix(absRequested, absWorkDir+sep) {
		return "", ErrPathOutsideWorkDir
	}

	return absRequested, nil
}

func CheckRelativePath(workingDir, requestedPath string) (string, error) {
	absPath, err := CheckPath(workingDir, requestedPath)
	if err != nil {
		return "", err
	}

	absWorkDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", ErrPathNormalization
	}
	absWorkDir = NormalizePath(absWorkDir)

	rel, err := filepath.Rel(absWorkDir, absPath)
	if err != nil {
		return "", ErrPathNormalization
	}

	return rel, nil
}
