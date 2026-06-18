package approval

import (
	"testing"
)

func TestCreateRequest(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	if req.ID == "" {
		t.Error("expected non-empty ID")
	}
	if req.SessionID != "sess-1" {
		t.Errorf("expected SessionID sess-1, got %s", req.SessionID)
	}
	if req.ToolCallID != "tool-1" {
		t.Errorf("expected ToolCallID tool-1, got %s", req.ToolCallID)
	}
	if req.OperationType != OpFileWriteNew {
		t.Errorf("expected OpFileWriteNew, got %s", req.OperationType)
	}
	if req.Status != StatusPending {
		t.Errorf("expected StatusPending, got %s", req.Status)
	}
	if req.Details["path"] != "test.txt" {
		t.Errorf("expected details path=test.txt, got %v", req.Details["path"])
	}
	if req.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestGetRequest(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	got, ok := mgr.GetRequest(req.ID)
	if !ok {
		t.Fatalf("request not found: %s", req.ID)
	}

	if got.ID != req.ID {
		t.Errorf("expected ID %s, got %s", req.ID, got.ID)
	}
}

func TestGetRequestNotFound(t *testing.T) {
	mgr := NewManager()
	_, ok := mgr.GetRequest("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent request")
	}
}

func TestResolveApproved(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	resolved, ok := mgr.Resolve(req.ID, StatusApproved)
	if !ok {
		t.Fatal("expected resolve to succeed")
	}

	if resolved.Status != StatusApproved {
		t.Errorf("expected StatusApproved, got %s", resolved.Status)
	}
	if resolved.ResolvedAt == nil {
		t.Error("expected ResolvedAt to be set")
	}
	if resolved.ResolvedAt.IsZero() {
		t.Error("expected ResolvedAt to be non-zero")
	}
}

func TestResolveRejected(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	resolved, ok := mgr.Resolve(req.ID, StatusRejected)
	if !ok {
		t.Fatal("expected resolve to succeed")
	}

	if resolved.Status != StatusRejected {
		t.Errorf("expected StatusRejected, got %s", resolved.Status)
	}
}

func TestResolveAlreadyResolved(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	_, ok := mgr.Resolve(req.ID, StatusApproved)
	if !ok {
		t.Fatal("first resolve should succeed")
	}

	_, ok = mgr.Resolve(req.ID, StatusRejected)
	if ok {
		t.Fatal("second resolve should fail on already resolved request")
	}
}

func TestGetPendingRequest(t *testing.T) {
	mgr := NewManager()

	mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})
	mgr.CreateRequest("sess-2", "tool-2", OpFileWriteOverwrite, RiskMedium, map[string]any{"path": "test2.txt"})

	pending := mgr.GetPendingRequest("sess-1")
	if pending == nil {
		t.Fatal("expected pending request for sess-1")
	}
	if pending.SessionID != "sess-1" {
		t.Errorf("expected SessionID sess-1, got %s", pending.SessionID)
	}

	pending = mgr.GetPendingRequest("sess-2")
	if pending == nil {
		t.Fatal("expected pending request for sess-2")
	}

	pending = mgr.GetPendingRequest("nonexistent")
	if pending != nil {
		t.Errorf("expected nil for nonexistent session, got %v", pending)
	}
}

func TestGetPendingAfterResolve(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	mgr.Resolve(req.ID, StatusApproved)

	pending := mgr.GetPendingRequest("sess-1")
	if pending != nil {
		t.Error("expected no pending request after resolution")
	}
}
