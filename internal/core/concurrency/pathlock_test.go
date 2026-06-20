package concurrency

import (
	"sync"
	"testing"
	"time"
)

func TestPathLockBasic(t *testing.T) {
	plm := NewPathLockManager()

	plm.Lock("/tmp/test")
	plm.Unlock("/tmp/test")

	plm.Lock("/tmp/test")
	plm.Unlock("/tmp/test")
}

func TestPathLockSerialization(t *testing.T) {
	plm := NewPathLockManager()
	var order []int
	var mu sync.Mutex

	plm.Lock("/tmp/test")

	done := make(chan struct{})

	go func() {
		plm.Lock("/tmp/test")
		mu.Lock()
		order = append(order, 2)
		mu.Unlock()
		plm.Unlock("/tmp/test")
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	order = append(order, 1)
	mu.Unlock()
	plm.Unlock("/tmp/test")

	<-done

	if len(order) != 2 {
		t.Fatalf("expected 2 operations, got %d", len(order))
	}
	if order[0] != 1 || order[1] != 2 {
		t.Errorf("expected serial order [1, 2], got %v", order)
	}
}

func TestPathLockDifferentPathsConcurrent(t *testing.T) {
	plm := NewPathLockManager()

	plm.Lock("/tmp/file1")

	done := make(chan struct{})
	go func() {
		plm.Lock("/tmp/file2")
		plm.Unlock("/tmp/file2")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("different paths should not block each other")
	}

	plm.Unlock("/tmp/file1")
}

func TestPathLockReentrantBlocks(t *testing.T) {
	plm := NewPathLockManager()

	plm.Lock("/tmp/test")

	acquired := make(chan struct{})
	go func() {
		plm.Lock("/tmp/test")
		close(acquired)
		plm.Unlock("/tmp/test")
	}()

	select {
	case <-acquired:
		t.Fatal("reentrant lock on same path should block")
	case <-time.After(100 * time.Millisecond):
	}

	plm.Unlock("/tmp/test")

	select {
	case <-acquired:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("second lock should acquire after first unlock")
	}
}

func TestPathLockTryLock(t *testing.T) {
	plm := NewPathLockManager()

	if !plm.TryLock("/tmp/test") {
		t.Fatal("TryLock should succeed on free path")
	}

	if plm.TryLock("/tmp/test") {
		t.Fatal("TryLock should fail on locked path")
	}

	plm.Unlock("/tmp/test")

	if !plm.TryLock("/tmp/test") {
		t.Fatal("TryLock should succeed after unlock")
	}
	plm.Unlock("/tmp/test")
}

func TestPathLockNormalizePath(t *testing.T) {
	plm := NewPathLockManager()

	plm.Lock("/tmp/test/../test/file")

	done := make(chan struct{})
	go func() {
		plm.Lock("/tmp/test/file")
		close(done)
		plm.Unlock("/tmp/test/file")
	}()

	select {
	case <-done:
		t.Fatal("normalized paths should be treated as same path and block")
	case <-time.After(100 * time.Millisecond):
	}

	plm.Unlock("/tmp/test/../test/file")

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("second lock should acquire after first unlock")
	}
}

func TestPathLockConcurrentStress(t *testing.T) {
	plm := NewPathLockManager()
	var counter int
	var wg sync.WaitGroup

	goroutines := 20
	iterations := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				plm.Lock("/tmp/shared")
				val := counter
				time.Sleep(time.Microsecond)
				counter = val + 1
				plm.Unlock("/tmp/shared")
			}
		}()
	}

	wg.Wait()

	expected := goroutines * iterations
	if counter != expected {
		t.Errorf("expected counter %d, got %d - race condition detected", expected, counter)
	}
}

func TestPathLockCleanup(t *testing.T) {
	plm := NewPathLockManager()

	plm.Lock("/tmp/test")
	plm.Unlock("/tmp/test")

	plm.mu.Lock()
	_, exists := plm.locks[plm.normalizeKey("/tmp/test")]
	plm.mu.Unlock()

	if exists {
		t.Error("lock should be cleaned up after ref count reaches zero")
	}
}
