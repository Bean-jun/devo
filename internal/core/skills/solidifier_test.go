package skills

import (
	"context"
	"strings"
	"testing"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

func TestBuildConversationText(t *testing.T) {
	msgs := []session.Message{
		{
			ID:        "msg1",
			Role:      session.RoleUser,
			Content:   "Help me fix this bug",
			CreatedAt: time.Now(),
		},
		{
			ID:        "msg2",
			Role:      session.RoleAssistant,
			Content:   "",
			ToolCalls: []session.ToolCall{{ID: "tc1", ToolName: "read_file", Params: map[string]interface{}{"path": "main.go"}}},
			CreatedAt: time.Now(),
		},
		{
			ID:         "msg3",
			Role:       session.RoleTool,
			Content:    "package main\n\nfunc main() { ... }",
			ToolCallID: "tc1",
			CreatedAt:  time.Now(),
		},
	}

	text := buildConversationText(msgs)

	if !strings.Contains(text, "Help me fix this bug") {
		t.Error("expected user message in conversation")
	}
	if !strings.Contains(text, "read_file") {
		t.Error("expected tool call in conversation")
	}
	if !strings.Contains(text, "package main") {
		t.Error("expected tool result in conversation")
	}
}

func TestCleanSkillContent(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"```\n# Test\n\nContent\n```", "# Test\n\nContent"},
		{"```markdown\n# Test\nContent\n```", "# Test\nContent"},
		{"# Test\nContent", "# Test\nContent"},
		{"\n\n   # Test\nContent   \n", "# Test\nContent"},
	}

	for _, tt := range tests {
		result := cleanSkillContent(tt.input)
		if result != tt.expected {
			t.Errorf("cleanSkillContent(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSolidify_NoMessages(t *testing.T) {
	mockStore := &mockSessionStore{messages: map[string][]session.Message{}}
	mockLLM := llmclient.NewMockClient()
	mgr := NewManager(t.TempDir())
	solidifier := NewSolidifier(mockLLM, mgr, mockStore)

	result, err := solidifier.SolidifySession(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("SolidifySession: %v", err)
	}
	if !result.NoSkill {
		t.Error("expected NoSkill=true for empty session")
	}
}

type mockSessionStore struct {
	sessions map[string]*session.Session
	messages map[string][]session.Message
}

func (m *mockSessionStore) Create(s *session.Session) error {
	if m.sessions == nil {
		m.sessions = make(map[string]*session.Session)
	}
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionStore) Get(id string) (*session.Session, error) {
	return m.sessions[id], nil
}

func (m *mockSessionStore) Update(s *session.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockSessionStore) ListSessions(status, project string, limit, offset int) ([]session.Session, int, error) {
	return nil, 0, nil
}

func (m *mockSessionStore) AddMessage(sessionID string, msg session.Message) error {
	if m.messages == nil {
		m.messages = make(map[string][]session.Message)
	}
	m.messages[sessionID] = append(m.messages[sessionID], msg)
	return nil
}

func (m *mockSessionStore) GetMessages(sessionID string, limit, offset int) ([]session.Message, int, error) {
	msgs := m.messages[sessionID]
	return msgs, len(msgs), nil
}

func (m *mockSessionStore) GetEventBus(sessionID string) (*session.EventBus, error) {
	return nil, nil
}

func (m *mockSessionStore) AddEvent(sessionID string, event session.Event) error {
	return nil
}

func (m *mockSessionStore) GetEvents(sessionID string, sinceID int64) ([]session.Event, error) {
	return nil, nil
}

func (m *mockSessionStore) IncrementSSEConnections(sessionID string) error {
	return nil
}

func (m *mockSessionStore) DecrementSSEConnections(sessionID string) error {
	return nil
}

func (m *mockSessionStore) AddUsageStep(sessionID string, stepSeq int, inputTokens, outputTokens int, source string) error {
	return nil
}

func (m *mockSessionStore) GetUsageSteps(sessionID string) ([]session.UsageStepRecord, error) {
	return nil, nil
}

func (m *mockSessionStore) UpdateSessionUsage(sessionID string, inputTokens, outputTokens int) error {
	return nil
}

func (m *mockSessionStore) GetUsageStats(groupBy, dateRange, project string) (*session.UsageStatsResult, error) {
	return nil, nil
}

func (m *mockSessionStore) Close() error {
	return nil
}

func (m *mockSessionStore) DeleteMessagesAfter(sessionID string, messageID string) (int, error) {
	return 0, nil
}

func (m *mockSessionStore) GetMessageByID(sessionID string, messageID string) (*session.Message, error) {
	return nil, nil
}

func (m *mockSessionStore) RecordFileModification(record session.FileModificationRecord) error {
	return nil
}

func (m *mockSessionStore) GetFileModifications(sessionID string) ([]session.FileModificationRecord, error) {
	return nil, nil
}

func (m *mockSessionStore) DeleteFileModificationsAfter(sessionID string, afterTime time.Time) error {
	return nil
}

func (m *mockSessionStore) ListUniqueWorkspaces() ([]string, error) {
	return nil, nil
}

func (m *mockSessionStore) DeleteByWorkspace(path string) (int, error) {
	return 0, nil
}

func (m *mockSessionStore) DeleteSession(id string) error {
	return nil
}

func (m *mockSessionStore) GetLastMessages(sessionIDs []string) (map[string]session.LastMessageInfo, error) {
	return nil, nil
}
