package sqlite

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"devo/internal/core/session"
)

type SessionModel struct {
	ID                        string    `gorm:"primaryKey;size:64"`
	Title                     string    `gorm:"size:256"`
	WorkingDirectory          string    `gorm:"size:512;index:idx_working_dir"`
	AgentID                   string    `gorm:"size:64;default:devo-default"`
	State                     string    `gorm:"size:32;index:idx_state"`
	CreatedAt                 time.Time `gorm:"autoCreateTime"`
	LastActiveAt              time.Time
	ActiveSSEConnections      int
	TrustLevel                string `gorm:"size:32;default:normal"`
	ApprovalPolicyJSON        string `gorm:"type:text"`
	ApprovalTimeoutSeconds    int    `gorm:"default:300"`
	ToolCallCount             int    `gorm:"default:0"`
	MessageCount              int    `gorm:"default:0"`
	LastLoopTerminationReason string `gorm:"size:32"`
	TokenUsageInput           int    `gorm:"default:0"`
	TokenUsageOutput          int    `gorm:"default:0"`
	TokenUsageTotal           int    `gorm:"default:0"`
	CompressionCount          int    `gorm:"default:0"`
	MaxConcurrentToolCalls    int    `gorm:"default:0"`
	MaxConcurrentSubprocesses int    `gorm:"default:0"`
	CurrentContextTokens      int    `gorm:"default:0"`
	ActiveSkillsJSON          string `gorm:"type:text"`
}

type MessageModel struct {
	ID               string `gorm:"primaryKey;size:64"`
	SessionID        string `gorm:"size:64;index:idx_msg_session_id;not null"`
	Role             string `gorm:"size:16"`
	Content          string `gorm:"type:text"`
	ContentPartsJSON string `gorm:"type:text"`
	Reasoning        string `gorm:"type:text"`
	ToolCallsJSON    string `gorm:"type:text"`
	ToolCallID       string `gorm:"size:64"`
	Seq              int    `gorm:"index:idx_msg_seq"`
	CreatedAt        time.Time
}

type EventModel struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	SessionID string `gorm:"size:64;index:idx_event_session_id;not null"`
	EventID   int64  `gorm:"index:idx_event_id"`
	EventType string `gorm:"size:64"`
	DataJSON  string `gorm:"type:text"`
	CreatedAt time.Time
}

func (m *SessionModel) ToDomain() *session.Session {
	sess := &session.Session{
		ID:                        m.ID,
		Title:                     m.Title,
		WorkingDirectory:          m.WorkingDirectory,
		AgentID:                   m.AgentID,
		State:                     session.State(m.State),
		CreatedAt:                 m.CreatedAt,
		LastActiveAt:              m.LastActiveAt,
		ActiveSSEConnections:      m.ActiveSSEConnections,
		TrustLevel:                m.TrustLevel,
		ApprovalPolicy:            make(map[string]string),
		ApprovalTimeoutSeconds:    m.ApprovalTimeoutSeconds,
		ToolCallCount:             m.ToolCallCount,
		MessageCount:              m.MessageCount,
		LastLoopTerminationReason: session.LoopTerminationReason(m.LastLoopTerminationReason),
		TokenUsage: session.TokenUsage{
			Input:  m.TokenUsageInput,
			Output: m.TokenUsageOutput,
			Total:  m.TokenUsageTotal,
		},
		CompressionCount:          m.CompressionCount,
		MaxConcurrentToolCalls:    m.MaxConcurrentToolCalls,
		MaxConcurrentSubprocesses: m.MaxConcurrentSubprocesses,
		CurrentContextTokens:      m.CurrentContextTokens,
	}

	if m.ApprovalPolicyJSON != "" {
		var policy map[string]string
		if err := json.Unmarshal([]byte(m.ApprovalPolicyJSON), &policy); err == nil {
			sess.ApprovalPolicy = policy
		}
	}

	if m.ActiveSkillsJSON != "" {
		var skills []string
		if err := json.Unmarshal([]byte(m.ActiveSkillsJSON), &skills); err == nil {
			sess.ActiveSkills = skills
		}
	}

	return sess
}

func fromDomain(s *session.Session) *SessionModel {
	model := &SessionModel{
		ID:                        s.ID,
		Title:                     s.Title,
		WorkingDirectory:          s.WorkingDirectory,
		AgentID:                   s.AgentID,
		State:                     string(s.State),
		CreatedAt:                 s.CreatedAt,
		LastActiveAt:              s.LastActiveAt,
		ActiveSSEConnections:      s.ActiveSSEConnections,
		TrustLevel:                s.TrustLevel,
		ApprovalTimeoutSeconds:    s.ApprovalTimeoutSeconds,
		ToolCallCount:             s.ToolCallCount,
		LastLoopTerminationReason: string(s.LastLoopTerminationReason),
		TokenUsageInput:           s.TokenUsage.Input,
		TokenUsageOutput:          s.TokenUsage.Output,
		TokenUsageTotal:           s.TokenUsage.Total,
		CompressionCount:          s.CompressionCount,
		MaxConcurrentToolCalls:    s.MaxConcurrentToolCalls,
		MaxConcurrentSubprocesses: s.MaxConcurrentSubprocesses,
		CurrentContextTokens:      s.CurrentContextTokens,
	}

	if s.ApprovalPolicy != nil && len(s.ApprovalPolicy) > 0 {
		data, err := json.Marshal(s.ApprovalPolicy)
		if err == nil {
			model.ApprovalPolicyJSON = string(data)
		}
	}

	if len(s.ActiveSkills) > 0 {
		data, err := json.Marshal(s.ActiveSkills)
		if err == nil {
			model.ActiveSkillsJSON = string(data)
		}
	}

	return model
}

func (m *MessageModel) ToDomain() session.Message {
	msg := session.Message{
		ID:         m.ID,
		Role:       session.Role(m.Role),
		Content:    m.Content,
		Reasoning:  m.Reasoning,
		ToolCallID: m.ToolCallID,
		CreatedAt:  m.CreatedAt,
	}

	if m.ToolCallsJSON != "" {
		var toolCalls []session.ToolCall
		if err := json.Unmarshal([]byte(m.ToolCallsJSON), &toolCalls); err == nil {
			msg.ToolCalls = toolCalls
		}
	}

	if m.ContentPartsJSON != "" {
		var contentParts []session.ContentPart
		if err := json.Unmarshal([]byte(m.ContentPartsJSON), &contentParts); err == nil {
			msg.ContentParts = contentParts
		}
	}

	return msg
}

func fromMessage(sessionID string, seq int, m session.Message) *MessageModel {
	model := &MessageModel{
		ID:         m.ID,
		SessionID:  sessionID,
		Role:       string(m.Role),
		Content:    m.Content,
		Reasoning:  m.Reasoning,
		ToolCallID: m.ToolCallID,
		Seq:        seq,
		CreatedAt:  m.CreatedAt,
	}

	if len(m.ToolCalls) > 0 {
		data, err := json.Marshal(m.ToolCalls)
		if err == nil {
			model.ToolCallsJSON = string(data)
		}
	}

	if len(m.ContentParts) > 0 {
		data, err := json.Marshal(m.ContentParts)
		if err == nil {
			model.ContentPartsJSON = string(data)
		}
	}

	return model
}

func (m *EventModel) ToDomain() (session.Event, error) {
	var data any
	if m.DataJSON != "" {
		if err := json.Unmarshal([]byte(m.DataJSON), &data); err != nil {
			return session.Event{}, err
		}
	}
	return session.Event{
		ID:        m.EventID,
		Type:      m.EventType,
		Data:      data,
		CreatedAt: m.CreatedAt,
	}, nil
}

func fromEvent(sessionID string, e session.Event) (*EventModel, error) {
	dataJSON, err := json.Marshal(e.Data)
	if err != nil {
		return nil, err
	}
	return &EventModel{
		SessionID: sessionID,
		EventID:   e.ID,
		EventType: e.Type,
		DataJSON:  string(dataJSON),
		CreatedAt: e.CreatedAt,
	}, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&SessionModel{}, &MessageModel{}, &EventModel{}, &UserConfigModel{}, &TokenUsageStepModel{}, &FileModificationLogModel{})
}

type TokenUsageStepModel struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"`
	SessionID    string    `gorm:"size:64;index:idx_usage_step_session;not null"`
	StepSeq      int       `gorm:"index:idx_usage_step_seq"`
	InputTokens  int       `gorm:"default:0"`
	OutputTokens int       `gorm:"default:0"`
	Source       string    `gorm:"size:16"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

type UserConfigModel struct {
	Key   string `gorm:"primaryKey;size:128"`
	Value string `gorm:"type:text"`
}

type FileModificationLogModel struct {
	ID                uint      `gorm:"primaryKey;autoIncrement"`
	SessionID         string    `gorm:"size:64;index:idx_fml_session;not null"`
	FilePath          string    `gorm:"size:512"`
	ModifiedAt        time.Time `gorm:"index:idx_fml_time"`
	CausedByMessageID string    `gorm:"size:64"`
}
