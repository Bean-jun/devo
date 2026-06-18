package approval

import (
	"testing"
	"time"
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

func TestDefaultApprovalPolicy(t *testing.T) {
	policy := DefaultApprovalPolicy()

	if policy[OpFileWriteNew] != PolicyAlwaysAsk {
		t.Errorf("expected file_write_new default to always_ask, got %s", policy[OpFileWriteNew])
	}
	if policy[OpExecuteCommand] != PolicyAlwaysAsk {
		t.Errorf("expected execute_command default to always_ask, got %s", policy[OpExecuteCommand])
	}
	if policy[OpMemoryUpdate] != PolicyAutoApprove {
		t.Errorf("expected memory_update default to auto_approve, got %s", policy[OpMemoryUpdate])
	}
	if policy[OpSolidifySkill] != PolicyAutoApprove {
		t.Errorf("expected solidify_skill default to auto_approve, got %s", policy[OpSolidifySkill])
	}
}

func TestIsValidOperationType(t *testing.T) {
	validTypes := []string{"file_write_new", "file_write_overwrite", "file_edit", "execute_command", "memory_update", "solidify_skill"}
	for _, opType := range validTypes {
		if !IsValidOperationType(opType) {
			t.Errorf("expected %s to be valid", opType)
		}
	}

	if IsValidOperationType("invalid_type") {
		t.Error("expected invalid_type to be invalid")
	}
}

func TestIsValidPolicyLevel(t *testing.T) {
	validLevels := []string{"always_ask", "session_trust", "full_trust", "auto_approve"}
	for _, level := range validLevels {
		if !IsValidPolicyLevel(level) {
			t.Errorf("expected %s to be valid", level)
		}
	}

	if IsValidPolicyLevel("invalid_level") {
		t.Error("expected invalid_level to be invalid")
	}
}

func TestIsValidTrustLevel(t *testing.T) {
	validLevels := []string{"low", "normal", "elevated"}
	for _, level := range validLevels {
		if !IsValidTrustLevel(level) {
			t.Errorf("expected %s to be valid", level)
		}
	}

	if IsValidTrustLevel("invalid") {
		t.Error("expected invalid to be invalid")
	}
}

func TestResolveWithSource(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	resolved, ok := mgr.ResolveWithSource(req.ID, StatusRejected, SourceTimeout)
	if !ok {
		t.Fatal("expected resolve to succeed")
	}

	if resolved.Status != StatusRejected {
		t.Errorf("expected StatusRejected, got %s", resolved.Status)
	}
	if resolved.Source != SourceTimeout {
		t.Errorf("expected SourceTimeout, got %s", resolved.Source)
	}
}

func TestSetTimeout(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	timeoutAt := req.CreatedAt.Add(5 * time.Second)
	err := mgr.SetTimeout(req.ID, timeoutAt)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	got, _ := mgr.GetRequest(req.ID)
	if got.TimeoutAt == nil {
		t.Fatal("expected TimeoutAt to be set")
	}
}

func TestSetTimeoutNotFound(t *testing.T) {
	mgr := NewManager()
	err := mgr.SetTimeout("nonexistent", time.Now())
	if err == nil {
		t.Error("expected error for nonexistent request")
	}
}

func TestSetTimeoutOnResolved(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})
	mgr.Resolve(req.ID, StatusApproved)

	err := mgr.SetTimeout(req.ID, time.Now())
	if err == nil {
		t.Error("expected error for resolved request")
	}
}

func TestIsExpired(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	if mgr.IsExpired(req.ID) {
		t.Error("request without timeout should not be expired")
	}

	pastTime := time.Now().Add(-1 * time.Second)
	mgr.SetTimeout(req.ID, pastTime)

	if !mgr.IsExpired(req.ID) {
		t.Error("request with past timeout should be expired")
	}

	futureTime := time.Now().Add(1 * time.Hour)
	mgr.SetTimeout(req.ID, futureTime)

	if mgr.IsExpired(req.ID) {
		t.Error("request with future timeout should not be expired")
	}
}

func TestIsExpiredAfterResolve(t *testing.T) {
	mgr := NewManager()
	req := mgr.CreateRequest("sess-1", "tool-1", OpFileWriteNew, RiskMedium, map[string]any{"path": "test.txt"})

	pastTime := time.Now().Add(-1 * time.Second)
	mgr.SetTimeout(req.ID, pastTime)
	mgr.Resolve(req.ID, StatusApproved)

	if mgr.IsExpired(req.ID) {
		t.Error("resolved request should not be considered expired")
	}
}

func TestResolveEffectivePolicy_SessionPolicy(t *testing.T) {
	mgr := NewManager()

	sessionPolicy := map[OperationType]PolicyLevel{
		OpFileWriteNew: PolicySessionTrust,
	}

	policy := mgr.ResolveEffectivePolicy(sessionPolicy, OpFileWriteNew)
	if policy != PolicySessionTrust {
		t.Errorf("expected session_trust, got %s", policy)
	}
}

func TestResolveEffectivePolicy_DefaultFallback(t *testing.T) {
	mgr := NewManager()

	policy := mgr.ResolveEffectivePolicy(nil, OpExecuteCommand)
	if policy != PolicyAlwaysAsk {
		t.Errorf("expected default always_ask for execute_command, got %s", policy)
	}

	policy = mgr.ResolveEffectivePolicy(nil, OpMemoryUpdate)
	if policy != PolicyAutoApprove {
		t.Errorf("expected default auto_approve for memory_update, got %s", policy)
	}
}

func TestResolveEffectivePolicy_UnknownType(t *testing.T) {
	mgr := NewManager()

	policy := mgr.ResolveEffectivePolicy(nil, "unknown_type")
	if policy != PolicyAlwaysAsk {
		t.Errorf("expected always_ask for unknown type, got %s", policy)
	}
}

func TestIsAutoApproved(t *testing.T) {
	mgr := NewManager()

	if mgr.IsAutoApproved(PolicyAlwaysAsk) {
		t.Error("always_ask should not be auto-approved")
	}
	if !mgr.IsAutoApproved(PolicySessionTrust) {
		t.Error("session_trust should be auto-approved")
	}
	if !mgr.IsAutoApproved(PolicyFullTrust) {
		t.Error("full_trust should be auto-approved")
	}
	if !mgr.IsAutoApproved(PolicyAutoApprove) {
		t.Error("auto_approve should be auto-approved")
	}
}

func TestResolveEffectivePolicy_FullTrustViaUserStore(t *testing.T) {
	mgr := NewManager()
	mgr.SetUserPolicyStore(&mockUserPolicyStore{
		fullTrust: map[OperationType]bool{
			OpFileWriteNew: true,
		},
	})

	policy := mgr.ResolveEffectivePolicy(nil, OpFileWriteNew)
	if policy != PolicyFullTrust {
		t.Errorf("expected full_trust from user store, got %s", policy)
	}

	policy = mgr.ResolveEffectivePolicy(nil, OpExecuteCommand)
	if policy != PolicyAlwaysAsk {
		t.Errorf("expected always_ask for non-full-trust type, got %s", policy)
	}
}

func TestResolveEffectivePolicy_SessionPolicyOverridesUserStore(t *testing.T) {
	mgr := NewManager()
	mgr.SetUserPolicyStore(&mockUserPolicyStore{
		fullTrust: map[OperationType]bool{
			OpFileWriteNew: true,
		},
	})

	sessionPolicy := map[OperationType]PolicyLevel{
		OpFileWriteNew: PolicyAlwaysAsk,
	}

	policy := mgr.ResolveEffectivePolicy(sessionPolicy, OpFileWriteNew)
	if policy != PolicyAlwaysAsk {
		t.Errorf("session policy should override user store, expected always_ask, got %s", policy)
	}
}

type mockUserPolicyStore struct {
	fullTrust map[OperationType]bool
}

func (m *mockUserPolicyStore) GetFullTrust(operationType OperationType) (bool, error) {
	if m.fullTrust == nil {
		return false, nil
	}
	return m.fullTrust[operationType], nil
}

func (m *mockUserPolicyStore) SetFullTrust(operationType OperationType, enabled bool) error {
	if m.fullTrust == nil {
		m.fullTrust = make(map[OperationType]bool)
	}
	m.fullTrust[operationType] = enabled
	return nil
}

func (m *mockUserPolicyStore) GetAllFullTrust() (map[OperationType]bool, error) {
	return m.fullTrust, nil
}
