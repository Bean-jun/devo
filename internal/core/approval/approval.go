package approval

import (
	"sync"
	"time"

	"devo/internal/core/session"
)

type OperationType string

const (
	OpFileWriteNew       OperationType = "file_write_new"
	OpFileWriteOverwrite OperationType = "file_write_overwrite"
	OpFileEdit           OperationType = "file_edit"
)

type RiskLevel string

const (
	RiskMedium RiskLevel = "medium"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

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

func (m *Manager) CreateRequest(sessionID, toolCallID string, opType OperationType, details map[string]any) *ApprovalRequest {
	m.mu.Lock()
	defer m.mu.Unlock()

	req := &ApprovalRequest{
		ID:            session.GenerateID("approval"),
		SessionID:     sessionID,
		ToolCallID:    toolCallID,
		OperationType: opType,
		RiskLevel:     RiskMedium,
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
