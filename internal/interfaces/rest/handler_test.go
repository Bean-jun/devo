package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"devo/internal/config"
	"devo/internal/core/agent"
	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

func setupTestServer() (*httptest.Server, *session.InMemoryStore) {
	store := session.NewInMemoryStore()

	tmpDir, err := os.MkdirTemp("", "devo-test-*")
	if err != nil {
		panic(err)
	}
	memStore, err := memory.NewFileStore(tmpDir)
	if err != nil {
		panic(err)
	}
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)

	ag := agent.New(
		agent.Config{ID: "test", Name: "Test", Description: "Test agent", SystemPrompt: "", Tools: nil},
		store, nil, config.DefaultConfig(),
		approvalMgr, memManager, nil, nil, nil, nil,
	)
	registry := agent.NewRegistry(ag)

	handler := NewHandler(HandlerDeps{Store: store, AgentRegistry: registry, Version: "0.0.1"})

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return httptest.NewServer(mux), store
}

func setupTestServerWithTools() (*httptest.Server, *session.InMemoryStore, *agent.Agent) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	mockLLM := &approvalMockClient{}

	tmpDir, err := os.MkdirTemp("", "devo-test-*")
	if err != nil {
		panic(err)
	}
	memStore, err := memory.NewFileStore(tmpDir)
	if err != nil {
		panic(err)
	}
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)

	ag := agent.New(
		agent.Config{ID: "test", Name: "Test", Description: "Test agent", SystemPrompt: "", Tools: nil, LLMClient: mockLLM},
		store, toolRegistry, config.DefaultConfig(),
		approvalMgr, memManager, nil, nil, nil, nil,
	)
	registry := agent.NewRegistry(ag)

	handler := NewHandler(HandlerDeps{Store: store, AgentRegistry: registry, Version: "0.0.1"})

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return httptest.NewServer(mux), store, ag
}

type approvalMockClient struct {
	callCount int
}

func (m *approvalMockClient) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	m.callCount++

	lastMsg := messages[len(messages)-1]

	if lastMsg.Role == session.RoleTool {
		return &llmclient.CompleteResult{
			Text: "I received the tool result: " + lastMsg.Content,
		}, nil
	}

	if lastMsg.Role == session.RoleUser || lastMsg.Role == session.RoleSystem {
		if m.callCount == 1 {
			return &llmclient.CompleteResult{
				ToolCalls: []session.ToolCall{
					{
						ID:       "call-1",
						ToolName: "write_file",
						Params: map[string]interface{}{
							"path":    "test_approve.txt",
							"content": "Hello from approve test",
						},
					},
				},
			}, nil
		}
		return &llmclient.CompleteResult{
			Text: "Task completed.",
		}, nil
	}

	return &llmclient.CompleteResult{Text: "OK"}, nil
}

func (m *approvalMockClient) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	result, err := m.Complete(ctx, messages, systemPrompt)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}

	words := strings.Fields(result.Text)
	for i, word := range words {
		fullText := strings.Join(words[:i+1], " ")
		select {
		case <-ctx.Done():
			callback(llmclient.StreamEvent{Type: "error", Err: ctx.Err()})
			return ctx.Err()
		default:
		}
		callback(llmclient.StreamEvent{
			Type:     "token",
			Token:    word,
			FullText: fullText,
		})
		time.Sleep(1 * time.Millisecond)
	}

	callback(llmclient.StreamEvent{
		Type:         "done",
		FullText:     result.Text,
		ToolCalls:    result.ToolCalls,
		FinishReason: "stop",
		TokenUsage:   result.TokenUsage,
	})

	return nil
}

func (m *approvalMockClient) TestConnection(ctx context.Context) error { return nil }

func waitForEvent(ch chan session.Event, eventType string, timeout time.Duration) (*session.Event, bool) {
	timer := time.After(timeout)
	for {
		select {
		case <-timer:
			return nil, false
		case evt, ok := <-ch:
			if !ok {
				return nil, false
			}
			if eventType == "" || evt.Type == eventType {
				return &evt, true
			}
		}
	}
}

func doPut(t *testing.T, url string, body []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestGetAgents(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/api/v1/agents", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var agents []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(agents) == 0 {
		t.Fatal("expected at least one agent")
	}

	found := false
	for _, ag := range agents {
		if ag["id"] == "test" {
			found = true
			if ag["name"] != "Test" {
				t.Errorf("expected name 'Test', got %v", ag["name"])
			}
			if ag["description"] != "Test agent" {
				t.Errorf("expected description 'Test agent', got %v", ag["description"])
			}
		}
	}
	if !found {
		t.Error("expected to find agent with id 'test'")
	}
}

func TestSetTeamMode(t *testing.T) {
	store := session.NewInMemoryStore()

	tmpDir, err := os.MkdirTemp("", "devo-test-*")
	if err != nil {
		panic(err)
	}
	memStore, err := memory.NewFileStore(tmpDir)
	if err != nil {
		panic(err)
	}
	pathLock := concurrency.NewPathLockManager()
	approvalMgr := approval.NewManager()
	memManager := memory.NewManager(memStore, pathLock, approvalMgr)

	defaultAgent := agent.New(
		agent.Config{ID: "devo-default", Name: "Devo", Description: "Default", SystemPrompt: "", Tools: nil, Builtin: true},
		store, nil, config.DefaultConfig(),
		approvalMgr, memManager, nil, nil, nil, nil,
	)
	registry := agent.NewRegistry(defaultAgent)

	subAgent := agent.New(
		agent.Config{ID: "code-reviewer", Name: "Reviewer", Description: "Sub", SystemPrompt: "", Tools: nil, Builtin: true},
		store, nil, config.DefaultConfig(),
		approvalMgr, memManager, nil, nil, nil, nil,
	)
	registry.Register(subAgent)

	handler := NewHandler(HandlerDeps{Store: store, AgentRegistry: registry, Version: "0.0.1"})
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	body := strings.NewReader(`{"enabled": true}`)
	req, err := http.NewRequest("PUT", ts.URL+"/api/v1/config/team-mode", body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result["team_mode"] != true {
		t.Errorf("expected team_mode=true, got %v", result["team_mode"])
	}

	subAgents, ok := result["available_sub_agents"].([]interface{})
	if !ok {
		t.Fatal("expected available_sub_agents array")
	}
	if len(subAgents) == 0 {
		t.Error("expected at least one available sub-agent")
	}
	found := false
	for _, a := range subAgents {
		if s, ok := a.(string); ok && s == "code-reviewer" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'code-reviewer' in available_sub_agents")
	}
}
