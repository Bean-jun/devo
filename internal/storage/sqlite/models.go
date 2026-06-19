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
	State                     string    `gorm:"size:32;index:idx_state"`
	CreatedAt                 time.Time `gorm:"autoCreateTime"`
	LastActiveAt              time.Time
	ActiveSSEConnections      int
	TrustLevel                string `gorm:"size:32;default:normal"`
	ApprovalPolicyJSON        string `gorm:"type:text"`
	ApprovalTimeoutSeconds    int    `gorm:"default:300"`
	ToolCallLimit             int    `gorm:"default:50"`
	ToolCallCount             int    `gorm:"default:0"`
	LastLoopTerminationReason string `gorm:"size:32"`
	TokenUsageInput           int    `gorm:"default:0"`
	TokenUsageOutput          int    `gorm:"default:0"`
	TokenUsageTotal           int    `gorm:"default:0"`
	CompressionStateJSON      string `gorm:"type:text"`
	CompressionCount          int    `gorm:"default:0"`
	CompressThreshold         int    `gorm:"default:0"`
	KeepRecent                int    `gorm:"default:0"`
}

type MessageModel struct {
	ID            string `gorm:"primaryKey;size:64"`
	SessionID     string `gorm:"size:64;index:idx_msg_session_id;not null"`
	Role          string `gorm:"size:16"`
	Content       string `gorm:"type:text"`
	ToolCallsJSON string `gorm:"type:text"`
	ToolCallID    string `gorm:"size:64"`
	Seq           int    `gorm:"index:idx_msg_seq"`
	CreatedAt     time.Time
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
		State:                     session.State(m.State),
		CreatedAt:                 m.CreatedAt,
		LastActiveAt:              m.LastActiveAt,
		ActiveSSEConnections:      m.ActiveSSEConnections,
		TrustLevel:                m.TrustLevel,
		ApprovalPolicy:            make(map[string]string),
		ApprovalTimeoutSeconds:    m.ApprovalTimeoutSeconds,
		ToolCallLimit:             m.ToolCallLimit,
		ToolCallCount:             m.ToolCallCount,
		LastLoopTerminationReason: session.LoopTerminationReason(m.LastLoopTerminationReason),
		TokenUsage: session.TokenUsage{
			Input:  m.TokenUsageInput,
			Output: m.TokenUsageOutput,
			Total:  m.TokenUsageTotal,
		},
		CompressionCount:  m.CompressionCount,
		CompressThreshold: m.CompressThreshold,
		KeepRecent:        m.KeepRecent,
	}

	if m.ApprovalPolicyJSON != "" {
		var policy map[string]string
		if err := json.Unmarshal([]byte(m.ApprovalPolicyJSON), &policy); err == nil {
			sess.ApprovalPolicy = policy
		}
	}

	if m.CompressionStateJSON != "" {
		var state session.CompressionState
		if err := json.Unmarshal([]byte(m.CompressionStateJSON), &state); err == nil {
			sess.CompressionState = &state
		}
	}

	return sess
}

func fromDomain(s *session.Session) *SessionModel {
	model := &SessionModel{
		ID:                        s.ID,
		Title:                     s.Title,
		WorkingDirectory:          s.WorkingDirectory,
		State:                     string(s.State),
		CreatedAt:                 s.CreatedAt,
		LastActiveAt:              s.LastActiveAt,
		ActiveSSEConnections:      s.ActiveSSEConnections,
		TrustLevel:                s.TrustLevel,
		ApprovalTimeoutSeconds:    s.ApprovalTimeoutSeconds,
		ToolCallLimit:             s.ToolCallLimit,
		ToolCallCount:             s.ToolCallCount,
		LastLoopTerminationReason: string(s.LastLoopTerminationReason),
		TokenUsageInput:           s.TokenUsage.Input,
		TokenUsageOutput:          s.TokenUsage.Output,
		TokenUsageTotal:           s.TokenUsage.Total,
		CompressionCount:          s.CompressionCount,
		CompressThreshold:         s.CompressThreshold,
		KeepRecent:                s.KeepRecent,
	}

	if s.ApprovalPolicy != nil && len(s.ApprovalPolicy) > 0 {
		data, err := json.Marshal(s.ApprovalPolicy)
		if err == nil {
			model.ApprovalPolicyJSON = string(data)
		}
	}

	if s.CompressionState != nil {
		data, err := json.Marshal(s.CompressionState)
		if err == nil {
			model.CompressionStateJSON = string(data)
		}
	}

	return model
}

func (m *MessageModel) ToDomain() session.Message {
	msg := session.Message{
		ID:         m.ID,
		Role:       session.Role(m.Role),
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
		CreatedAt:  m.CreatedAt,
	}

	if m.ToolCallsJSON != "" {
		var toolCalls []session.ToolCall
		if err := json.Unmarshal([]byte(m.ToolCallsJSON), &toolCalls); err == nil {
			msg.ToolCalls = toolCalls
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
