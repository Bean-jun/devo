package mcp

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"sync"
	"time"

	projectconfig "devo/internal/core/config"
	"devo/internal/core/session"
	"devo/internal/taskexec/tools"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ToolDiscoveredCallback func(tool McpTool)

type Manager struct {
	mu              sync.RWMutex
	servers         map[string]*ServerInfo
	configs         map[string]McpServerConfig
	sessions        map[string]*sdkmcp.ClientSession
	clients         map[string]*sdkmcp.Client
	discoveryCB     ToolDiscoveredCallback
	globalEventBus  *session.EventBus
	workingDir      string
	reconnectDelay  time.Duration
	reconnectCtx    context.Context
	reconnectCancel context.CancelFunc
	reconnectWG     sync.WaitGroup
	toolRegistry    *tools.Registry
}

func NewManager(workingDir string) *Manager {
	return &Manager{
		servers:        make(map[string]*ServerInfo),
		configs:        make(map[string]McpServerConfig),
		sessions:       make(map[string]*sdkmcp.ClientSession),
		clients:        make(map[string]*sdkmcp.Client),
		workingDir:     workingDir,
		reconnectDelay: 5 * time.Second,
		globalEventBus: session.NewEventBus(session.DefaultEventHistorySize),
	}
}

func (m *Manager) SetToolDiscoveredCallback(cb ToolDiscoveredCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.discoveryCB = cb
}

func (m *Manager) GlobalEventBus() *session.EventBus {
	return m.globalEventBus
}

func (m *Manager) ConnectAll(ctx context.Context) error {
	configs, err := LoadConfig(m.workingDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	m.mu.Lock()
	for _, cfg := range configs {
		m.configs[cfg.ServerID] = cfg
	}
	m.mu.Unlock()

	mcpWhitelist, hasConfig := m.loadMCPWhitelist()

	m.reconnectCtx, m.reconnectCancel = context.WithCancel(context.Background())

	for _, cfg := range configs {
		if hasConfig && !mcpWhitelist[cfg.ServerID] {
			m.mu.Lock()
			m.servers[cfg.ServerID] = &ServerInfo{
				Config: cfg,
				Status: StatusDisconnected,
			}
			m.mu.Unlock()
			continue
		}

		if err := m.Connect(ctx, cfg.ServerID); err != nil {
			m.mu.Lock()
			if _, ok := m.servers[cfg.ServerID]; !ok {
				m.servers[cfg.ServerID] = &ServerInfo{
					Config:   cfg,
					Status:   StatusError,
					ErrorMsg: err.Error(),
				}
			}
			m.mu.Unlock()
			m.startReconnect(cfg.ServerID)
		}
	}

	return nil
}

func (m *Manager) loadMCPWhitelist() (map[string]bool, bool) {
	cfg, err := projectconfig.Load(m.workingDir)
	if err != nil || cfg == nil {
		return nil, false
	}
	whitelist := make(map[string]bool, len(cfg.MCP))
	for _, id := range cfg.MCP {
		whitelist[id] = true
	}
	return whitelist, true
}

func (m *Manager) Connect(ctx context.Context, serverID string) error {
	m.mu.RLock()
	cfg, exists := m.configs[serverID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("server %s not configured", serverID)
	}

	client := sdkmcp.NewClient(&sdkmcp.Implementation{
		Name:    "devo",
		Version: "1.0.0",
	}, nil)

	var transport sdkmcp.Transport
	switch cfg.Transport {
	case "stdio":
		parts := splitCommand(cfg.Endpoint)
		if len(parts) == 0 {
			return fmt.Errorf("invalid stdio command: %s", cfg.Endpoint)
		}
		transport = &sdkmcp.CommandTransport{
			Command: exec.Command(parts[0], parts[1:]...),
		}
	default:
		transport = &sdkmcp.StreamableClientTransport{
			Endpoint: cfg.Endpoint,
		}
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		session.Close()
		return fmt.Errorf("list tools: %w", err)
	}

	mcpTools := make([]McpTool, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		mcpTools = append(mcpTools, McpTool{
			ToolName:    t.Name,
			ServerID:    serverID,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	m.mu.Lock()
	m.clients[serverID] = client
	m.sessions[serverID] = session
	m.servers[serverID] = &ServerInfo{
		Config:   cfg,
		Status:   StatusConnected,
		Tools:    mcpTools,
		ErrorMsg: "",
	}
	m.mu.Unlock()

	if m.discoveryCB != nil {
		for _, t := range mcpTools {
			m.discoveryCB(t)
		}
	}

	for _, t := range mcpTools {
		m.globalEventBus.Publish("mcp_tool_discovered", map[string]interface{}{
			"tool_name": t.ToolName,
			"server_id": t.ServerID,
		})
	}

	return nil
}

func (m *Manager) Disconnect(serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[serverID]
	if !exists {
		return fmt.Errorf("server %s not connected", serverID)
	}

	if err := session.Close(); err != nil {
		return fmt.Errorf("close session: %w", err)
	}

	delete(m.sessions, serverID)
	delete(m.clients, serverID)
	if info, ok := m.servers[serverID]; ok {
		info.Status = StatusDisconnected
	}

	return nil
}

func (m *Manager) GetServerInfo(serverID string) (*ServerInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.servers[serverID]
	return info, ok
}

func (m *Manager) GetAllServerInfos() []ServerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ServerInfo, 0, len(m.servers))
	for _, info := range m.servers {
		result = append(result, *info)
	}
	return result
}

func (m *Manager) GetAllServerConfigs() []McpServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]McpServerConfig, 0, len(m.configs))
	for _, cfg := range m.configs {
		result = append(result, cfg)
	}
	return result
}

func (m *Manager) EnableServer(ctx context.Context, serverID string) error {
	m.mu.Lock()

	if m.workingDir == "" {
		m.mu.Unlock()
		return fmt.Errorf("no working directory set")
	}

	cfg, err := projectconfig.Load(m.workingDir)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	if cfg == nil {
		cfg = &projectconfig.ProjectConfig{}
	}

	found := false
	for _, id := range cfg.MCP {
		if id == serverID {
			found = true
			break
		}
	}
	if !found {
		cfg.MCP = append(cfg.MCP, serverID)
	}

	if err := projectconfig.Save(m.workingDir, cfg); err != nil {
		m.mu.Unlock()
		return err
	}

	m.mu.Unlock()

	if err := m.Connect(ctx, serverID); err != nil {
		return err
	}

	m.registerServerTools(serverID)
	return nil
}

func (m *Manager) DisableServer(serverID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.workingDir == "" {
		return fmt.Errorf("no working directory set")
	}

	cfg, err := projectconfig.Load(m.workingDir)
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &projectconfig.ProjectConfig{}
	}

	filtered := make([]string, 0, len(cfg.MCP))
	for _, id := range cfg.MCP {
		if id != serverID {
			filtered = append(filtered, id)
		}
	}
	cfg.MCP = filtered

	if err := projectconfig.Save(m.workingDir, cfg); err != nil {
		return err
	}

	if info, ok := m.servers[serverID]; ok {
		for _, tool := range info.Tools {
			if m.toolRegistry != nil {
				m.toolRegistry.Unregister(tool.ToolName)
			}
		}
		info.Status = StatusDisconnected
		info.Tools = nil
	}

	if session, ok := m.sessions[serverID]; ok {
		session.Close()
		delete(m.sessions, serverID)
	}
	if _, ok := m.clients[serverID]; ok {
		delete(m.clients, serverID)
	}

	return nil
}

func (m *Manager) registerServerTools(serverID string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.toolRegistry == nil {
		return
	}

	info, ok := m.servers[serverID]
	if !ok || info.Status != StatusConnected {
		return
	}

	for _, tool := range info.Tools {
		adapter := &mcpToolAdapter{
			manager:  m,
			toolName: tool.ToolName,
		}
		m.toolRegistry.Register(adapter)
	}
}

func (m *Manager) RemoveServer(serverID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, ok := m.servers[serverID]; ok {
		for _, tool := range info.Tools {
			if m.toolRegistry != nil {
				m.toolRegistry.Unregister(tool.ToolName)
			}
		}
	}

	delete(m.configs, serverID)
	delete(m.servers, serverID)

	if session, ok := m.sessions[serverID]; ok {
		session.Close()
		delete(m.sessions, serverID)
	}
	if _, ok := m.clients[serverID]; ok {
		delete(m.clients, serverID)
	}
}

func (m *Manager) GetAllTools() []McpTool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []McpTool
	for _, info := range m.servers {
		if info.Status == StatusConnected {
			result = append(result, info.Tools...)
		}
	}
	return result
}

func (m *Manager) GetTool(toolName string) (*McpTool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, info := range m.servers {
		if info.Status != StatusConnected {
			continue
		}
		for _, t := range info.Tools {
			if t.ToolName == toolName {
				return &t, true
			}
		}
	}
	return nil, false
}

func (m *Manager) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	mcpTool, ok := m.GetTool(toolName)
	if !ok {
		return "", fmt.Errorf("MCP tool not found: %s", toolName)
	}

	m.mu.RLock()
	session, exists := m.sessions[mcpTool.ServerID]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("server %s not connected", mcpTool.ServerID)
	}

	result, err := session.CallTool(ctx, &sdkmcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		m.markServerError(mcpTool.ServerID, err.Error())
		m.startReconnect(mcpTool.ServerID)
		return "", fmt.Errorf("call tool %s: %w", toolName, err)
	}

	if result.IsError {
		var errText string
		for _, c := range result.Content {
			if tc, ok := c.(*sdkmcp.TextContent); ok {
				errText += tc.Text
			}
		}
		if errText == "" {
			errText = "MCP tool returned an error"
		}
		return errText, nil
	}

	var textContent string
	for _, c := range result.Content {
		if tc, ok := c.(*sdkmcp.TextContent); ok {
			textContent += tc.Text
		}
	}

	return textContent, nil
}

func (m *Manager) DiscoverTools(ctx context.Context, serverID string) ([]McpTool, error) {
	m.mu.RLock()
	session, exists := m.sessions[serverID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("server %s not connected", serverID)
	}

	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	mcpTools := make([]McpTool, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		mcpTools = append(mcpTools, McpTool{
			ToolName:    t.Name,
			ServerID:    serverID,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	m.mu.Lock()
	if info, ok := m.servers[serverID]; ok {
		oldToolNames := make(map[string]bool)
		for _, t := range info.Tools {
			oldToolNames[t.ToolName] = true
		}

		info.Tools = mcpTools

		if m.discoveryCB != nil {
			for _, nt := range mcpTools {
				if !oldToolNames[nt.ToolName] {
					m.discoveryCB(nt)
				}
			}
		}

		for _, nt := range mcpTools {
			if !oldToolNames[nt.ToolName] {
				m.globalEventBus.Publish("mcp_tool_discovered", map[string]interface{}{
					"tool_name": nt.ToolName,
					"server_id": nt.ServerID,
				})
			}
		}
	}
	m.mu.Unlock()

	return mcpTools, nil
}

func (m *Manager) RegisterTools(registry *tools.Registry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.toolRegistry = registry

	for _, info := range m.servers {
		if info.Status != StatusConnected {
			continue
		}
		for _, tool := range info.Tools {
			adapter := &mcpToolAdapter{
				manager:  m,
				toolName: tool.ToolName,
			}
			registry.Register(adapter)
		}
	}
}

func (m *Manager) Shutdown(ctx context.Context) error {
	if m.reconnectCancel != nil {
		m.reconnectCancel()
	}
	m.reconnectWG.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	for serverID, session := range m.sessions {
		session.Close()
		delete(m.sessions, serverID)
		delete(m.clients, serverID)
		if info, ok := m.servers[serverID]; ok {
			info.Status = StatusDisconnected
		}
	}
	return nil
}

func (m *Manager) markServerError(serverID, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if info, ok := m.servers[serverID]; ok {
		info.Status = StatusError
		info.ErrorMsg = errMsg
	}
}

func (m *Manager) startReconnect(serverID string) {
	m.mu.RLock()
	_, exists := m.configs[serverID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	m.reconnectWG.Add(1)
	go func() {
		defer m.reconnectWG.Done()

		for {
			select {
			case <-m.reconnectCtx.Done():
				return
			case <-time.After(m.reconnectDelay):
			}

			m.mu.RLock()
			info, ok := m.servers[serverID]
			m.mu.RUnlock()

			if !ok || info.Status == StatusConnected {
				return
			}

			log.Printf("[mcp] attempting reconnect to server %s...", serverID)
			ctx, cancel := context.WithTimeout(m.reconnectCtx, 10*time.Second)
			if err := m.Connect(ctx, serverID); err != nil {
				log.Printf("[mcp] reconnect failed for %s: %v", serverID, err)
				cancel()
				continue
			}
			cancel()
			log.Printf("[mcp] reconnect succeeded for %s", serverID)
			m.registerServerTools(serverID)
			return
		}
	}()
}

func splitCommand(cmd string) []string {
	var parts []string
	var current string
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(cmd); i++ {
		ch := cmd[i]
		switch {
		case ch == '"' || ch == '\'':
			if inQuote && ch == quoteChar {
				inQuote = false
			} else if !inQuote {
				inQuote = true
				quoteChar = ch
			} else {
				current += string(ch)
			}
		case ch == ' ' && !inQuote:
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

var _ tools.Tool = (*mcpToolAdapter)(nil)

type mcpToolAdapter struct {
	manager  *Manager
	toolName string
}

func (a *mcpToolAdapter) Name() string {
	return a.toolName
}

func (a *mcpToolAdapter) Description() string {
	t, ok := a.manager.GetTool(a.toolName)
	if !ok {
		return ""
	}
	return t.Description
}

func (a *mcpToolAdapter) RiskLevel() tools.RiskLevel {
	return tools.RiskLevelMedium
}

func (a *mcpToolAdapter) ParamsSchema() map[string]interface{} {
	t, ok := a.manager.GetTool(a.toolName)
	if !ok {
		return nil
	}
	if schema, ok := t.InputSchema.(map[string]interface{}); ok {
		return schema
	}
	return nil
}

func (a *mcpToolAdapter) Execute(workingDir string, params map[string]interface{}) (string, error) {
	ctx := context.Background()
	return a.manager.CallTool(ctx, a.toolName, params)
}
