package sqlite

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"devo/internal/core/session"
)

type SessionModel struct {
	ID                   string    `gorm:"primaryKey;size:64"`
	Title                string    `gorm:"size:256"`
	WorkingDirectory     string    `gorm:"size:512;index:idx_working_dir"`
	State                string    `gorm:"size:32;index:idx_state"`
	CreatedAt            time.Time `gorm:"autoCreateTime"`
	LastActiveAt         time.Time
	ActiveSSEConnections int
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
	return &session.Session{
		ID:                   m.ID,
		Title:                m.Title,
		WorkingDirectory:     m.WorkingDirectory,
		State:                session.State(m.State),
		CreatedAt:            m.CreatedAt,
		LastActiveAt:         m.LastActiveAt,
		ActiveSSEConnections: m.ActiveSSEConnections,
	}
}

func fromDomain(s *session.Session) *SessionModel {
	return &SessionModel{
		ID:                   s.ID,
		Title:                s.Title,
		WorkingDirectory:     s.WorkingDirectory,
		State:                string(s.State),
		CreatedAt:            s.CreatedAt,
		LastActiveAt:         s.LastActiveAt,
		ActiveSSEConnections: s.ActiveSSEConnections,
	}
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
	return db.AutoMigrate(&SessionModel{}, &MessageModel{}, &EventModel{})
}
