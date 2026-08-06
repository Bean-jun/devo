package llmclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
)

type CompleteResult struct {
	Text       string                 `json:"text"`
	Reasoning  string                 `json:"reasoning,omitempty"`
	ToolCalls  []session.ToolCall     `json:"tool_calls"`
	TokenUsage *tokenmeter.TokenUsage `json:"token_usage,omitempty"`
}

type StreamEvent struct {
	Type          string                 `json:"type"`
	Token         string                 `json:"token,omitempty"`
	Reasoning     string                 `json:"reasoning,omitempty"`
	FullText      string                 `json:"full_text,omitempty"`
	FullReasoning string                 `json:"full_reasoning,omitempty"`
	ToolCalls     []session.ToolCall     `json:"tool_calls,omitempty"`
	FinishReason  string                 `json:"finish_reason,omitempty"`
	TokenUsage    *tokenmeter.TokenUsage `json:"token_usage,omitempty"`
	Err           error                  `json:"-"`
}

type StreamCallback func(event StreamEvent)

type Client interface {
	Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*CompleteResult, error)
	CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback StreamCallback) error
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

func (m *MockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(StreamEvent{Type: "error", Err: err})
		return err
	}

	words := strings.Fields(result.Text)
	for i, word := range words {
		fullText := strings.Join(words[:i+1], " ")
		select {
		case <-ctx.Done():
			callback(StreamEvent{Type: "error", Err: ctx.Err()})
			return ctx.Err()
		default:
		}
		callback(StreamEvent{
			Type:     "token",
			Token:    word,
			FullText: fullText,
		})
		time.Sleep(1 * time.Millisecond)
	}

	callback(StreamEvent{
		Type:         "done",
		FullText:     result.Text,
		ToolCalls:    result.ToolCalls,
		FinishReason: "stop",
		TokenUsage:   result.TokenUsage,
	})

	return nil
}
