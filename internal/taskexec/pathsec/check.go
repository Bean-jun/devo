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

func CheckPath(workingDir, requestedPath string) (string, error) {
	absWorkDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", ErrPathNormalization
	}
	absWorkDir = NormalizePath(absWorkDir)

	var absRequested string
	if filepath.IsAbs(requestedPath) {
		absRequested = requestedPath
	} else {
		absRequested = filepath.Join(absWorkDir, requestedPath)
	}

	absRequested, err = filepath.Abs(absRequested)
	if err != nil {
		return "", ErrPathNormalization
	}
	absRequested = NormalizePath(absRequested)

	if !strings.HasPrefix(absRequested, absWorkDir) {
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
