package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

type stubTool struct{}

func (s *stubTool) Name() string                                       { return "stub" }
func (s *stubTool) Description() string                                { return "stub tool" }
func (s *stubTool) RiskLevel() tools.RiskLevel                          { return tools.RiskLevelNone }
func (s *stubTool) ParamsSchema() map[string]interface{}                 { return nil }
func (s *stubTool) Execute(context.Context, string, map[string]interface{}, tools.StreamWriter) error {
	return nil
}

func newSSEServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatalf("response writer does not support flushing")
		}
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		for _, c := range chunks {
			_, _ = io.WriteString(w, c)
			flusher.Flush()
		}
	}))
}

func TestParseSSEStream_ReasoningContent(t *testing.T) {
	chunks := []string{
		`data: {"choices":[{"delta":{"reasoning_content":"先思考一下"},"finish_reason":""}]}` + "\n\n",
		`data: {"choices":[{"delta":{"reasoning_content":"这个问题"},"finish_reason":""}]}` + "\n\n",
		`data: {"choices":[{"delta":{"content":"答案是"},"finish_reason":""}]}` + "\n\n",
		`data: {"choices":[{"delta":{"content":"42"},"finish_reason":""}]}` + "\n\n",
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n",
		`data: {"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"completion_tokens_details":{"reasoning_tokens":4}}}` + "\n\n",
		`data: [DONE]` + "\n\n",
	}
	server := newSSEServer(t, chunks)
	defer server.Close()

	cfg := Config{BaseURL: server.URL, APIKey: "test"}
	c := New(cfg, nil)

	var reasoningTokens []string
	var contentTokens []string
	var doneEvent *llmclient.StreamEvent

	err := c.CompleteStream(context.Background(), nil, "system", func(evt llmclient.StreamEvent) {
		switch evt.Type {
		case "reasoning_token":
			reasoningTokens = append(reasoningTokens, evt.Reasoning)
		case "token":
			contentTokens = append(contentTokens, evt.Token)
		case "done":
			cp := evt
			doneEvent = &cp
		}
	})
	if err != nil {
		t.Fatalf("CompleteStream failed: %v", err)
	}

	if len(reasoningTokens) != 2 {
		t.Errorf("expected 2 reasoning tokens, got %d (%v)", len(reasoningTokens), reasoningTokens)
	}
	if reasoningTokens[0] != "先思考一下" || reasoningTokens[1] != "这个问题" {
		t.Errorf("unexpected reasoning tokens: %v", reasoningTokens)
	}
	if len(contentTokens) != 2 {
		t.Errorf("expected 2 content tokens, got %d", len(contentTokens))
	}
	if contentTokens[0] != "答案是" || contentTokens[1] != "42" {
		t.Errorf("unexpected content tokens: %v", contentTokens)
	}

	if doneEvent == nil {
		t.Fatal("expected done event")
	}
	if doneEvent.FullReasoning != "先思考一下这个问题" {
		t.Errorf("expected full reasoning '先思考一下这个问题', got %q", doneEvent.FullReasoning)
	}
	if doneEvent.FullText != "答案是42" {
		t.Errorf("expected full text '答案是42', got %q", doneEvent.FullText)
	}
	if doneEvent.TokenUsage == nil {
		t.Fatal("expected token usage")
	}
	if doneEvent.TokenUsage.ReasoningTokens != 4 {
		t.Errorf("expected 4 reasoning tokens, got %d", doneEvent.TokenUsage.ReasoningTokens)
	}
}

func TestParseSSEStream_ReasoningField(t *testing.T) {
	chunks := []string{
		`data: {"choices":[{"delta":{"reasoning":"openai style thinking"},"finish_reason":""}]}` + "\n\n",
		`data: {"choices":[{"delta":{"content":"final"},"finish_reason":"stop"}]}` + "\n\n",
		`data: [DONE]` + "\n\n",
	}
	server := newSSEServer(t, chunks)
	defer server.Close()

	cfg := Config{BaseURL: server.URL, APIKey: "test"}
	c := New(cfg, nil)

	var reasoning []string
	err := c.CompleteStream(context.Background(), nil, "", func(evt llmclient.StreamEvent) {
		if evt.Type == "reasoning_token" {
			reasoning = append(reasoning, evt.Reasoning)
		}
	})
	if err != nil {
		t.Fatalf("CompleteStream failed: %v", err)
	}
	if len(reasoning) != 1 || reasoning[0] != "openai style thinking" {
		t.Errorf("unexpected reasoning tokens: %v", reasoning)
	}
}

func TestParseSSEStream_NoReasoning(t *testing.T) {
	chunks := []string{
		`data: {"choices":[{"delta":{"content":"hello"},"finish_reason":"stop"}]}` + "\n\n",
		`data: [DONE]` + "\n\n",
	}
	server := newSSEServer(t, chunks)
	defer server.Close()

	cfg := Config{BaseURL: server.URL, APIKey: "test"}
	c := New(cfg, nil)

	var reasoningEvents int
	err := c.CompleteStream(context.Background(), nil, "", func(evt llmclient.StreamEvent) {
		if evt.Type == "reasoning_token" {
			reasoningEvents++
		}
	})
	if err != nil {
		t.Fatalf("CompleteStream failed: %v", err)
	}
	if reasoningEvents != 0 {
		t.Errorf("expected 0 reasoning events, got %d", reasoningEvents)
	}
}

func TestComplete_NonStreamWithReasoningContent(t *testing.T) {
	respBody := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"role":             "assistant",
					"content":          "final answer",
					"reasoning_content": "thinking process",
				},
			},
		},
		"usage": map[string]any{
			"prompt_tokens":                5,
			"completion_tokens":            3,
			"total_tokens":                 8,
			"completion_tokens_details":    map[string]any{"reasoning_tokens": 2},
		},
	}
	body, _ := json.Marshal(respBody)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	cfg := Config{BaseURL: server.URL, APIKey: "test"}
	c := New(cfg, nil)

	result, err := c.Complete(context.Background(), []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "hi"},
	}, "system")
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if result.Reasoning != "thinking process" {
		t.Errorf("expected reasoning 'thinking process', got %q", result.Reasoning)
	}
	if result.Text != "final answer" {
		t.Errorf("expected text 'final answer', got %q", result.Text)
	}
	if result.TokenUsage == nil || result.TokenUsage.ReasoningTokens != 2 {
		t.Errorf("expected reasoning_tokens=2, got %+v", result.TokenUsage)
	}
}

func TestBuildChatRequest_ReasoningEffort(t *testing.T) {
	c := New(Config{ReasoningEffort: "high"}, nil)
	req := c.buildChatRequest(nil, "sys", false)
	if req.ReasoningEffort != "high" {
		t.Errorf("expected reasoning_effort 'high', got %q", req.ReasoningEffort)
	}

	c2 := New(Config{}, nil)
	req2 := c2.buildChatRequest(nil, "sys", false)
	if req2.ReasoningEffort != "" {
		t.Errorf("expected empty reasoning_effort, got %q", req2.ReasoningEffort)
	}
}

func TestExtractReasoningFromDelta_PrefersReasoningContent(t *testing.T) {
	got := extractReasoningFromDelta(openaiStreamDelta{
		ReasoningContent: "deepseek",
		Reasoning:        "ignored",
	})
	if got != "deepseek" {
		t.Errorf("expected 'deepseek', got %q", got)
	}

	got2 := extractReasoningFromDelta(openaiStreamDelta{Reasoning: "openai"})
	if got2 != "openai" {
		t.Errorf("expected 'openai', got %q", got2)
	}

	got3 := extractReasoningFromDelta(openaiStreamDelta{})
	if got3 != "" {
		t.Errorf("expected empty, got %q", got3)
	}
}

func TestConvertUsage_ReasoningTokens(t *testing.T) {
	usage := &openaiUsage{
		PromptTokens:             100,
		CompletionTokens:         50,
		TotalTokens:              150,
		CompletionTokensDetails: &openaiCompletionTokensDetails{ReasoningTokens: 30},
	}
	tu := convertUsage(usage)
	if tu.ReasoningTokens != 30 {
		t.Errorf("expected 30 reasoning tokens, got %d", tu.ReasoningTokens)
	}

	tu2 := convertUsage(&openaiUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15})
	if tu2.ReasoningTokens != 0 {
		t.Errorf("expected 0 reasoning tokens when details nil, got %d", tu2.ReasoningTokens)
	}
}

func TestBuildChatRequest_NoReasoningEffortByDefault(t *testing.T) {
	c := New(Config{}, nil)
	req := c.buildChatRequest(nil, "sys", true)
	body, _ := json.Marshal(req)
	if strings.Contains(string(body), "reasoning_effort") {
		t.Errorf("expected no reasoning_effort field in default request, got: %s", string(body))
	}
}
