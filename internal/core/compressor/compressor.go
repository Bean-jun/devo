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
	ProgressiveBatchSize = 10 // 渐进式压缩：每次最多压缩 10 条消息
	summarySystemPrompt  = `You are a specialized conversation summarizer for developer-AI coding sessions. Your task is to compress conversation history into a structured summary.

Requirements:
1. Preserve critical information:
   - All technical decisions and their rationale (why option A over option B)
   - File paths, function names, class names, and variable names
   - Specific code changes and their purpose
   - Error messages and their resolutions
   - Unfinished tasks and pending action items

2. Output format:
   - Use concise bullet points, avoid verbose narration
   - Group information chronologically or by topic
   - Keep code identifiers, file names, and error messages in their original form

3. Compression principles:
   - Remove repetitive discussions and redundant back-and-forth
   - Merge multiple interactions on the same topic
   - Discard outdated information that has been superseded by later actions
   - Keep the summary under 20% of the original length

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

func (c *Compressor) Compress(ctx context.Context, sessionID string, eventBus *session.EventBus, systemPromptTokens int) (*CompressResult, error) {
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

	estimatedTokens := EstimateContextTokens(msgs, sess.CompressionState) + systemPromptTokens
	compressThreshold := int(float64(maxContext) * 0.8)

	if estimatedTokens <= compressThreshold {
		return nil, nil
	}

	keepRecent := sess.KeepRecent
	if keepRecent <= 0 {
		keepRecent = 30
	}

	compressible, toCompress := selectMessagesToCompress(msgs, keepRecent)
	if len(toCompress) == 0 {
		return nil, nil
	}

	if len(toCompress) > ProgressiveBatchSize {
		toCompress = toCompress[:ProgressiveBatchSize]
	}

	summaryText, err := c.generateSummary(ctx, toCompress)
	if err != nil {
		return nil, fmt.Errorf("generate summary: %w", err)
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

	compressedRange := session.CompressedRange{
		StartMessageID: toCompress[0].ID,
		EndMessageID:   toCompress[len(toCompress)-1].ID,
	}

	compressionSummary := session.CompressionSummary{
		SummaryText: summaryText,
		CoversRange: compressedRange,
		CreatedAt:   time.Now(),
	}

	if sess.CompressionState == nil {
		sess.CompressionState = &session.CompressionState{}
	}
	sess.CompressionState.CompressedRanges = append(sess.CompressionState.CompressedRanges, compressedRange)
	sess.CompressionState.Summaries = append(sess.CompressionState.Summaries, compressionSummary)
	sess.CompressionCount++

	if err := c.store.Update(sess); err != nil {
		return nil, fmt.Errorf("update session compression state: %w", err)
	}

	tokensRemoved := estimateTokens(toCompress)

	if eventBus != nil {
		eventBus.Publish("context_compressed", map[string]any{
			"compressed_count": len(toCompress),
			"tokens_removed":   tokensRemoved,
		})
	}

	_ = compressible

	return &CompressResult{
		CompressedCount: len(toCompress),
		TokensRemoved:   tokensRemoved,
		SummaryText:     summaryText,
	}, nil
}

func selectMessagesToCompress(msgs []session.Message, keepRecent int) (remaining []session.Message, toCompress []session.Message) {
	if keepRecent >= len(msgs) {
		return msgs, nil
	}

	splitIdx := len(msgs) - keepRecent
	if splitIdx < 0 {
		splitIdx = 0
	}

	// Collect tool_call_ids from tool messages in the compressible area.
	// These tool results must be preserved, so their owning assistant
	// messages must also be preserved to form valid API request pairs.
	toolCallIDsInCompressible := make(map[string]bool)
	for _, msg := range msgs[:splitIdx] {
		if msg.Role == session.RoleTool && msg.ToolCallID != "" {
			toolCallIDsInCompressible[msg.ToolCallID] = true
		}
	}

	// Also collect tool_call_ids from the recent area, because an assistant
	// in the compressible area may own a tool call whose result is in the
	// recent area. If we compress the assistant, the tool result becomes
	// orphaned and the API call will fail with a 400 error.
	toolCallIDsInRecent := make(map[string]bool)
	for _, msg := range msgs[splitIdx:] {
		if msg.Role == session.RoleTool && msg.ToolCallID != "" {
			toolCallIDsInRecent[msg.ToolCallID] = true
		}
	}

	for _, msg := range msgs[:splitIdx] {
		if msg.Role == session.RoleSystem {
			remaining = append(remaining, msg)
		} else if msg.Role == session.RoleTool {
			remaining = append(remaining, msg)
		} else if msg.Role == session.RoleAssistant && len(msg.ToolCalls) > 0 {
			// Preserve assistant if any of its tool calls have results in
			// either the compressible area or the recent area.
			preserve := false
			for _, tc := range msg.ToolCalls {
				if toolCallIDsInCompressible[tc.ID] || toolCallIDsInRecent[tc.ID] {
					preserve = true
					break
				}
			}
			if preserve {
				remaining = append(remaining, msg)
			} else {
				toCompress = append(toCompress, msg)
			}
		} else {
			toCompress = append(toCompress, msg)
		}
	}

	remaining = append(remaining, msgs[splitIdx:]...)

	return remaining, toCompress
}

func (c *Compressor) generateSummary(ctx context.Context, msgs []session.Message) (string, error) {
	if len(msgs) == 0 {
		return "", nil
	}

	summaryPrompt := summaryUserPrefix
	for _, msg := range msgs {
		summaryPrompt += fmt.Sprintf("\n[%s]: %s", msg.Role, msg.Content)
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

func EstimateContextTokens(msgs []session.Message, compressionState *session.CompressionState) int {
	activeMsgs := FilterActiveMessages(msgs, compressionState)
	return estimateTokens(activeMsgs)
}

func FilterActiveMessages(msgs []session.Message, compressionState *session.CompressionState) []session.Message {
	if compressionState == nil || len(compressionState.CompressedRanges) == 0 {
		return msgs
	}

	compressedIDs := make(map[string]bool)
	for _, r := range compressionState.CompressedRanges {
		inRange := false
		for _, msg := range msgs {
			if msg.ID == r.StartMessageID {
				inRange = true
			}
			if inRange {
				compressedIDs[msg.ID] = true
			}
			if msg.ID == r.EndMessageID {
				inRange = false
			}
		}
	}

	var filtered []session.Message
	for _, msg := range msgs {
		if compressedIDs[msg.ID] {
			continue
		}
		filtered = append(filtered, msg)
	}

	return filtered
}
