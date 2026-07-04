package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
)

type Manager struct {
	store           *FileStore
	pathLockManager *concurrency.PathLockManager
	approvalManager *approval.Manager
	mu              sync.Mutex
	pendingDrafts   map[string][]MemoryDraft
}

func NewManager(store *FileStore, pathLockManager *concurrency.PathLockManager, approvalManager *approval.Manager) *Manager {
	return &Manager{
		store:           store,
		pathLockManager: pathLockManager,
		approvalManager: approvalManager,
		pendingDrafts:   make(map[string][]MemoryDraft),
	}
}

func (m *Manager) ProjectKey(workingDir string) string {
	abs, err := filepath.Abs(workingDir)
	if err != nil {
		abs = workingDir
	}
	hash := sha256.Sum256([]byte(filepath.Clean(abs)))
	return hex.EncodeToString(hash[:])[:16]
}

func (m *Manager) UserKey() string {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "default"
	}
	return username
}

func (m *Manager) Upsert(typ MemoryType, workingDir, key, content string, source MemorySource) (*Memory, error) {
	var fileKey string
	switch typ {
	case TypeUser:
		fileKey = m.UserKey()
	case TypeProject:
		fileKey = m.ProjectKey(workingDir)
	}

	lockPath := m.lockPath(typ, fileKey)
	m.pathLockManager.Lock(lockPath)
	defer m.pathLockManager.Unlock(lockPath)

	if err := m.store.Upsert(typ, fileKey, key, content); err != nil {
		return nil, fmt.Errorf("upsert memory: %w", err)
	}

	return &Memory{
		ID:      generateMemoryID(key),
		Type:    typ,
		Key:     key,
		Content: content,
		Source:  source,
	}, nil
}

func (m *Manager) Append(typ MemoryType, workingDir, key, content string, source MemorySource) (*Memory, error) {
	var fileKey string
	switch typ {
	case TypeUser:
		fileKey = m.UserKey()
	case TypeProject:
		fileKey = m.ProjectKey(workingDir)
	}

	lockPath := m.lockPath(typ, fileKey)
	m.pathLockManager.Lock(lockPath)
	defer m.pathLockManager.Unlock(lockPath)

	if err := m.store.Append(typ, fileKey, key, content); err != nil {
		return nil, fmt.Errorf("append memory: %w", err)
	}

	combinedContent, err := m.store.GetSection(typ, fileKey, key)
	if err != nil {
		combinedContent = content
	}

	return &Memory{
		ID:      generateMemoryID(key),
		Type:    typ,
		Key:     key,
		Content: combinedContent,
		Source:  source,
	}, nil
}

func (m *Manager) Get(id string) (*Memory, error) {
	userKey := m.UserKey()
	sections, _ := m.store.ListUserSections(userKey)
	for _, mem := range sections {
		if mem.ID == id {
			return &mem, nil
		}
	}

	entries, err := os.ReadDir(filepath.Join(m.store.baseDir, "projects"))
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fileKey := strings.TrimSuffix(entry.Name(), ".md")
			sections, _ := m.store.ListProjectSections(fileKey)
			for _, mem := range sections {
				if mem.ID == id {
					return &mem, nil
				}
			}
		}
	}

	return nil, ErrMemoryNotFound
}

func (m *Manager) Delete(memoryType MemoryType, workingDir, sectionKey string) error {
	var fileKey string
	switch memoryType {
	case TypeUser:
		fileKey = m.UserKey()
	case TypeProject:
		fileKey = m.ProjectKey(workingDir)
	}

	lockPath := m.lockPath(memoryType, fileKey)
	m.pathLockManager.Lock(lockPath)
	defer m.pathLockManager.Unlock(lockPath)

	return m.store.DeleteSection(memoryType, fileKey, sectionKey)
}

func (m *Manager) List(memoryType MemoryType, workingDir string) ([]Memory, error) {
	switch memoryType {
	case TypeUser:
		return m.store.ListUserSections(m.UserKey())
	case TypeProject:
		return m.store.ListProjectSections(m.ProjectKey(workingDir))
	default:
		return nil, fmt.Errorf("unknown memory type: %s", memoryType)
	}
}

func (m *Manager) GetRelevantMemories(workingDir, sessionID string) string {
	userKey := m.UserKey()
	projectKey := m.ProjectKey(workingDir)

	userMemories, _ := m.store.ListUserSections(userKey)
	projectMemories, _ := m.store.ListProjectSections(projectKey)

	if len(userMemories) == 0 && len(projectMemories) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "## 长期记忆")

	if len(userMemories) > 0 {
		parts = append(parts, "### 用户偏好")
		for _, mem := range userMemories {
			parts = append(parts, fmt.Sprintf("- **%s**: %s", mem.Key, mem.Content))
		}
	}

	if len(projectMemories) > 0 {
		parts = append(parts, "### 项目记忆")
		for _, mem := range projectMemories {
			parts = append(parts, fmt.Sprintf("- **%s**: %s", mem.Key, mem.Content))
		}
	}

	return strings.Join(parts, "\n")
}

func (m *Manager) DetectAutoUpdate(userMessage string, sessionID string) []MemoryDraft {
	lower := strings.ToLower(userMessage)

	if strings.Contains(lower, "记住这个") || strings.Contains(lower, "记住一下") ||
		strings.Contains(lower, "记住") || strings.Contains(lower, "记下来") {
		key := m.extractKey(userMessage)
		if key == "" {
			return nil
		}
		return []MemoryDraft{{
			Type:             TypeUser,
			Key:              key,
			SuggestedContent: userMessage,
		}}
	}

	return nil
}

func (m *Manager) AddPendingDrafts(sessionID string, drafts []MemoryDraft) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendingDrafts[sessionID] = append(m.pendingDrafts[sessionID], drafts...)
}

func (m *Manager) FlushPendingDrafts(sessionID string) []MemoryDraft {
	m.mu.Lock()
	defer m.mu.Unlock()
	drafts := m.pendingDrafts[sessionID]
	delete(m.pendingDrafts, sessionID)
	return drafts
}

func (m *Manager) HasPendingDrafts(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.pendingDrafts[sessionID]) > 0
}

func (m *Manager) ShouldAutoApprove(sessionPolicy map[string]string) bool {
	policyLevel := approval.PolicyLevel(sessionPolicy[string(approval.OpMemoryUpdate)])
	if policyLevel == "" {
		policyLevel = approval.PolicyAutoApprove
	}
	return m.approvalManager.IsAutoApproved(policyLevel)
}

func (m *Manager) CommitDrafts(sessionID string, drafts []MemoryDraft) error {
	for _, draft := range drafts {
		if _, err := m.Upsert(draft.Type, draft.WorkingDir, draft.Key, draft.SuggestedContent, SourceAuto); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) extractKey(userMessage string) string {
	lower := strings.ToLower(userMessage)
	triggers := []string{"记住这个", "记住一下", "记住", "记下来"}
	for _, t := range triggers {
		idx := strings.Index(lower, t)
		if idx >= 0 {
			rest := strings.TrimSpace(userMessage[idx+len(t):])
			rest = strings.Trim(rest, "，。,.!！ ")
			if len(rest) > 50 {
				rest = rest[:50]
			}
			if rest == "" {
				rest = "用户偏好"
			}
			return rest
		}
	}
	return ""
}

func (m *Manager) lockPath(typ MemoryType, fileKey string) string {
	return filepath.Join("memory", string(typ), fileKey)
}

func (m *Manager) SortByUpdated(memories []Memory) {
	sort.Slice(memories, func(i, j int) bool {
		return memories[i].UpdatedAt.After(memories[j].UpdatedAt)
	})
}
