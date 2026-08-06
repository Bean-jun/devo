package compressor

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
	"devo/internal/taskexec/llmclient"
)

const (
	summarySystemPrompt = `You are a specialized conversation summarizer for developer-AI coding sessions. Your task is to compress conversation history into a structured summary using the exact section format below.

Output must follow this structure exactly:

## Project State
- Current task/feature being worked on, overall progress
- What is working and what is not
- Pending TODOs and next steps

## Key Decisions
- Technical decisions made and their rationale (why option A over option B)
- Architecture choices, design patterns selected
- Any trade-offs discussed and accepted

## File Changes
- Files created, modified, or deleted
- Key code changes and their purpose
- Keep file paths, function names, class names, and variable names in original form

## Errors and Fixes
- Error messages encountered and their root causes
- Solutions applied and their effectiveness
- Any workarounds still in place

## Important Context
- Configuration values, environment details, dependency versions
- User preferences or constraints explicitly stated
- Any other information essential for continuing the work

Compression rules:
- Remove repetitive discussions and redundant back-and-forth
- Merge multiple interactions on the same topic into single entries
- Discard outdated information that has been superseded by later actions
- Keep the total summary under 20% of the original length
- If a section has no relevant content, write "None" under it

Output only the compressed summary, nothing else.`

	summaryUserPrefix = "Summarize the following conversation segment concisely:"
)

type Compressor struct {
	llmClient llmclient.Client
	store     session.SessionStore
}

func New(llmClient llmclient.Client, store session.SessionStore) *Compressor {
	return &Compressor{
		llmClient: llmClient,
		store:     store,
	}
}

type CompressResult struct {
	CompressedCount int
	TokensRemoved   int
	SummaryText     string
}

func (c *Compressor) ForceCompress(ctx context.Context, sessionID string, eventBus *session.EventBus, systemPromptTokens int) (*CompressResult, error) {
	return c.compress(ctx, sessionID, eventBus, systemPromptTokens, true)
}

func (c *Compressor) Compress(ctx context.Context, sessionID string, eventBus *session.EventBus, systemPromptTokens int) (*CompressResult, error) {
	return c.compress(ctx, sessionID, eventBus, systemPromptTokens, false)
}

func (c *Compressor) compress(ctx context.Context, sessionID string, eventBus *session.EventBus, systemPromptTokens int, force bool) (*CompressResult, error) {
	msgs, _, err := c.store.GetMessages(sessionID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	sess, err := c.store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	maxContext := sess.MaxContextTokens
	if maxContext <= 0 {
		maxContext = 128000
	}

	estimatedTokens := EstimateContextTokens(msgs) + systemPromptTokens
	compressThreshold := int(float64(maxContext) * 0.8)

	if !force && estimatedTokens <= compressThreshold {
		return nil, nil
	}

	keepRecent := sess.KeepRecent
	if keepRecent <= 0 {
		keepRecent = 30
	}

	remaining, toCompress := selectMessagesToCompress(msgs, keepRecent)
	if len(toCompress) == 0 {
		return nil, nil
	}

	summaryText, err := c.generateSummary(ctx, toCompress)
	if err != nil {
		return nil, fmt.Errorf("generate summary: %w", err)
	}

	compressedIDs := make([]string, len(toCompress))
	for i, msg := range toCompress {
		compressedIDs[i] = msg.ID
	}

	deletedCount, err := c.store.DeleteMessages(sessionID, compressedIDs)
	if err != nil {
		return nil, fmt.Errorf("delete compressed messages: %w", err)
	}

	summaryMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleSystem,
		Content:   fmt.Sprintf("[上下文压缩摘要] %s", summaryText),
		CreatedAt: time.Now(),
	}
	if err := c.store.AddMessage(sessionID, summaryMsg); err != nil {
		return nil, fmt.Errorf("add summary message: %w", err)
	}

	sess.CompressionCount++
	if err := c.store.Update(sess); err != nil {
		return nil, fmt.Errorf("update session compression count: %w", err)
	}

	tokensRemoved := estimateTokens(toCompress)

	if eventBus != nil {
		eventBus.Publish("context_compressed", map[string]any{
			"compressed_count": deletedCount,
			"tokens_removed":   tokensRemoved,
		})
	}

	_ = remaining

	return &CompressResult{
		CompressedCount: deletedCount,
		TokensRemoved:   tokensRemoved,
		SummaryText:     summaryText,
	}, nil
}

func selectMessagesToCompress(msgs []session.Message, keepRecent int) (remaining []session.Message, toCompress []session.Message) {
	if keepRecent >= len(msgs) {
		return msgs, nil
	}

	splitIdx := len(msgs) - keepRecent

	for {
		needIDs := make(map[string]bool)
		for i := splitIdx; i < len(msgs); i++ {
			if msgs[i].Role == session.RoleTool && msgs[i].ToolCallID != "" {
				needIDs[msgs[i].ToolCallID] = true
			}
		}

		if len(needIDs) == 0 {
			break
		}

		found := false
		for i := splitIdx - 1; i >= 0; i-- {
			if msgs[i].Role == session.RoleAssistant {
				for _, tc := range msgs[i].ToolCalls {
					if needIDs[tc.ID] {
						splitIdx = i
						found = true
						break
					}
				}
			}
			if found {
				break
			}
		}

		if !found {
			break
		}
	}

	toCompress = msgs[:splitIdx]
	remaining = msgs[splitIdx:]

	return remaining, toCompress
}

func (c *Compressor) generateSummary(ctx context.Context, msgs []session.Message) (string, error) {
	if len(msgs) == 0 {
		return "", nil
	}

	summaryPrompt := summaryUserPrefix
	for _, msg := range msgs {
		switch msg.Role {
		case session.RoleUser:
			summaryPrompt += fmt.Sprintf("\n[user]: %s", msg.Content)
		case session.RoleAssistant:
			if msg.Reasoning != "" {
				summaryPrompt += fmt.Sprintf("\n[assistant thinking]: %s", msg.Reasoning)
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					paramsJSON, _ := json.Marshal(tc.Params)
					summaryPrompt += fmt.Sprintf("\n[assistant → tool_call %s]: %s", tc.ToolName, string(paramsJSON))
				}
			}
			if msg.Content != "" {
				summaryPrompt += fmt.Sprintf("\n[assistant]: %s", msg.Content)
			}
		case session.RoleTool:
			summaryPrompt += fmt.Sprintf("\n[tool_result %s]: %s", msg.ToolCallID, msg.Content)
		case session.RoleSystem:
			summaryPrompt += fmt.Sprintf("\n[system]: %s", msg.Content)
		}
	}

	summaryReq := []session.Message{
		{
			ID:        session.GenerateID("msg"),
			Role:      session.RoleUser,
			Content:   summaryPrompt,
			CreatedAt: time.Now(),
		},
	}

	result, err := c.llmClient.Complete(ctx, summaryReq, summarySystemPrompt)
	if err != nil {
		return "", fmt.Errorf("llm complete for summary: %w", err)
	}

	return result.Text, nil
}

func estimateTokens(msgs []session.Message) int {
	total := 0
	for _, msg := range msgs {
		total += tokenmeter.EstimateTokens(msg.Content)
		total += tokenmeter.EstimateTokens(msg.ID)
		total += tokenmeter.EstimateTokens(msg.ToolCallID)
		for _, tc := range msg.ToolCalls {
			total += tokenmeter.EstimateTokens(tc.ID)
			total += tokenmeter.EstimateTokens(tc.ToolName)
			if paramsJSON, err := json.Marshal(tc.Params); err == nil {
				total += tokenmeter.EstimateTokens(string(paramsJSON))
			}
		}
	}
	return total
}

func EstimateContextTokens(msgs []session.Message) int {
	return estimateTokens(msgs)
}
