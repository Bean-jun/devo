package memory

import (
	"errors"
	"time"

	"devo/internal/core/session"
)

type MemoryType string

const (
	TypeUser    MemoryType = "user"
	TypeProject MemoryType = "project"
)

type MemorySource string

const (
	SourceManual MemorySource = "manual"
	SourceAuto   MemorySource = "auto"
)

type Memory struct {
	ID        string       `json:"id"`
	Type      MemoryType   `json:"type"`
	Key       string       `json:"key"`
	Content   string       `json:"content"`
	Source    MemorySource `json:"source"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type MemoryDraft struct {
	Type             MemoryType
	Key              string
	SuggestedContent string
	WorkingDir       string
}

type ApprovalDrafts struct {
	SessionID string
	Drafts    []MemoryDraft
}

var (
	ErrMemoryNotFound = errors.New("memory not found")
)

func NewMemory(typ MemoryType, key, content string, source MemorySource) *Memory {
	now := time.Now()
	return &Memory{
		ID:        session.GenerateID("mem"),
		Type:      typ,
		Key:       key,
		Content:   content,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
