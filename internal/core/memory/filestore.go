package memory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileStore struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create memory dir: %w", err)
	}
	return &FileStore{baseDir: baseDir}, nil
}

func DefaultFileStore() (*FileStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	return NewFileStore(filepath.Join(homeDir, ".devo", "memory"))
}

func (s *FileStore) filePath(memoryType MemoryType, fileKey string) string {
	switch memoryType {
	case TypeUser:
		return filepath.Join(s.baseDir, fmt.Sprintf("user-%s.md", fileKey))
	case TypeProject:
		projDir := filepath.Join(s.baseDir, "projects")
		os.MkdirAll(projDir, 0755)
		return filepath.Join(projDir, fmt.Sprintf("%s.md", fileKey))
	default:
		return filepath.Join(s.baseDir, "unknown.md")
	}
}

func (s *FileStore) Upsert(memoryType MemoryType, fileKey, sectionKey, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.filePath(memoryType, fileKey)
	sections, err := s.readSections(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	sections[sectionKey] = content
	return s.writeSections(filePath, sections)
}

func (s *FileStore) Append(memoryType MemoryType, fileKey, sectionKey, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.filePath(memoryType, fileKey)
	sections, err := s.readSections(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if existing, ok := sections[sectionKey]; ok {
		sections[sectionKey] = existing + "\n" + content
	} else {
		sections[sectionKey] = content
	}

	return s.writeSections(filePath, sections)
}

func (s *FileStore) GetSection(memoryType MemoryType, fileKey, sectionKey string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.filePath(memoryType, fileKey)
	sections, err := s.readSections(filePath)
	if err != nil {
		return "", ErrMemoryNotFound
	}

	content, ok := sections[sectionKey]
	if !ok {
		return "", ErrMemoryNotFound
	}
	return content, nil
}

func (s *FileStore) DeleteSection(memoryType MemoryType, fileKey, sectionKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := s.filePath(memoryType, fileKey)
	sections, err := s.readSections(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrMemoryNotFound
		}
		return err
	}

	if _, ok := sections[sectionKey]; !ok {
		return ErrMemoryNotFound
	}

	delete(sections, sectionKey)

	if len(sections) == 0 {
		return os.Remove(filePath)
	}

	return s.writeSections(filePath, sections)
}

func (s *FileStore) ListUserSections(fileKey string) ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.filePath(TypeUser, fileKey)
	sections, err := s.readSections(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []Memory
	for key, content := range sections {
		result = append(result, Memory{
			ID:      generateMemoryID(key),
			Type:    TypeUser,
			Key:     key,
			Content: content,
		})
	}
	return result, nil
}

func (s *FileStore) ListProjectSections(fileKey string) ([]Memory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filePath := s.filePath(TypeProject, fileKey)
	sections, err := s.readSections(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []Memory
	for key, content := range sections {
		result = append(result, Memory{
			ID:      generateMemoryID(key),
			Type:    TypeProject,
			Key:     key,
			Content: content,
		})
	}
	return result, nil
}

func (s *FileStore) Close() error {
	return nil
}

func (s *FileStore) readSections(filePath string) (map[string]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sections := make(map[string]string)
	var currentKey string
	var currentContent strings.Builder
	hasContent := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if currentKey != "" {
				sections[currentKey] = strings.TrimSpace(currentContent.String())
			}
			currentKey = strings.TrimPrefix(line, "## ")
			currentContent.Reset()
			hasContent = false
		} else {
			if currentKey != "" {
				if hasContent {
					currentContent.WriteString("\n")
				}
				currentContent.WriteString(line)
				hasContent = true
			}
		}
	}

	if currentKey != "" {
		sections[currentKey] = strings.TrimSpace(currentContent.String())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return sections, nil
}

func (s *FileStore) writeSections(filePath string, sections map[string]string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	for key, content := range sections {
		if _, err := fmt.Fprintf(f, "## %s\n%s\n\n", key, content); err != nil {
			return err
		}
	}

	return nil
}

func generateMemoryID(key string) string {
	return "mem-" + sanitizeKey(key)
}

func sanitizeKey(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, s)
}
