package concurrency

import (
	"fmt"
	"path/filepath"
	"sync"
)

type PathLockManager struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
	refs  map[string]int
}

func NewPathLockManager() *PathLockManager {
	return &PathLockManager{
		locks: make(map[string]*sync.Mutex),
		refs:  make(map[string]int),
	}
}

func (m *PathLockManager) Lock(path string) {
	key := m.normalizeKey(path)

	m.mu.Lock()
	mu, exists := m.locks[key]
	if !exists {
		mu = &sync.Mutex{}
		m.locks[key] = mu
	}
	m.refs[key]++
	m.mu.Unlock()

	mu.Lock()
}

func (m *PathLockManager) Unlock(path string) {
	key := m.normalizeKey(path)

	m.mu.Lock()
	mu, exists := m.locks[key]
	if !exists {
		m.mu.Unlock()
		panic(fmt.Sprintf("pathlock: unlock of unlocked path: %s", path))
	}
	m.refs[key]--
	if m.refs[key] <= 0 {
		delete(m.locks, key)
		delete(m.refs, key)
	}
	m.mu.Unlock()

	mu.Unlock()
}

func (m *PathLockManager) TryLock(path string) bool {
	key := m.normalizeKey(path)

	m.mu.Lock()
	mu, exists := m.locks[key]
	if !exists {
		mu = &sync.Mutex{}
		m.locks[key] = mu
	}
	m.refs[key]++
	m.mu.Unlock()

	return mu.TryLock()
}

func (m *PathLockManager) normalizeKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(abs)
}
