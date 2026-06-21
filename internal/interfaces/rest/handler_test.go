package rest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"devo/internal/core/agentloop"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

func setupTestServer() (*httptest.Server, *session.InMemoryStore) {
	store := session.NewInMemoryStore()
	llm := llmclient.NewMockClient()
	loop := agentloop.New(store, llm)
	handler := NewHandler(store, loop, "0.0.1")

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return httptest.NewServer(mux), store
}

func setupTestServerWithTools() (*httptest.Server, *session.InMemoryStore, *agentloop.Loop) {
	store := session.NewInMemoryStore()

	toolRegistry := tools.NewRegistry()
	toolRegistry.Register(&tools.WriteFileTool{})

	llm := &approvalMockClient{}
	loop := agentloop.NewWithTools(store, llm, toolRegistry)
	handler := NewHandler(store, loop, "0.0.1")

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return httptest.NewServer(mux), store, loop
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
