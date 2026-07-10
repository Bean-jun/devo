package tools

import (
	"fmt"
	"sync"
	"time"
)

const gracefulShutdownTimeout = 5 * time.Second

const livenessPollInterval = 2 * time.Second

type bgProcess struct {
	pid       int
	pgid      int
	cmd       string
	sessionID string
	logPath   string
	startedAt time.Time

	handle procHandle

	mu     sync.Mutex
	exited bool
}

type procHandle interface {
	isAlive() (bool, error)
	terminateGraceful() error
	killForce() error
}

type BackgroundProcessManager struct {
	mu    sync.Mutex
	procs map[int]*bgProcess

	stopMonitor chan struct{}
	monitorOnce sync.Once
}

func NewBackgroundProcessManager() *BackgroundProcessManager {
	m := &BackgroundProcessManager{
		procs:       make(map[int]*bgProcess),
		stopMonitor: make(chan struct{}),
	}
	go m.monitorLoop()
	return m
}

func (m *BackgroundProcessManager) Register(pid int, cmd string, sessionID string, logPath string) error {
	handle, err := newProcHandle(pid)
	if err != nil {
		return fmt.Errorf("register background process %d: %w", pid, err)
	}

	proc := &bgProcess{
		pid:       pid,
		pgid:      pid,
		cmd:       cmd,
		sessionID: sessionID,
		logPath:   logPath,
		startedAt: time.Now(),
		handle:    handle,
	}

	m.mu.Lock()
	m.procs[pid] = proc
	m.mu.Unlock()
	return nil
}

func (m *BackgroundProcessManager) unregister(pid int) {
	m.mu.Lock()
	delete(m.procs, pid)
	m.mu.Unlock()
}

func (m *BackgroundProcessManager) monitorLoop() {
	ticker := time.NewTicker(livenessPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopMonitor:
			return
		case <-ticker.C:
			m.reapExited()
		}
	}
}

func (m *BackgroundProcessManager) reapExited() {
	m.mu.Lock()
	snapshot := make([]*bgProcess, 0, len(m.procs))
	for _, p := range m.procs {
		snapshot = append(snapshot, p)
	}
	m.mu.Unlock()

	for _, p := range snapshot {
		alive, err := p.handle.isAlive()
		if err != nil {
			continue
		}
		if !alive {
			m.unregister(p.pid)
		}
	}
}

type BackgroundProcessInfo struct {
	PID       int       `json:"pid"`
	Cmd       string    `json:"cmd"`
	SessionID string    `json:"session_id,omitempty"`
	LogPath   string    `json:"log_path,omitempty"`
	StartedAt time.Time `json:"started_at"`
	Alive     bool      `json:"alive"`
}

func (m *BackgroundProcessManager) List(sessionID string) []BackgroundProcessInfo {
	m.mu.Lock()
	snapshot := make([]*bgProcess, 0, len(m.procs))
	for _, p := range m.procs {
		if sessionID != "" && p.sessionID != sessionID {
			continue
		}
		snapshot = append(snapshot, p)
	}
	m.mu.Unlock()

	result := make([]BackgroundProcessInfo, 0, len(snapshot))
	for _, p := range snapshot {
		alive, _ := p.handle.isAlive()
		result = append(result, BackgroundProcessInfo{
			PID:       p.pid,
			Cmd:       p.cmd,
			SessionID: p.sessionID,
			LogPath:   p.logPath,
			StartedAt: p.startedAt,
			Alive:     alive,
		})
	}
	return result
}

func (m *BackgroundProcessManager) Stop(pid int) error {
	m.mu.Lock()
	p, ok := m.procs[pid]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("no registered background process with pid %d (it may have already exited)", pid)
	}

	alive, err := p.handle.isAlive()
	if err != nil {
		return fmt.Errorf("check process %d liveness: %w", pid, err)
	}
	if !alive {
		m.unregister(pid)
		return fmt.Errorf("process %d has already exited", pid)
	}

	if err := p.handle.terminateGraceful(); err != nil {
		if killErr := p.handle.killForce(); killErr != nil {
			return fmt.Errorf("graceful terminate failed (%v) and force kill also failed: %w", err, killErr)
		}
		m.unregister(pid)
		return nil
	}

	deadline := time.After(gracefulShutdownTimeout)
	pollTicker := time.NewTicker(200 * time.Millisecond)
	defer pollTicker.Stop()

	for {
		select {
		case <-deadline:
			if err := p.handle.killForce(); err != nil {
				return fmt.Errorf("process %d did not exit gracefully and force kill failed: %w", pid, err)
			}
			m.unregister(pid)
			return nil
		case <-pollTicker.C:
			stillAlive, _ := p.handle.isAlive()
			if !stillAlive {
				m.unregister(pid)
				return nil
			}
		}
	}
}

func (m *BackgroundProcessManager) ShutdownAll(sessionID string) []error {
	m.mu.Lock()
	pids := make([]int, 0, len(m.procs))
	for pid, p := range m.procs {
		if sessionID != "" && p.sessionID != sessionID {
			continue
		}
		pids = append(pids, pid)
	}
	m.mu.Unlock()

	var errs []error
	var wg sync.WaitGroup
	var errMu sync.Mutex

	for _, pid := range pids {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			if err := m.Stop(pid); err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
			}
		}(pid)
	}
	wg.Wait()
	return errs
}

func (m *BackgroundProcessManager) Close() {
	m.monitorOnce.Do(func() {
		close(m.stopMonitor)
	})
}
