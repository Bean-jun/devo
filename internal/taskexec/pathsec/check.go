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

func CheckPath(workingDir, requestedPath string) (string, error) {
	absWorkDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", ErrPathNormalization
	}

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

	rel, err := filepath.Rel(absWorkDir, absPath)
	if err != nil {
		return "", ErrPathNormalization
	}

	return rel, nil
}
