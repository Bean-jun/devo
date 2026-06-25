package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"devo/internal/taskexec/mcp"
)

func TestGetMcpTools_NoManager(t *testing.T) {
	ts, _ := setupTestServer()
	defer ts.Close()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/mcp/tools", ts.URL))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result mcpToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Tools == nil {
		t.Error("expected tools to be non-nil (empty array)")
	}

	if len(result.Tools) != 0 {
		t.Errorf("expected 0 tools without manager, got %d", len(result.Tools))
	}
}

func TestGetMcpTools_WithManager(t *testing.T) {
	ts, store := setupTestServer()
	defer ts.Close()

	wd := t.TempDir()
	mcpManager := mcp.NewManager(wd)
	handler := NewHandler(store, nil, nil, "0.0.1")
	handler.SetMcpManager(mcpManager)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts2 := httptest.NewServer(mux)
	defer ts2.Close()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/mcp/tools", ts2.URL))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result mcpToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Tools == nil {
		t.Error("expected tools to be non-nil")
	}
}

func TestGetMcpTools_TrustNote(t *testing.T) {
	ts, store := setupTestServer()
	defer ts.Close()

	wd := t.TempDir()
	mcpManager := mcp.NewManager(wd)
	handler := NewHandler(store, nil, nil, "0.0.1")
	handler.SetMcpManager(mcpManager)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	ts2 := httptest.NewServer(mux)
	defer ts2.Close()

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/mcp/tools", ts2.URL))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var result mcpToolsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	expectedTrustNote := "MCP 工具对文件系统的访问不受 Devo 直接控制，请仅连接可信的本地/私有 MCP 服务器"
	if len(result.Tools) > 0 {
		tool := result.Tools[0]
		if tool.TrustNote != expectedTrustNote {
			t.Errorf("expected trust_note=%q, got %q", expectedTrustNote, tool.TrustNote)
		}
	}
}
