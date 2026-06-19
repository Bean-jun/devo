package compressor

import (
	"context"
	"fmt"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

const (
	DefaultCompressThreshold = 60
	DefaultKeepRecent        = 20
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

func (c *Compressor) Compress(ctx context.Context, sessionID string, eventBus *session.EventBus) (*CompressResult, error) {
	msgs, _, err := c.store.GetMessages(sessionID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	sess, err := c.store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	threshold := sess.CompressThreshold
	if threshold <= 0 {
		threshold = DefaultCompressThreshold
	}

	keepRecent := sess.KeepRecent
	if keepRecent <= 0 {
		keepRecent = DefaultKeepRecent
	}

	if len(msgs) <= threshold {
		return nil, nil
	}

	compressible, toCompress := selectMessagesToCompress(msgs, keepRecent)
	if len(toCompress) == 0 {
		return nil, nil
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

	for _, msg := range msgs[:splitIdx] {
		if msg.Role == session.RoleSystem || msg.Role == session.RoleTool {
			remaining = append(remaining, msg)
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

	summaryPrompt := "请将以下对话片段压缩为一条简洁的摘要，保留关键决策、技术细节和重要上下文："
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

	result, err := c.llmClient.Complete(ctx, summaryReq, "你是一个专业的对话摘要生成器。请用简洁的中文或英文总结对话的关键点。")
	if err != nil {
		return "", fmt.Errorf("llm complete for summary: %w", err)
	}

	return result.Text, nil
}

func estimateTokens(msgs []session.Message) int {
	total := 0
	for _, msg := range msgs {
		total += len(msg.Content) / 4
		total += len(msg.ID) / 4
	}
	return total
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
