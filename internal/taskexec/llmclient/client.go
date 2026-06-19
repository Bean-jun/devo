package llmclient

import (
	"context"
	"fmt"

	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
)

type CompleteResult struct {
	Text       string                 `json:"text"`
	ToolCalls  []session.ToolCall     `json:"tool_calls"`
	TokenUsage *tokenmeter.TokenUsage `json:"token_usage,omitempty"`
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
		return &CompleteResult{
			Text: "No messages to respond to.",
			TokenUsage: &tokenmeter.TokenUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
				Source:       tokenmeter.SourceEstimated,
			},
		}, nil
	}

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		inputTokens := len(messages) * 4
		outputTokens := len(lastMsg.Content) / 4
		if outputTokens < 5 {
			outputTokens = 5
		}
		return &CompleteResult{
			Text: fmt.Sprintf("I received the result of the tool call: %s", lastMsg.Content),
			TokenUsage: &tokenmeter.TokenUsage{
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				TotalTokens:  inputTokens + outputTokens,
				Source:       tokenmeter.SourceEstimated,
			},
		}, nil
	}

	if lastMsg.Role != session.RoleUser {
		return &CompleteResult{
			Text: "I received your message.",
			TokenUsage: &tokenmeter.TokenUsage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
				Source:       tokenmeter.SourceEstimated,
			},
		}, nil
	}

	outputText := fmt.Sprintf("Echo: %s", lastMsg.Content)
	inputTokens := len(messages)*4 + len(systemPrompt)/4
	outputTokens := len(outputText) / 4
	if outputTokens < 5 {
		outputTokens = 5
	}

	return &CompleteResult{
		Text: outputText,
		TokenUsage: &tokenmeter.TokenUsage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  inputTokens + outputTokens,
			Source:       tokenmeter.SourceEstimated,
		},
	}, nil
}
