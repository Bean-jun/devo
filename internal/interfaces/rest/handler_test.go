package rest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"devo/internal/core/agentloop"
	"devo/internal/core/approval"
	"devo/internal/core/concurrency"
	"devo/internal/core/memory"
	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
)

func setupTestServer() (*httptest.Server, *session.InMemoryStore) {
	store := session.NewInMemoryStore()
	llm := llmclient.NewMockClient()
	loop := agentloop.New(store, llm)
	// Use temp dir for test
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
	loop.SetMemoryManager(memManager)
	handler := NewHandler(store, loop, memManager, "0.0.1")

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
	// Use temp dir for test
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
	loop.SetMemoryManager(memManager)
	handler := NewHandler(store, loop, memManager, "0.0.1")

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
