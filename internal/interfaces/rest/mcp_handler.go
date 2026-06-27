package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strings"

	"devo/internal/taskexec/mcp"
)

type mcpToolsResponse struct {
	Tools []mcpToolItem `json:"tools"`
}

type mcpToolItem struct {
	ToolName    string      `json:"tool_name"`
	ServerID    string      `json:"server_id"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
	TrustNote   string      `json:"trust_note"`
}

func (h *Handler) GetMcpTools(w http.ResponseWriter, r *http.Request) {
	if h.mcpManager == nil {
		writeJSON(w, http.StatusOK, mcpToolsResponse{Tools: []mcpToolItem{}})
		return
	}

	tools := h.mcpManager.GetAllTools()
	items := make([]mcpToolItem, 0, len(tools))
	for _, t := range tools {
		items = append(items, mcpToolItem{
			ToolName:    t.ToolName,
			ServerID:    t.ServerID,
			Description: t.Description,
			InputSchema: t.InputSchema,
			TrustNote:   "MCP 工具对文件系统的访问不受 Devo 直接控制，请仅连接可信的本地/私有 MCP 服务器",
		})
	}

	writeJSON(w, http.StatusOK, mcpToolsResponse{Tools: items})
}

type mcpServerItem struct {
	ServerID  string        `json:"server_id"`
	Source    string        `json:"source"`
	Endpoint  string        `json:"endpoint"`
	Transport string        `json:"transport"`
	Status    string        `json:"status"`
	ToolCount int           `json:"tool_count"`
	Tools     []mcpToolItem `json:"tools"`
	ErrorMsg  string        `json:"error_msg,omitempty"`
}

type mcpServersResponse struct {
	Servers []mcpServerItem `json:"servers"`
}

func (h *Handler) GetMcpServers(w http.ResponseWriter, r *http.Request) {
	if h.mcpManager == nil {
		writeJSON(w, http.StatusOK, mcpServersResponse{Servers: []mcpServerItem{}})
		return
	}

	infos := h.mcpManager.GetAllServerInfos()
	servers := make([]mcpServerItem, 0, len(infos))
	for _, info := range infos {
		tools := make([]mcpToolItem, 0, len(info.Tools))
		for _, t := range info.Tools {
			tools = append(tools, mcpToolItem{
				ToolName:    t.ToolName,
				ServerID:    t.ServerID,
				Description: t.Description,
				InputSchema: t.InputSchema,
			})
		}
		servers = append(servers, mcpServerItem{
			ServerID:  info.Config.ServerID,
			Source:    info.Config.Source,
			Endpoint:  info.Config.Endpoint,
			Transport: info.Config.Transport,
			Status:    string(info.Status),
			ToolCount: len(info.Tools),
			Tools:     tools,
			ErrorMsg:  info.ErrorMsg,
		})
	}

	sort.Slice(servers, func(i, j int) bool {
		connectedI := servers[i].Status == "connected"
		connectedJ := servers[j].Status == "connected"
		if connectedI != connectedJ {
			return connectedI
		}
		return strings.ToLower(servers[i].ServerID) < strings.ToLower(servers[j].ServerID)
	})

	writeJSON(w, http.StatusOK, mcpServersResponse{Servers: servers})
}

type toggleServerRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *Handler) ToggleMcpServer(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server id is required")
		return
	}

	if h.mcpManager == nil {
		writeError(w, http.StatusInternalServerError, "mcp manager not available")
		return
	}

	var req toggleServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Enabled {
		if err := h.mcpManager.EnableServer(r.Context(), serverID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to enable server: "+err.Error())
			return
		}
	} else {
		if err := h.mcpManager.DisableServer(serverID); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to disable server: "+err.Error())
			return
		}
	}

	info, _ := h.mcpManager.GetServerInfo(serverID)
	status := string(mcp.StatusDisconnected)
	if info != nil {
		status = string(info.Status)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"server_id": serverID,
		"status":    status,
	})
}

type addServerRequest struct {
	ServerID  string `json:"server_id"`
	Endpoint  string `json:"endpoint"`
	Transport string `json:"transport"`
	Scope     string `json:"scope"`
}

func (h *Handler) AddMcpServer(w http.ResponseWriter, r *http.Request) {
	if h.mcpManager == nil {
		writeError(w, http.StatusInternalServerError, "mcp manager not available")
		return
	}

	var req addServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "server_id is required")
		return
	}
	if req.Endpoint == "" {
		writeError(w, http.StatusBadRequest, "endpoint is required")
		return
	}
	if req.Scope != "global" {
		req.Scope = "project"
	}

	if err := mcp.AddServerConfig(h.projectDir, req.Scope, req.ServerID, req.Endpoint, req.Transport); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.mcpManager.ConnectAll(r.Context()); err != nil {
		log.Printf("[devo] MCP reconnect after add warning: %v", err)
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"server_id": req.ServerID,
		"status":    "added",
	})
}

func (h *Handler) RemoveMcpServer(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	if serverID == "" {
		writeError(w, http.StatusBadRequest, "server id is required")
		return
	}

	scope := r.URL.Query().Get("scope")
	if scope != "global" {
		scope = "project"
	}

	if h.mcpManager == nil {
		writeError(w, http.StatusInternalServerError, "mcp manager not available")
		return
	}

	if err := h.mcpManager.DisableServer(serverID); err != nil {
		log.Printf("[devo] MCP disable during remove warning: %v", err)
	}

	if err := mcp.RemoveServerConfig(h.projectDir, scope, serverID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.mcpManager.RemoveServer(serverID)

	writeJSON(w, http.StatusOK, map[string]string{
		"server_id": serverID,
		"status":    "removed",
	})
}
