package rest

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"devo/internal/core/session"
)

func TestGetSessionUsage_Success(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	sess := &session.Session{
		ID:               "sess-usage-1",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage: session.TokenUsage{
			Input:  500,
			Output: 200,
			Total:  700,
		},
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-usage-1/usage")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	totalInput, ok := result["total_input_tokens"].(float64)
	if !ok {
		t.Fatal("missing total_input_tokens")
	}
	if int(totalInput) != 500 {
		t.Errorf("expected total_input_tokens 500, got %v", totalInput)
	}

	totalOutput, ok := result["total_output_tokens"].(float64)
	if !ok {
		t.Fatal("missing total_output_tokens")
	}
	if int(totalOutput) != 200 {
		t.Errorf("expected total_output_tokens 200, got %v", totalOutput)
	}

	totalTokens, ok := result["total_tokens"].(float64)
	if !ok {
		t.Fatal("missing total_tokens")
	}
	if int(totalTokens) != 700 {
		t.Errorf("expected total_tokens 700, got %v", totalTokens)
	}

	compressionCount, ok := result["compression_count"].(float64)
	if !ok {
		t.Fatal("missing compression_count")
	}
	if int(compressionCount) != 0 {
		t.Errorf("expected compression_count 0, got %v", compressionCount)
	}
}

func TestGetSessionUsage_NotFound(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/sessions/nonexistent/usage")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestGetSessionUsage_WithSteps(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	sess := &session.Session{
		ID:               "sess-usage-steps",
		WorkingDirectory: t.TempDir(),
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage: session.TokenUsage{
			Input:  300,
			Output: 150,
			Total:  450,
		},
	}
	store.Create(sess)

	store.AddUsageStep("sess-usage-steps", 1, 100, 50, "estimated")
	store.AddUsageStep("sess-usage-steps", 2, 200, 100, "estimated")

	resp, err := http.Get(server.URL + "/api/v1/sessions/sess-usage-steps/usage")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	steps, ok := result["steps"].([]interface{})
	if !ok {
		t.Fatal("expected steps array in response")
	}
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}
}

func TestGetUsageStats_ByProject(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	sess1 := &session.Session{
		ID:               "sess-stats-1",
		WorkingDirectory: "/tmp/project-a",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage:       session.TokenUsage{Input: 100, Output: 50, Total: 150},
	}
	sess2 := &session.Session{
		ID:               "sess-stats-2",
		WorkingDirectory: "/tmp/project-b",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage:       session.TokenUsage{Input: 200, Output: 80, Total: 280},
	}
	store.Create(sess1)
	store.Create(sess2)

	resp, err := http.Get(server.URL + "/api/v1/usage/stats?group_by=project")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	groups, ok := result["groups"].([]interface{})
	if !ok {
		t.Fatal("expected groups array in response")
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	summary, ok := result["summary"].(map[string]any)
	if !ok {
		t.Fatal("expected summary in response")
	}
	if int(summary["total"].(float64)) != 430 {
		t.Errorf("expected summary total 430, got %v", summary["total"])
	}
}

func TestGetUsageStats_BySession(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	sess := &session.Session{
		ID:               "sess-stats-single",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage:       session.TokenUsage{Input: 50, Output: 25, Total: 75},
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/usage/stats?group_by=session")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	groups, ok := result["groups"].([]interface{})
	if !ok {
		t.Fatal("expected groups array in response")
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
}

func TestGetUsageStats_ByDate(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	sess := &session.Session{
		ID:               "sess-stats-date",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage:       session.TokenUsage{Input: 30, Output: 15, Total: 45},
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/usage/stats?group_by=date")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	groups, ok := result["groups"].([]interface{})
	if !ok {
		t.Fatal("expected groups array in response")
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
}

func TestGetUsageStats_WithProjectFilter(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	sess1 := &session.Session{
		ID:               "sess-filter-1",
		WorkingDirectory: "/tmp/project-a",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage:       session.TokenUsage{Input: 100, Output: 50, Total: 150},
	}
	sess2 := &session.Session{
		ID:               "sess-filter-2",
		WorkingDirectory: "/tmp/project-b",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage:       session.TokenUsage{Input: 200, Output: 80, Total: 280},
	}
	store.Create(sess1)
	store.Create(sess2)

	resp, err := http.Get(server.URL + "/api/v1/usage/stats?group_by=project&project=/tmp/project-a")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	groups, ok := result["groups"].([]interface{})
	if !ok {
		t.Fatal("expected groups array in response")
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group with filter, got %d", len(groups))
	}
}

func TestGetUsageStats_DefaultGroupBy(t *testing.T) {
	server, store := setupTestServer()
	defer server.Close()

	sess := &session.Session{
		ID:               "sess-stats-default",
		WorkingDirectory: "/tmp/test",
		State:            session.StateIdle,
		CreatedAt:        time.Now(),
		LastActiveAt:     time.Now(),
		TokenUsage:       session.TokenUsage{Input: 10, Output: 5, Total: 15},
	}
	store.Create(sess)

	resp, err := http.Get(server.URL + "/api/v1/usage/stats")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestGetUsageStats_Empty(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/usage/stats?group_by=project")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)

	groups, ok := result["groups"].([]interface{})
	if !ok {
		t.Fatal("expected groups array in response")
	}
	if len(groups) != 0 {
		t.Fatalf("expected 0 groups, got %d", len(groups))
	}
}
