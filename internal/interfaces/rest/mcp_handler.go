package rest

import (
	"net/http"
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
