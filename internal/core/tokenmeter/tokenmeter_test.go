package tokenmeter

import (
	"testing"

	"devo/internal/core/session"
)

func TestNewMeter(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	if meter == nil {
		t.Fatal("expected non-nil meter")
	}
	if meter.store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestRecordStep(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
	}
	store.Create(sess)

	usage := &TokenUsage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
		Source:       SourceEstimated,
	}

	err := meter.RecordStep("sess-1", 1, usage)
	if err != nil {
		t.Fatalf("RecordStep failed: %v", err)
	}

	updated, _ := store.Get("sess-1")
	if updated.TokenUsage.Input != 100 {
		t.Errorf("expected input tokens 100, got %d", updated.TokenUsage.Input)
	}
	if updated.TokenUsage.Output != 50 {
		t.Errorf("expected output tokens 50, got %d", updated.TokenUsage.Output)
	}
	if updated.TokenUsage.Total != 150 {
		t.Errorf("expected total tokens 150, got %d", updated.TokenUsage.Total)
	}

	steps, err := meter.GetUsageSteps("sess-1")
	if err != nil {
		t.Fatalf("GetUsageSteps failed: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
	if steps[0].StepSeq != 1 {
		t.Errorf("expected step_seq 1, got %d", steps[0].StepSeq)
	}
	if steps[0].InputTokens != 100 {
		t.Errorf("expected input tokens 100, got %d", steps[0].InputTokens)
	}
	if steps[0].OutputTokens != 50 {
		t.Errorf("expected output tokens 50, got %d", steps[0].OutputTokens)
	}
	if steps[0].Source != string(SourceEstimated) {
		t.Errorf("expected source estimated, got %s", steps[0].Source)
	}
}

func TestRecordStepNilUsage(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
	}
	store.Create(sess)

	err := meter.RecordStep("sess-1", 1, nil)
	if err != nil {
		t.Fatalf("RecordStep with nil usage should not error: %v", err)
	}
}

func TestRecordStepMultipleSteps(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
	}
	store.Create(sess)

	usages := []*TokenUsage{
		{InputTokens: 100, OutputTokens: 50, TotalTokens: 150, Source: SourceEstimated},
		{InputTokens: 200, OutputTokens: 80, TotalTokens: 280, Source: SourceEstimated},
		{InputTokens: 150, OutputTokens: 60, TotalTokens: 210, Source: SourceExact},
	}

	for i, u := range usages {
		if err := meter.RecordStep("sess-1", i+1, u); err != nil {
			t.Fatalf("RecordStep %d failed: %v", i+1, err)
		}
	}

	updated, _ := store.Get("sess-1")
	expectedInput := 100 + 200 + 150
	expectedOutput := 50 + 80 + 60
	expectedTotal := expectedInput + expectedOutput

	if updated.TokenUsage.Input != expectedInput {
		t.Errorf("expected input tokens %d, got %d", expectedInput, updated.TokenUsage.Input)
	}
	if updated.TokenUsage.Output != expectedOutput {
		t.Errorf("expected output tokens %d, got %d", expectedOutput, updated.TokenUsage.Output)
	}
	if updated.TokenUsage.Total != expectedTotal {
		t.Errorf("expected total tokens %d, got %d", expectedTotal, updated.TokenUsage.Total)
	}

	steps, _ := meter.GetUsageSteps("sess-1")
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}
}

func TestGetSessionUsage(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		TokenUsage: session.TokenUsage{
			Input:  500,
			Output: 200,
			Total:  700,
		},
	}
	store.Create(sess)

	usage, err := meter.GetSessionUsage("sess-1")
	if err != nil {
		t.Fatalf("GetSessionUsage failed: %v", err)
	}
	if usage.Input != 500 {
		t.Errorf("expected input 500, got %d", usage.Input)
	}
	if usage.Output != 200 {
		t.Errorf("expected output 200, got %d", usage.Output)
	}
	if usage.Total != 700 {
		t.Errorf("expected total 700, got %d", usage.Total)
	}
}

func TestGetUsageStatsByProject(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess1 := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/project-a",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 100, Output: 50, Total: 150},
	}
	sess2 := &session.Session{
		ID:               "sess-2",
		WorkingDirectory: "/tmp/project-b",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 200, Output: 80, Total: 280},
	}
	sess3 := &session.Session{
		ID:               "sess-3",
		WorkingDirectory: "/tmp/project-a",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 300, Output: 100, Total: 400},
	}

	store.Create(sess1)
	store.Create(sess2)
	store.Create(sess3)

	stats, err := meter.GetUsageStats("project", "", "")
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if len(stats.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(stats.Groups))
	}

	projectA := stats.Groups[0]
	projectB := stats.Groups[1]

	if projectA.Key != "/tmp/project-a" && projectA.Key != "/tmp/project-b" {
		t.Errorf("unexpected group key: %s", projectA.Key)
	}

	if projectA.Key == "/tmp/project-a" {
		if projectA.InputTokens != 400 {
			t.Errorf("project-a: expected input 400, got %d", projectA.InputTokens)
		}
		if projectA.OutputTokens != 150 {
			t.Errorf("project-a: expected output 150, got %d", projectA.OutputTokens)
		}
		if projectA.TotalTokens != 550 {
			t.Errorf("project-a: expected total 550, got %d", projectA.TotalTokens)
		}
	}

	if projectB.Key == "/tmp/project-b" {
		if projectB.InputTokens != 200 {
			t.Errorf("project-b: expected input 200, got %d", projectB.InputTokens)
		}
	}

	if stats.Summary.Total != 830 {
		t.Errorf("expected summary total 830, got %d", stats.Summary.Total)
	}
}

func TestGetUsageStatsBySession(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess1 := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 100, Output: 50, Total: 150},
	}
	sess2 := &session.Session{
		ID:               "sess-2",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 200, Output: 80, Total: 280},
	}

	store.Create(sess1)
	store.Create(sess2)

	stats, err := meter.GetUsageStats("session", "", "")
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if len(stats.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(stats.Groups))
	}

	if stats.Summary.Total != 430 {
		t.Errorf("expected summary total 430, got %d", stats.Summary.Total)
	}
}

func TestGetUsageStatsByDate(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 100, Output: 50, Total: 150},
	}
	store.Create(sess)

	stats, err := meter.GetUsageStats("date", "", "")
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if len(stats.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(stats.Groups))
	}
}

func TestGetUsageStatsWithProjectFilter(t *testing.T) {
	store := session.NewInMemoryStore()
	meter := NewMeter(store)

	sess1 := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: "/tmp/project-a",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 100, Output: 50, Total: 150},
	}
	sess2 := &session.Session{
		ID:               "sess-2",
		WorkingDirectory: "/tmp/project-b",
		State:            session.StateIdle,
		TokenUsage:       session.TokenUsage{Input: 200, Output: 80, Total: 280},
	}

	store.Create(sess1)
	store.Create(sess2)

	stats, err := meter.GetUsageStats("project", "", "/tmp/project-a")
	if err != nil {
		t.Fatalf("GetUsageStats failed: %v", err)
	}

	if len(stats.Groups) != 1 {
		t.Fatalf("expected 1 group with filter, got %d", len(stats.Groups))
	}
	if stats.Groups[0].Key != "/tmp/project-a" {
		t.Errorf("expected key /tmp/project-a, got %s", stats.Groups[0].Key)
	}
	if stats.Summary.Total != 150 {
		t.Errorf("expected summary total 150, got %d", stats.Summary.Total)
	}
}

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		tokens   int
		expected string
	}{
		{0, "$0.0000"},
		{1000, "$0.0100"},
		{5000, "$0.0500"},
		{10000, "$0.1000"},
	}

	for _, tt := range tests {
		got := EstimateCost(tt.tokens)
		if got != tt.expected {
			t.Errorf("EstimateCost(%d): expected %s, got %s", tt.tokens, tt.expected, got)
		}
	}
}

func TestSourceConstants(t *testing.T) {
	if SourceExact != "exact" {
		t.Errorf("expected SourceExact='exact', got '%s'", SourceExact)
	}
	if SourceEstimated != "estimated" {
		t.Errorf("expected SourceEstimated='estimated', got '%s'", SourceEstimated)
	}
}
