package approval

import (
	"fmt"
	"sync"
	"time"

	"devo/internal/core/session"
)

type OperationType string

const (
	OpFileWriteNew       OperationType = "file_write_new"
	OpFileWriteOverwrite OperationType = "file_write_overwrite"
	OpFileEdit           OperationType = "file_edit"
	OpExecPython         OperationType = "exec_python"
	OpMemoryUpdate       OperationType = "memory_update"
	OpSolidifySkill      OperationType = "solidify_skill"
)

type RiskLevel string

const (
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type PolicyLevel string

const (
	PolicyAlwaysAsk    PolicyLevel = "always_ask"
	PolicySessionTrust PolicyLevel = "session_trust"
	PolicyAutoApprove  PolicyLevel = "auto_approve"
)

type TrustLevel string

const (
	TrustLow      TrustLevel = "low"
	TrustNormal   TrustLevel = "normal"
	TrustElevated TrustLevel = "elevated"
)

type ApprovalSource string

const (
	SourceUser    ApprovalSource = "user"
	SourceTimeout ApprovalSource = "timeout"
)

func DefaultApprovalPolicy() map[OperationType]PolicyLevel {
	return map[OperationType]PolicyLevel{
		OpFileWriteNew:       PolicyAlwaysAsk,
		OpFileWriteOverwrite: PolicyAlwaysAsk,
		OpFileEdit:           PolicyAlwaysAsk,
		OpExecPython:         PolicyAlwaysAsk,
		OpMemoryUpdate:       PolicyAutoApprove,
		OpSolidifySkill:      PolicyAutoApprove,
	}
}

func IsValidOperationType(opType string) bool {
	switch OperationType(opType) {
	case OpFileWriteNew, OpFileWriteOverwrite, OpFileEdit, OpExecPython, OpMemoryUpdate, OpSolidifySkill:
		return true
	}
	return false
}

func IsValidPolicyLevel(level string) bool {
	switch PolicyLevel(level) {
	case PolicyAlwaysAsk, PolicySessionTrust, PolicyAutoApprove:
		return true
	}
	return false
}

func IsValidTrustLevel(level string) bool {
	switch TrustLevel(level) {
	case TrustLow, TrustNormal, TrustElevated:
		return true
	}
	return false
}

type ApprovalRequest struct {
	ID            string         `json:"id"`
	SessionID     string         `json:"session_id"`
	ToolCallID    string         `json:"tool_call_id"`
	OperationType OperationType  `json:"operation_type"`
	RiskLevel     RiskLevel      `json:"risk_level"`
	Details       map[string]any `json:"details"`
	Status        Status         `json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	ResolvedAt    *time.Time     `json:"resolved_at,omitempty"`
	TimeoutAt     *time.Time     `json:"timeout_at,omitempty"`
	Source        ApprovalSource `json:"source,omitempty"`
}

type Manager struct {
	mu       sync.RWMutex
	requests map[string]*ApprovalRequest
}

func NewManager() *Manager {
	return &Manager{
		requests: make(map[string]*ApprovalRequest),
	}
}

func (m *Manager) CreateRequest(sessionID, toolCallID string, opType OperationType, riskLevel RiskLevel, details map[string]any) *ApprovalRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	if riskLevel == "" {
		riskLevel = RiskMedium
	}

	req := &ApprovalRequest{
		ID:            session.GenerateID("approval"),
		SessionID:     sessionID,
		ToolCallID:    toolCallID,
		OperationType: opType,
		RiskLevel:     riskLevel,
		Details:       details,
		Status:        StatusPending,
		CreatedAt:     time.Now(),
	}

	m.requests[req.ID] = req
	return req
}

func (m *Manager) GetRequest(id string) (*ApprovalRequest, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, ok := m.requests[id]
	return req, ok
}

func (m *Manager) Resolve(id string, decision Status) (*ApprovalRequest, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return nil, false
	}

	if req.Status != StatusPending {
		return nil, false
	}

	now := time.Now()
	req.Status = decision
	req.ResolvedAt = &now

	return req, true
}

func (m *Manager) ResolveWithSource(id string, decision Status, source ApprovalSource) (*ApprovalRequest, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return nil, false
	}

	if req.Status != StatusPending {
		return nil, false
	}

	now := time.Now()
	req.Status = decision
	req.ResolvedAt = &now
	req.Source = source

	return req, true
}

func (m *Manager) SetTimeout(id string, timeoutAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	req, ok := m.requests[id]
	if !ok {
		return fmt.Errorf("approval request not found: %s", id)
	}

	if req.Status != StatusPending {
		return fmt.Errorf("approval request %s is not pending", id)
	}

	req.TimeoutAt = &timeoutAt
	return nil
}

func (m *Manager) IsExpired(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	req, ok := m.requests[id]
	if !ok {
		return false
	}

	if req.Status != StatusPending {
		return false
	}

	if req.TimeoutAt == nil {
		return false
	}

	return time.Now().After(*req.TimeoutAt)
}

func (m *Manager) GetPendingRequest(sessionID string) *ApprovalRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, req := range m.requests {
		if req.SessionID == sessionID && req.Status == StatusPending {
			return req
		}
	}
	return nil
}

func (m *Manager) ResolveEffectivePolicy(sessionPolicy map[OperationType]PolicyLevel, cfgPolicy map[OperationType]PolicyLevel, operationType OperationType) PolicyLevel {
	if sessionPolicy != nil {
		if policy, ok := sessionPolicy[operationType]; ok {
			return policy
		}
	}

	if cfgPolicy != nil {
		if policy, ok := cfgPolicy[operationType]; ok {
			return policy
		}
	}

	defaults := DefaultApprovalPolicy()
	if policy, ok := defaults[operationType]; ok {
		return policy
	}

	return PolicyAlwaysAsk
}

func (m *Manager) IsAutoApproved(policy PolicyLevel) bool {
	return policy == PolicySessionTrust || policy == PolicyAutoApprove
}
