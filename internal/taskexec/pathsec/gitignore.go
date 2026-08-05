package pathsec

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type gitignoreRule struct {
	pattern string
	negate  bool
	dirOnly bool
}

type Gitignore struct {
	rules []gitignoreRule
}

func LoadGitignore(workDir string) *Gitignore {
	gi := &Gitignore{}

	gitignorePath := filepath.Join(workDir, ".gitignore")
	f, err := os.Open(gitignorePath)
	if err != nil {
		return gi
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		negate := false
		if strings.HasPrefix(line, "!") {
			negate = true
			line = line[1:]
		}

		dirOnly := false
		if strings.HasSuffix(line, "/") {
			dirOnly = true
			line = line[:len(line)-1]
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		gi.rules = append(gi.rules, gitignoreRule{
			pattern: line,
			negate:  negate,
			dirOnly: dirOnly,
		})
	}

	return gi
}

func (gi *Gitignore) IsIgnored(relPath string, isDir bool) bool {
	if gi == nil || len(gi.rules) == 0 {
		return false
	}

	relPath = filepath.ToSlash(relPath)
	matched := false

	for _, rule := range gi.rules {
		if rule.dirOnly && !isDir {
			continue
		}

		if match(rule.pattern, relPath) {
			matched = !rule.negate
		}
	}

	return matched
}

func match(pattern, path string) bool {
	pattern = strings.TrimPrefix(pattern, "/")

	parts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(parts) == 1 && !strings.Contains(parts[0], "/") {
		filename := pathParts[len(pathParts)-1]
		return matchPattern(parts[0], filename)
	}

	if strings.Contains(pattern, "**") {
		return matchDoubleStar(pattern, path)
	}

	return matchExact(parts, pathParts)
}

func matchPattern(pattern, name string) bool {
	matched, _ := filepath.Match(pattern, name)
	return matched
}

func matchDoubleStar(pattern, path string) bool {
	idx := strings.Index(pattern, "**")
	if idx == -1 {
		return false
	}

	prefix := strings.TrimSuffix(pattern[:idx], "/")
	suffix := strings.TrimPrefix(pattern[idx+2:], "/")

	path = strings.TrimPrefix(path, prefix)
	if prefix != "" && !strings.HasPrefix(path, prefix) {
		if !strings.HasPrefix(path, prefix) {
			pathParts := strings.Split(path, "/")
			prefixParts := strings.Split(prefix, "/")
			if len(pathParts) < len(prefixParts) {
				return false
			}
			joined := strings.Join(pathParts[:len(prefixParts)], "/")
			if joined != prefix {
				return false
			}
			path = strings.Join(pathParts[len(prefixParts):], "/")
		}
	}

	if suffix == "" {
		return true
	}

	pathParts := strings.Split(path, "/")
	suffixParts := strings.Split(suffix, "/")
	if len(pathParts) < len(suffixParts) {
		return false
	}

	for i := 0; i <= len(pathParts)-len(suffixParts); i++ {
		ok := true
		for j, sp := range suffixParts {
			if !matchPattern(sp, pathParts[i+j]) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func matchExact(parts, pathParts []string) bool {
	// Patterns containing "/" are anchored: they only match a path of the same
	// depth. A pattern longer than the path can never match (the slice
	// arithmetic below used to panic with [-N:] here); a pattern shorter than
	// the path doesn't match either, since the pattern is not a prefix.
	if len(parts) != len(pathParts) {
		return false
	}
	for i := range parts {
		if !matchPattern(parts[i], pathParts[i]) {
			return false
		}
	}
	return true
}
