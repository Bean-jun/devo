package llmclient

import (
	"context"
	"fmt"

	"devo/internal/core/session"
)

type Client interface {
	Complete(ctx context.Context, messages []session.Message, systemPrompt string) (string, error)
}

type MockClient struct{}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (m *MockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (string, error) {
	if len(messages) == 0 {
		return "No messages to respond to.", nil
	}

	lastMsg := messages[len(messages)-1]
	if lastMsg.Role != session.RoleUser {
		return "I received your message.", nil
	}

	return fmt.Sprintf("Echo: %s", lastMsg.Content), nil
}
