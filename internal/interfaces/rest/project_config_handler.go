package rest

import (
	"encoding/json"
	"net/http"

	"devo/internal/config"
)

type getProjectConfigResponse struct {
	Skills           []string          `json:"skills"`
	MCP              []string          `json:"mcp"`
	ApprovalPolicy   map[string]string `json:"approval_policy"`
	ToolCallLimit    int               `json:"tool_call_limit,omitempty"`
	MaxContextTokens int               `json:"max_context_tokens,omitempty"`
	KeepRecent       int               `json:"keep_recent,omitempty"`
}

func (h *Handler) GetProjectConfig(w http.ResponseWriter, r *http.Request) {
	wd := h.projectDir
	if wd == "" {
		writeJSON(w, http.StatusOK, getProjectConfigResponse{Skills: []string{}, MCP: []string{}, ApprovalPolicy: map[string]string{}})
		return
	}

	cfg, _ := config.LoadFullConfig(wd)
	if cfg == nil {
		writeJSON(w, http.StatusOK, getProjectConfigResponse{Skills: []string{}, MCP: []string{}, ApprovalPolicy: map[string]string{}})
		return
	}

	approvalPolicy := cfg.ApprovalPolicy
	if approvalPolicy == nil {
		approvalPolicy = map[string]string{}
	}

	writeJSON(w, http.StatusOK, getProjectConfigResponse{
		Skills:           cfg.Skills,
		MCP:              cfg.MCP,
		ApprovalPolicy:   approvalPolicy,
		ToolCallLimit:    cfg.ToolCallLimit,
		MaxContextTokens: cfg.MaxContextTokens,
		KeepRecent:       cfg.KeepRecent,
	})
}

type setProjectConfigRequest struct {
	Skills           []string          `json:"skills"`
	MCP              []string          `json:"mcp"`
	ApprovalPolicy   map[string]string `json:"approval_policy"`
	ToolCallLimit    int               `json:"tool_call_limit,omitempty"`
	MaxContextTokens int               `json:"max_context_tokens,omitempty"`
	KeepRecent       int               `json:"keep_recent,omitempty"`
}

func (h *Handler) SetProjectConfig(w http.ResponseWriter, r *http.Request) {
	wd := h.projectDir
	if wd == "" {
		writeError(w, http.StatusBadRequest, "no project directory set")
		return
	}

	if h.skillsManager == nil {
		writeError(w, http.StatusInternalServerError, "skills manager not available")
		return
	}

	var req setProjectConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg := &config.ProjectConfig{
		Skills:           req.Skills,
		MCP:              req.MCP,
		ApprovalPolicy:   req.ApprovalPolicy,
		ToolCallLimit:    req.ToolCallLimit,
		MaxContextTokens: req.MaxContextTokens,
		KeepRecent:       req.KeepRecent,
	}

	if err := config.SaveProjectConfig(wd, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	if err := h.skillsManager.RescanWithConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload skills: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
