package llmclient

import (
	"context"
	"fmt"

	"devo/internal/core/session"
)

type CompleteResult struct {
	Text      string             `json:"text"`
	ToolCalls []session.ToolCall `json:"tool_calls"`
}

type Client interface {
	Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*CompleteResult, error)
}

type MockClient struct {
	Tools []ToolDefinition
}

type ToolDefinition struct {
	Name        string
	Description string
	Params      map[string]interface{}
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*CompleteResult, error) {
	if len(messages) == 0 {
		return &CompleteResult{Text: "No messages to respond to."}, nil
	}

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &CompleteResult{Text: fmt.Sprintf("I received the result of the tool call: %s", lastMsg.Content)}, nil
	}

	if lastMsg.Role != session.RoleUser {
		return &CompleteResult{Text: "I received your message."}, nil
	}

	return &CompleteResult{Text: fmt.Sprintf("Echo: %s", lastMsg.Content)}, nil
}
