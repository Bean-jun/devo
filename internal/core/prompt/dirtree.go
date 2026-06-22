package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"devo/internal/core/session"
)

const (
	defaultMaxDepth = 3
	defaultMaxFiles = 200
)

type DirTreeConfig struct {
	MaxDepth int
	MaxFiles int
}

func DefaultDirTreeConfig() DirTreeConfig {
	return DirTreeConfig{
		MaxDepth: defaultMaxDepth,
		MaxFiles: defaultMaxFiles,
	}
}

func GenerateDirTree(workingDir string, config DirTreeConfig) (string, error) {
	absWorkDir, err := filepath.Abs(workingDir)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	if config.MaxDepth <= 0 {
		config.MaxDepth = defaultMaxDepth
	}
	if config.MaxFiles <= 0 {
		config.MaxFiles = defaultMaxFiles
	}

	var result strings.Builder
	result.WriteString("工作目录文件结构：\n")
	result.WriteString(fmt.Sprintf("%s/\n", filepath.Base(absWorkDir)))

	count := 0
	err = walkDirTree(&result, absWorkDir, absWorkDir, "", config.MaxDepth, 0, config.MaxFiles, &count)
	if err != nil {
		return "", err
	}

	if count >= config.MaxFiles {
		result.WriteString(fmt.Sprintf("\n... (输出被截断，已显示 %d 个条目)", config.MaxFiles))
	}

	return result.String(), nil
}

func walkDirTree(result *strings.Builder, absWorkDir, currentPath, prefix string, maxDepth, currentDepth, maxFiles int, count *int) error {
	if currentDepth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil
	}

	type entry struct {
		name  string
		isDir bool
	}

	var sorted []entry
	for _, e := range entries {
		name := e.Name()
		if name == ".git" {
			continue
		}
		if strings.HasPrefix(name, ".") && name != ".devo" {
			continue
		}
		sorted = append(sorted, entry{name: name, isDir: e.IsDir()})
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].isDir != sorted[j].isDir {
			return sorted[i].isDir
		}
		return sorted[i].name < sorted[j].name
	})

	for i, e := range sorted {
		if *count >= maxFiles {
			return nil
		}

		isLast := i == len(sorted)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		if e.isDir {
			result.WriteString(fmt.Sprintf("%s%s%s/\n", prefix, connector, e.name))
		} else {
			result.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, e.name))
		}
		*count++

		if e.isDir && currentDepth+1 < maxDepth {
			subPath := filepath.Join(currentPath, e.name)
			walkDirTree(result, absWorkDir, subPath, prefix+childPrefix, maxDepth, currentDepth+1, maxFiles, count)
		}
	}

	return nil
}

func IsDirTreeChanged(workingDir string, cachedSummary *session.DirectorySummary) (bool, error) {
	if cachedSummary == nil || !cachedSummary.Valid {
		return true, nil
	}

	absWorkDir, err := filepath.Abs(workingDir)
	if err != nil {
		return true, fmt.Errorf("resolve working directory: %w", err)
	}

	info, err := os.Stat(absWorkDir)
	if err != nil {
		return true, nil
	}

	if info.ModTime().After(cachedSummary.GeneratedAt) {
		return true, nil
	}

	return false, nil
}

func NewDirectorySummary(content string) *session.DirectorySummary {
	return &session.DirectorySummary{
		Content:     content,
		GeneratedAt: time.Now(),
		Valid:       true,
	}
}
