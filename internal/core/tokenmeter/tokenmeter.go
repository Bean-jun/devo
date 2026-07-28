package tokenmeter

import (
	"fmt"

	"devo/internal/core/session"
)

type Source string

const (
	SourceExact     Source = "exact"
	SourceEstimated Source = "estimated"
)

type TokenUsage struct {
	InputTokens     int    `json:"input_tokens"`
	OutputTokens    int    `json:"output_tokens"`
	ReasoningTokens int    `json:"reasoning_tokens,omitempty"`
	TotalTokens     int    `json:"total_tokens"`
	CachedTokens    int    `json:"cached_tokens,omitempty"`
	Source          Source `json:"source"`
}

type Meter struct {
	store session.SessionStore
}

func NewMeter(store session.SessionStore) *Meter {
	return &Meter{store: store}
}

func (m *Meter) RecordStep(sessionID string, stepSeq int, usage *TokenUsage) error {
	if usage == nil {
		return nil
	}

	if err := m.store.AddUsageStep(sessionID, stepSeq, usage.InputTokens, usage.OutputTokens, string(usage.Source)); err != nil {
		return fmt.Errorf("add usage step: %w", err)
	}

	if err := m.store.UpdateSessionUsage(sessionID, usage.InputTokens, usage.OutputTokens); err != nil {
		return fmt.Errorf("update session usage: %w", err)
	}

	return nil
}

func (m *Meter) GetSessionUsage(sessionID string) (session.TokenUsage, error) {
	sess, err := m.store.Get(sessionID)
	if err != nil {
		return session.TokenUsage{}, err
	}
	return sess.TokenUsage, nil
}

func (m *Meter) GetUsageSteps(sessionID string) ([]session.UsageStepRecord, error) {
	return m.store.GetUsageSteps(sessionID)
}

func (m *Meter) GetUsageStats(groupBy, dateRange, project string) (*session.UsageStatsResult, error) {
	return m.store.GetUsageStats(groupBy, dateRange, project)
}

func EstimateCost(totalTokens int) string {
	costPer1K := 0.01
	cost := float64(totalTokens) / 1000.0 * costPer1K
	return fmt.Sprintf("$%.4f", cost)
}

func EstimateTokens(s string) int {
	asciiChars, cjkChars := 0, 0
	for _, r := range s {
		if (r >= 0x4E00 && r <= 0x9FFF) ||
			(r >= 0x3400 && r <= 0x4DBF) ||
			(r >= 0x20000 && r <= 0x2A6DF) ||
			(r >= 0x3040 && r <= 0x30FF) ||
			(r >= 0xAC00 && r <= 0xD7AF) {
			cjkChars++
		} else {
			asciiChars++
		}
	}
	return asciiChars/3 + cjkChars
}
