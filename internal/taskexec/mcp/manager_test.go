package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"devo/internal/taskexec/tools"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type mockMCPServer struct {
	listener net.Listener
	server   *http.Server
	addr     string
	tools    []*sdkmcp.Tool
	mu       sync.Mutex
	reqCount atomic.Int64
}

func newMockMCPServer(t *testing.T) *mockMCPServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	s := &mockMCPServer{
		listener: listener,
		addr:     fmt.Sprintf("http://%s", listener.Addr().String()),
		tools: []*sdkmcp.Tool{
			{
				Name:        "mock_search",
				Description: "Search mock data",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
					},
					"required": []string{"query"},
				},
			},
			{
				Name:        "mock_fetch",
				Description: "Fetch mock data",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "URL to fetch",
						},
					},
					"required": []string{"url"},
				},
			},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleMessage)

	s.server = &http.Server{Handler: mux}

	go func() {
		_ = s.server.Serve(listener)
	}()

	time.Sleep(50 * time.Millisecond)

	return s
}

func (s *mockMCPServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleSSE(w, r)
		return
	}
	if r.Method == http.MethodDelete {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, 0, -32700, "Parse error")
		return
	}

	s.reqCount.Add(1)

	var result interface{}
	var errResp *jsonRPCError

	switch req.Method {
	case "initialize":
		result = sdkmcp.InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo: &sdkmcp.Implementation{
				Name:    "mock-server",
				Version: "1.0.0",
			},
			Capabilities: &sdkmcp.ServerCapabilities{
				Tools: &sdkmcp.ToolCapabilities{ListChanged: true},
			},
		}
	case "tools/list":
		s.mu.Lock()
		tools := s.tools
		s.mu.Unlock()
		result = sdkmcp.ListToolsResult{Tools: tools}
	case "tools/call":
		paramsBytes, _ := json.Marshal(req.Params)
		var callReq = struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}{}
		if err := json.Unmarshal(paramsBytes, &callReq); err == nil {
			result = sdkmcp.CallToolResult{
				Content: []sdkmcp.Content{
					&sdkmcp.TextContent{
						Text: fmt.Sprintf("Mock result for %s: %v", callReq.Name, callReq.Arguments),
					},
				},
			}
		} else {
			errResp = &jsonRPCError{Code: -32602, Message: "Invalid params"}
		}
	default:
		errResp = &jsonRPCError{Code: -32601, Message: "Method not found: " + req.Method}
	}

	if errResp != nil {
		writeJSONRPCError(w, req.ID, errResp.Code, errResp.Message)
		return
	}

	resultBytes, _ := json.Marshal(result)
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultBytes,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *mockMCPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", s.addr)
	flusher.Flush()

	ctx := r.Context()
	<-ctx.Done()
}

func (s *mockMCPServer) addTool(tool *sdkmcp.Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools = append(s.tools, tool)
}

func (s *mockMCPServer) close() {
	if s.server != nil {
		s.server.Close()
	}
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *mockMCPServer) requestCount() int64 {
	return s.reqCount.Load()
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeJSONRPCError(w http.ResponseWriter, id int64, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func TestManager_ConnectAndDiscoverTools(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)

	var discovered []McpTool
	mgr.SetToolDiscoveredCallback(func(tool McpTool) {
		discovered = append(discovered, tool)
	})

	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	info, ok := mgr.GetServerInfo("mock-server")
	if !ok {
		t.Fatal("expected server info to exist")
	}
	if info.Status != StatusConnected {
		t.Errorf("expected status connected, got %s", info.Status)
	}
	if len(info.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(info.Tools))
	}

	allTools := mgr.GetAllTools()
	if len(allTools) != 2 {
		t.Errorf("expected 2 tools from GetAllTools, got %d", len(allTools))
	}

	if len(discovered) != 2 {
		t.Errorf("expected 2 discovered tools, got %d", len(discovered))
	}
}

func TestManager_CallTool(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	result, err := mgr.CallTool(ctx, "mock_search", map[string]interface{}{
		"query": "test-query",
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if !strings.Contains(result, "mock_search") {
		t.Errorf("expected result to contain tool name, got: %s", result)
	}
	if !strings.Contains(result, "test-query") {
		t.Errorf("expected result to contain query, got: %s", result)
	}
}

func TestManager_GetTool(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	tool, ok := mgr.GetTool("mock_search")
	if !ok {
		t.Fatal("expected to find mock_search tool")
	}
	if tool.ServerID != "mock-server" {
		t.Errorf("expected server_id mock-server, got %s", tool.ServerID)
	}
	if tool.Description != "Search mock data" {
		t.Errorf("expected description 'Search mock data', got %s", tool.Description)
	}

	_, ok = mgr.GetTool("nonexistent_tool")
	if ok {
		t.Error("expected not to find nonexistent tool")
	}
}

func TestManager_DiscoverTools_NewTool(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)

	var discovered []McpTool
	mgr.SetToolDiscoveredCallback(func(tool McpTool) {
		discovered = append(discovered, tool)
	})

	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	discovered = nil

	mock.addTool(&sdkmcp.Tool{
		Name:        "mock_new_tool",
		Description: "A newly added tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param": map[string]interface{}{"type": "string"},
			},
		},
	})

	newTools, err := mgr.DiscoverTools(ctx, "mock-server")
	if err != nil {
		t.Fatalf("DiscoverTools failed: %v", err)
	}

	if len(newTools) != 3 {
		t.Errorf("expected 3 tools after discovery, got %d", len(newTools))
	}

	if len(discovered) != 1 {
		t.Errorf("expected 1 newly discovered tool, got %d", len(discovered))
	}
	if discovered[0].ToolName != "mock_new_tool" {
		t.Errorf("expected new tool 'mock_new_tool', got %s", discovered[0].ToolName)
	}
}

func TestManager_Disconnect(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}

	err = mgr.Disconnect("mock-server")
	if err != nil {
		t.Fatalf("Disconnect failed: %v", err)
	}

	info, ok := mgr.GetServerInfo("mock-server")
	if !ok {
		t.Fatal("server info should still exist after disconnect")
	}
	if info.Status != StatusDisconnected {
		t.Errorf("expected status disconnected, got %s", info.Status)
	}

	_, err = mgr.CallTool(ctx, "mock_search", map[string]interface{}{"query": "test"})
	if err == nil {
		t.Error("expected error when calling tool on disconnected server")
	}
}

func TestManager_ProjectPriorityOverGlobal(t *testing.T) {
	mock1 := newMockMCPServer(t)
	defer mock1.close()
	mock2 := newMockMCPServer(t)
	defer mock2.close()

	tmpDir := t.TempDir()
	homeDir := t.TempDir()

	writeMCPConfigWithPath(t, filepath.Join(tmpDir, ".devo", "mcp_servers.json"), mock1.addr, "shared-server")
	writeMCPConfigWithPath(t, filepath.Join(homeDir, ".devo", "mcp_servers.json"), mock2.addr, "shared-server")

	setHomeDir(t, homeDir)

	ctx := context.Background()
	mgr := NewManager(tmpDir)

	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	info, ok := mgr.GetServerInfo("shared-server")
	if !ok {
		t.Fatal("expected server info to exist")
	}

	if info.Config.Source != SourceProject {
		t.Errorf("expected project config to take priority, got source=%s", info.Config.Source)
	}
}

func TestManager_GetAllServerInfos(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	infos := mgr.GetAllServerInfos()
	if len(infos) != 1 {
		t.Errorf("expected 1 server info, got %d", len(infos))
	}
	if infos[0].Config.ServerID != "mock-server" {
		t.Errorf("expected server_id mock-server, got %s", infos[0].Config.ServerID)
	}
}

func TestManager_ConnectNonExistentServer(t *testing.T) {
	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, "http://127.0.0.1:19999", "bad-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll returned error: %v", err)
	}
	defer mgr.Shutdown(ctx)

	info, ok := mgr.GetServerInfo("bad-server")
	if !ok {
		t.Fatal("expected server info to exist")
	}
	if info.Status != StatusError {
		t.Errorf("expected status error for unreachable server, got %s", info.Status)
	}
}

func TestManager_Shutdown(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}

	err = mgr.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	infos := mgr.GetAllServerInfos()
	for _, info := range infos {
		if info.Status != StatusDisconnected {
			t.Errorf("expected status disconnected after shutdown, got %s", info.Status)
		}
	}
}

func TestMcpToolAdapter_ImplementsTool(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	adapter := &mcpToolAdapter{
		manager:  mgr,
		serverID: "mock-server",
		toolName: "mock_search",
	}

	var _ tools.Tool = adapter

	expectedName := "mcp_mock-server_mock_search"
	if adapter.Name() != expectedName {
		t.Errorf("expected name '%s', got '%s'", expectedName, adapter.Name())
	}
	if adapter.Description() != "Search mock data" {
		t.Errorf("expected description 'Search mock data', got %s", adapter.Description())
	}
	if adapter.RiskLevel() != tools.RiskLevelMedium {
		t.Errorf("expected risk level medium, got %s", adapter.RiskLevel())
	}

	schema := adapter.ParamsSchema()
	if schema == nil {
		t.Fatal("expected non-nil params schema")
	}

	result, err := executeMCPTool(t, adapter, tmpDir, map[string]interface{}{
		"query": "hello",
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, "mock_search") {
		t.Errorf("expected result to contain tool name, got: %s", result)
	}
}

func TestManager_CallTool_UnknownTool(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	_, err = mgr.CallTool(ctx, "nonexistent_tool", nil)
	if err == nil {
		t.Error("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("project_config_only", func(t *testing.T) {
		tmpDir := t.TempDir()
		setHomeDir(t, t.TempDir())
		writeMCPConfig(t, tmpDir, "http://localhost:8080", "test-server")

		configs, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		if len(configs) != 1 {
			t.Errorf("expected 1 config, got %d", len(configs))
		}
		if configs[0].ServerID != "test-server" {
			t.Errorf("expected server_id 'test-server', got %s", configs[0].ServerID)
		}
		if configs[0].Source != SourceProject {
			t.Errorf("expected source 'project', got %s", configs[0].Source)
		}
	})

	t.Run("no_config_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		setHomeDir(t, t.TempDir())
		configs, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		if len(configs) != 0 {
			t.Errorf("expected 0 configs, got %d", len(configs))
		}
	})

	t.Run("default_transport", func(t *testing.T) {
		tmpDir := t.TempDir()
		setHomeDir(t, t.TempDir())

		configPath := filepath.Join(tmpDir, ".devo", "mcp_servers.json")
		os.MkdirAll(filepath.Dir(configPath), 0755)
		configData := []map[string]interface{}{
			{
				"server_id": "no-transport-server",
				"endpoint":  "http://localhost:9999",
			},
		}
		data, _ := json.Marshal(configData)
		os.WriteFile(configPath, data, 0644)

		configs, err := LoadConfig(tmpDir)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		if len(configs) != 1 {
			t.Fatalf("expected 1 config, got %d", len(configs))
		}
		if configs[0].Transport != "sse" {
			t.Errorf("expected default transport 'sse', got %s", configs[0].Transport)
		}
	})
}

func TestSplitCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"npx -y @scope/package", []string{"npx", "-y", "@scope/package"}},
		{"python script.py", []string{"python", "script.py"}},
		{`node "path with spaces/app.js"`, []string{"node", "path with spaces/app.js"}},
		{"simple", []string{"simple"}},
		{"", nil},
	}

	for _, tt := range tests {
		result := splitCommand(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitCommand(%q): expected %v, got %v", tt.input, tt.expected, result)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitCommand(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], result[i])
			}
		}
	}
}

func TestManager_CallTool_Error(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	_, err = mgr.CallTool(ctx, "mock_search", nil)
	if err != nil {
		t.Fatalf("CallTool should not fail with nil params: %v", err)
	}
}

func writeMCPConfig(t *testing.T, tmpDir, addr, serverID string) {
	t.Helper()
	writeMCPConfigWithPath(t, filepath.Join(tmpDir, ".devo", "mcp_servers.json"), addr, serverID)
}

func writeMCPConfigs(t *testing.T, tmpDir string, servers ...struct {
	addr     string
	serverID string
}) {
	t.Helper()
	configs := make([]map[string]interface{}, 0, len(servers))
	for _, s := range servers {
		configs = append(configs, map[string]interface{}{
			"server_id": s.serverID,
			"endpoint":  s.addr,
			"transport": "sse",
		})
	}
	data, err := json.Marshal(configs)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	configPath := filepath.Join(tmpDir, ".devo", "mcp_servers.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func setHomeDir(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func writeMCPConfigWithPath(t *testing.T, configPath, addr, serverID string) {
	t.Helper()

	os.MkdirAll(filepath.Dir(configPath), 0755)

	configs := []map[string]interface{}{
		{
			"server_id": serverID,
			"endpoint":  addr,
			"transport": "sse",
		},
	}
	data, err := json.Marshal(configs)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestMcpToolName(t *testing.T) {
	result := mcpToolName("filesystem", "read")
	expected := "mcp_filesystem_read"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestGetToolByServer(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	tool, ok := mgr.GetToolByServer("mock-server", "mock_search")
	if !ok {
		t.Fatal("expected to find mock_search tool")
	}
	if tool.ServerID != "mock-server" {
		t.Errorf("expected server_id mock-server, got %s", tool.ServerID)
	}
	if tool.Description != "Search mock data" {
		t.Errorf("expected description 'Search mock data', got %s", tool.Description)
	}

	_, ok = mgr.GetToolByServer("mock-server", "nonexistent")
	if ok {
		t.Error("expected not to find nonexistent tool")
	}

	_, ok = mgr.GetToolByServer("nonexistent-server", "mock_search")
	if ok {
		t.Error("expected not to find tool on nonexistent server")
	}
}

func TestCallToolByServer(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	result, err := mgr.CallToolByServer(ctx, "mock-server", "mock_search", map[string]interface{}{
		"query": "test-query",
	})
	if err != nil {
		t.Fatalf("CallToolByServer failed: %v", err)
	}
	if !strings.Contains(result, "mock_search") {
		t.Errorf("expected result to contain tool name, got: %s", result)
	}
	if !strings.Contains(result, "test-query") {
		t.Errorf("expected result to contain query, got: %s", result)
	}

	_, err = mgr.CallToolByServer(ctx, "mock-server", "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}

	_, err = mgr.CallToolByServer(ctx, "nonexistent-server", "mock_search", nil)
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

func TestMcpToolAdapter_NameUniqueness(t *testing.T) {
	mock1 := newMockMCPServer(t)
	defer mock1.close()
	mock2 := newMockMCPServer(t)
	defer mock2.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfigs(t, tmpDir,
		struct {
			addr     string
			serverID string
		}{mock1.addr, "server-a"},
		struct {
			addr     string
			serverID string
		}{mock2.addr, "server-b"},
	)

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	registry := tools.NewRegistry()
	mgr.RegisterTools(registry)

	tools := registry.ListTools()
	if len(tools) != 4 {
		t.Fatalf("expected 4 tools (2 from each server), got %d", len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		name := tool.Name()
		if toolNames[name] {
			t.Errorf("duplicate tool name: %s", name)
		}
		toolNames[name] = true
	}

	adapterA := &mcpToolAdapter{
		manager:  mgr,
		serverID: "server-a",
		toolName: "mock_search",
	}
	adapterB := &mcpToolAdapter{
		manager:  mgr,
		serverID: "server-b",
		toolName: "mock_search",
	}

	nameA := adapterA.Name()
	nameB := adapterB.Name()

	if nameA == nameB {
		t.Errorf("tool names should be different: %s == %s", nameA, nameB)
	}

	expectedA := "mcp_server-a_mock_search"
	expectedB := "mcp_server-b_mock_search"
	if nameA != expectedA {
		t.Errorf("expected name '%s', got '%s'", expectedA, nameA)
	}
	if nameB != expectedB {
		t.Errorf("expected name '%s', got '%s'", expectedB, nameB)
	}
}

func TestMcpToolAdapter_ExecuteUsesServerRouting(t *testing.T) {
	mock1 := newMockMCPServer(t)
	defer mock1.close()
	mock2 := newMockMCPServer(t)
	defer mock2.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfigs(t, tmpDir,
		struct {
			addr     string
			serverID string
		}{mock1.addr, "server-a"},
		struct {
			addr     string
			serverID string
		}{mock2.addr, "server-b"},
	)

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	adapterA := &mcpToolAdapter{
		manager:  mgr,
		serverID: "server-a",
		toolName: "mock_search",
	}
	adapterB := &mcpToolAdapter{
		manager:  mgr,
		serverID: "server-b",
		toolName: "mock_search",
	}

	resultA, err := executeMCPTool(t, adapterA, tmpDir, map[string]interface{}{
		"query": "from-server-a",
	})
	if err != nil {
		t.Fatalf("adapterA.Execute failed: %v", err)
	}

	resultB, err := executeMCPTool(t, adapterB, tmpDir, map[string]interface{}{
		"query": "from-server-b",
	})
	if err != nil {
		t.Fatalf("adapterB.Execute failed: %v", err)
	}

	if !strings.Contains(resultA, "from-server-a") {
		t.Errorf("resultA should contain 'from-server-a', got: %s", resultA)
	}
	if !strings.Contains(resultB, "from-server-b") {
		t.Errorf("resultB should contain 'from-server-b', got: %s", resultB)
	}
}

func TestRegisterTools_PrefixedNames(t *testing.T) {
	mock := newMockMCPServer(t)
	defer mock.close()

	tmpDir := t.TempDir()
	setHomeDir(t, t.TempDir())
	writeMCPConfig(t, tmpDir, mock.addr, "mock-server")

	ctx := context.Background()
	mgr := NewManager(tmpDir)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	registry := tools.NewRegistry()
	mgr.RegisterTools(registry)

	tools := registry.ListTools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	expectedNames := map[string]bool{
		"mcp_mock-server_mock_search": true,
		"mcp_mock-server_mock_fetch":  true,
	}
	for _, tool := range tools {
		name := tool.Name()
		if !expectedNames[name] {
			t.Errorf("unexpected tool name: %s", name)
		}
	}
}

func TestSetProjectDir_WorkspaceIsolation(t *testing.T) {
	mockA := newMockMCPServer(t)
	defer mockA.close()
	mockB := newMockMCPServer(t)
	defer mockB.close()

	workdirA := t.TempDir()
	workdirB := t.TempDir()
	homeDir := t.TempDir()
	setHomeDir(t, homeDir)

	writeMCPConfig(t, workdirA, mockA.addr, "server-a")
	writeMCPConfig(t, workdirB, mockB.addr, "server-b")

	globalConfigPath := filepath.Join(homeDir, ".devo", "mcp_servers.json")
	os.MkdirAll(filepath.Dir(globalConfigPath), 0755)
	globalData := []map[string]interface{}{
		{
			"server_id": "global-server",
			"endpoint":  mockA.addr,
			"transport": "streamable",
		},
	}
	globalBytes, _ := json.Marshal(globalData)
	os.WriteFile(globalConfigPath, globalBytes, 0644)

	ctx := context.Background()

	mgr := NewManager(workdirA)
	err := mgr.ConnectAll(ctx)
	if err != nil {
		t.Fatalf("ConnectAll for workdirA failed: %v", err)
	}
	defer mgr.Shutdown(ctx)

	servers := mgr.GetAllServerInfos()
	serverIDs := make(map[string]bool)
	for _, s := range servers {
		serverIDs[s.Config.ServerID] = true
	}
	if !serverIDs["server-a"] {
		t.Error("expected server-a from workdirA")
	}
	if !serverIDs["global-server"] {
		t.Error("expected global-server")
	}
	if serverIDs["server-b"] {
		t.Error("should NOT have server-b from workdirB yet")
	}

	err = mgr.SetProjectDir(workdirB)
	if err != nil {
		t.Fatalf("SetProjectDir to workdirB failed: %v", err)
	}

	servers = mgr.GetAllServerInfos()
	serverIDs = make(map[string]bool)
	for _, s := range servers {
		serverIDs[s.Config.ServerID] = true
	}
	if serverIDs["server-a"] {
		t.Error("server-a should be gone after switching from workdirA")
	}
	if !serverIDs["server-b"] {
		t.Error("expected server-b from workdirB")
	}
	if !serverIDs["global-server"] {
		t.Error("expected global-server to persist across workspace switch")
	}
}

func executeMCPTool(t *testing.T, adapter *mcpToolAdapter, workingDir string, params map[string]interface{}) (string, error) {
	t.Helper()
	ctx := context.Background()
	ch := make(chan tools.StreamEvent, 256)
	sw := tools.NewChannelStreamWriter(ch)

	go func() {
		defer close(ch)
		if err := adapter.Execute(ctx, workingDir, params, sw); err != nil {
			sw.WriteError(err)
		}
	}()

	result := tools.CollectToolResult(ch)
	if !result.Success {
		return result.Content, fmt.Errorf("%s", result.Error)
	}
	return result.Content, nil
}
