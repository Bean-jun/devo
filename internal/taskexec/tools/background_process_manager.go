package tools

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// OutputForwarder is implemented by the agent loop and injected at startup so
// the manager can push background-process output to the session EventBus
// without depending on core/session. The loop's implementation looks up the
// session's EventBus by sessionID and publishes a "background_output" event.
type OutputForwarder interface {
	ForwardBackgroundOutput(sessionID string, pid int, stream string, data []byte)
}

// bgOutputFlushInterval caps how often a single stream's accumulated output is
// flushed to the forwarder. Long-running dev servers emit high-volume output;
// without throttling the EventBus history (200 events) is overwritten in
// milliseconds and downstream SSE clients are starved.
const bgOutputFlushInterval = 100 * time.Millisecond

// bgMaxLineLen bounds a single line in the output stream. Lines longer than
// this are split by bufio.Scanner (and an ErrTooLong is recorded).
const bgMaxLineLen = 1 * 1024 * 1024

type bgProcess struct {
	pid       int
	cmd       string
	sessionID string
	startedAt time.Time

	cancel context.CancelFunc
	done   chan struct{}

	closeOnce sync.Once
}

type BackgroundProcessManager struct {
	mu        sync.Mutex
	procs     map[int]*bgProcess
	forwarder OutputForwarder
}

func NewBackgroundProcessManager() *BackgroundProcessManager {
	return &BackgroundProcessManager{
		procs: make(map[int]*bgProcess),
	}
}

// SetOutputForwarder injects the forwarder used to publish background output
// events. Must be called before any Register call. May be nil in tests; in
// that case output is dropped.
func (m *BackgroundProcessManager) SetOutputForwarder(f OutputForwarder) {
	m.mu.Lock()
	m.forwarder = f
	m.mu.Unlock()
}

// Register attaches a background process to the manager and starts a goroutine
// that streams stdout/stderr to the forwarder until both pipes reach EOF or
// Stop is called. The manager owns the pipes from this point on - callers must
// not read or close them.
func (m *BackgroundProcessManager) Register(
	pid int,
	cmd string,
	sessionID string,
	stdoutPipe, stderrPipe io.ReadCloser,
) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &bgProcess{
		pid:       pid,
		cmd:       cmd,
		sessionID: sessionID,
		startedAt: time.Now(),
		cancel:    cancel,
		done:      make(chan struct{}),
	}

	m.mu.Lock()
	m.procs[pid] = p
	forwarder := m.forwarder
	m.mu.Unlock()

	go m.streamPipes(p, ctx, stdoutPipe, stderrPipe, forwarder)
}

func (m *BackgroundProcessManager) streamPipes(p *bgProcess, ctx context.Context, stdoutPipe, stderrPipe io.ReadCloser, forwarder OutputForwarder) {
	defer close(p.done)
	defer m.unregister(p.pid)

	onOutput := func(stream string, data []byte) {
		if forwarder == nil {
			return
		}
		forwarder.ForwardBackgroundOutput(p.sessionID, p.pid, stream, data)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		streamPipe(ctx, stdoutPipe, "stdout", onOutput)
	}()
	go func() {
		defer wg.Done()
		streamPipe(ctx, stderrPipe, "stderr", onOutput)
	}()
	wg.Wait()
}

func streamPipe(ctx context.Context, pipe io.ReadCloser, stream string, onOutput func(string, []byte)) {
	defer pipe.Close()

	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 0, 64*1024), bgMaxLineLen)

	// reader accumulates lines into buf under mu. The main loop flushes buf
	// every bgOutputFlushInterval via a ticker; this decouples "data arrived"
	// from "flush now" so a fast-emitting producer can't starve the flush by
	// resetting the timer on every line.
	var mu sync.Mutex
	var buf []byte

	flush := func() {
		mu.Lock()
		if len(buf) == 0 {
			mu.Unlock()
			return
		}
		data := buf
		buf = nil
		mu.Unlock()
		onOutput(stream, data)
	}

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for scanner.Scan() {
			mu.Lock()
			buf = append(buf, scanner.Bytes()...)
			buf = append(buf, '\n')
			mu.Unlock()
		}
	}()

	ticker := time.NewTicker(bgOutputFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-readerDone:
			flush()
			if err := scanner.Err(); err != nil && err != io.EOF {
				onOutput("stderr", []byte(fmt.Sprintf("\n[warning] output stream %s truncated: %v\n", stream, err)))
			}
			return
		case <-ticker.C:
			flush()
		}
	}
}

func (m *BackgroundProcessManager) unregister(pid int) {
	m.mu.Lock()
	delete(m.procs, pid)
	m.mu.Unlock()
}

type BackgroundProcessInfo struct {
	PID       int       `json:"pid"`
	Cmd       string    `json:"cmd"`
	SessionID string    `json:"session_id,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

func (m *BackgroundProcessManager) List(sessionID string) []BackgroundProcessInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]BackgroundProcessInfo, 0, len(m.procs))
	for _, p := range m.procs {
		if sessionID != "" && p.sessionID != sessionID {
			continue
		}
		result = append(result, BackgroundProcessInfo{
			PID:       p.pid,
			Cmd:       p.cmd,
			SessionID: p.sessionID,
			StartedAt: p.startedAt,
		})
	}
	return result
}

// Stop kills the process group rooted at pid, cancels the pipe goroutine, and
// waits for the goroutine to exit. Safe to call multiple times.
func (m *BackgroundProcessManager) Stop(pid int) error {
	m.mu.Lock()
	p, ok := m.procs[pid]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("no registered background process with pid %d", pid)
	}

	var stopErr error
	p.closeOnce.Do(func() {
		// Kill first so the OS closes the pipes' write ends, causing the
		// streamPipe scanner to return EOF and the goroutine to exit.
		if err := killProcess(p.pid); err != nil {
			stopErr = err
		}
		p.cancel()
	})

	// Wait for goroutine to exit (it will unregister itself).
	<-p.done
	return stopErr
}

// StopSession kills every background process belonging to sessionID. Used by
// session delete and crash recovery.
func (m *BackgroundProcessManager) StopSession(sessionID string) []error {
	m.mu.Lock()
	pids := make([]int, 0, len(m.procs))
	for pid, p := range m.procs {
		if p.sessionID == sessionID {
			pids = append(pids, pid)
		}
	}
	m.mu.Unlock()

	var errs []error
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, pid := range pids {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			if err := m.Stop(pid); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(pid)
	}
	wg.Wait()
	return errs
}

// Shutdown kills all background processes regardless of session. Called from
// App.Shutdown so devo exit cleans up children.
func (m *BackgroundProcessManager) Shutdown() {
	m.mu.Lock()
	pids := make([]int, 0, len(m.procs))
	for pid := range m.procs {
		pids = append(pids, pid)
	}
	m.mu.Unlock()

	for _, pid := range pids {
		_ = m.Stop(pid)
	}
}
