package rest

import (
	"encoding/json"
	"net/http"

	"devo/internal/config"
	"devo/internal/core/approval"
)

type llmConfigResponse struct {
	BaseURL         string `json:"base_url,omitempty"`
	Model           string `json:"model,omitempty"`
	EnableReasoning bool   `json:"enable_reasoning,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	MaxTokens       int    `json:"max_tokens,omitempty"`
}

type getGlobalConfigResponse struct {
	LLM              llmConfigResponse `json:"llm,omitempty"`
	ApprovalPolicy   map[string]string `json:"approval_policy,omitempty"`
	Skills           []string          `json:"skills,omitempty"`
	MCP              []string          `json:"mcp,omitempty"`
	ToolCallLimit    int               `json:"tool_call_limit,omitempty"`
	MaxContextTokens int               `json:"max_context_tokens,omitempty"`
	KeepRecent       int               `json:"keep_recent,omitempty"`
}

type setGlobalConfigRequest struct {
	LLM              *config.LLMConfig `json:"llm,omitempty"`
	ApprovalPolicy   map[string]string `json:"approval_policy,omitempty"`
	Skills           []string          `json:"skills,omitempty"`
	MCP              []string          `json:"mcp,omitempty"`
	ToolCallLimit    *int              `json:"tool_call_limit,omitempty"`
	MaxContextTokens *int              `json:"max_context_tokens,omitempty"`
	KeepRecent       *int              `json:"keep_recent,omitempty"`
}

func (h *Handler) GetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	writeJSON(w, http.StatusOK, getGlobalConfigResponse{
		LLM: llmConfigResponse{
			BaseURL:         cfg.LLM.BaseURL,
			Model:           cfg.LLM.Model,
			EnableReasoning: cfg.LLM.EnableReasoning,
			ReasoningEffort: cfg.LLM.ReasoningEffort,
			MaxTokens:       cfg.LLM.MaxTokens,
		},
		ApprovalPolicy:   cfg.ApprovalPolicy,
		Skills:           cfg.Skills,
		MCP:              cfg.MCP,
		ToolCallLimit:    cfg.ToolCallLimit,
		MaxContextTokens: cfg.MaxContextTokens,
		KeepRecent:       cfg.KeepRecent,
	})
}

func (h *Handler) SetGlobalConfig(w http.ResponseWriter, r *http.Request) {
	var req setGlobalConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if h.cfg == nil {
		cfg, _ := config.LoadGlobal()
		if cfg == nil {
			cfg = &config.Config{}
		}
		h.cfg = cfg
	}

	if req.LLM != nil {
		if req.LLM.APIKey != "" {
			h.cfg.LLM.APIKey = req.LLM.APIKey
		}
		if req.LLM.BaseURL != "" {
			h.cfg.LLM.BaseURL = req.LLM.BaseURL
		}
		if req.LLM.Model != "" {
			h.cfg.LLM.Model = req.LLM.Model
		}
		if req.LLM.ExtraHeaders != nil {
			h.cfg.LLM.ExtraHeaders = req.LLM.ExtraHeaders
		}
		h.cfg.LLM.EnableReasoning = req.LLM.EnableReasoning
		if req.LLM.ReasoningEffort != "" {
			h.cfg.LLM.ReasoningEffort = req.LLM.ReasoningEffort
		}
		if req.LLM.MaxTokens > 0 {
			h.cfg.LLM.MaxTokens = req.LLM.MaxTokens
		}
	}
	if req.ApprovalPolicy != nil {
		h.cfg.ApprovalPolicy = req.ApprovalPolicy
	}
	if req.Skills != nil {
		h.cfg.Skills = req.Skills
	}
	if req.MCP != nil {
		h.cfg.MCP = req.MCP
	}
	if req.ToolCallLimit != nil {
		h.cfg.ToolCallLimit = *req.ToolCallLimit
	}
	if req.MaxContextTokens != nil {
		h.cfg.MaxContextTokens = *req.MaxContextTokens
	}
	if req.KeepRecent != nil {
		h.cfg.KeepRecent = *req.KeepRecent
	}

	if err := config.SaveGlobalConfig(h.cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save global config: "+err.Error())
		return
	}

	h.LoadUserApprovalPolicy()

	writeJSON(w, http.StatusOK, getGlobalConfigResponse{
		LLM: llmConfigResponse{
			BaseURL:         h.cfg.LLM.BaseURL,
			Model:           h.cfg.LLM.Model,
			EnableReasoning: h.cfg.LLM.EnableReasoning,
			ReasoningEffort: h.cfg.LLM.ReasoningEffort,
			MaxTokens:       h.cfg.LLM.MaxTokens,
		},
		ApprovalPolicy:   h.cfg.ApprovalPolicy,
		Skills:           h.cfg.Skills,
		MCP:              h.cfg.MCP,
		ToolCallLimit:    h.cfg.ToolCallLimit,
		MaxContextTokens: h.cfg.MaxContextTokens,
		KeepRecent:       h.cfg.KeepRecent,
	})
}

func (h *Handler) LoadUserApprovalPolicy() {
	raw := h.loadUserApprovalPolicyFromConfig()
	policy := make(map[approval.OperationType]approval.PolicyLevel)
	for k, v := range raw {
		policy[approval.OperationType(k)] = approval.PolicyLevel(v)
	}
	h.loop.GetApprovalManager().SetUserGlobalPolicy(policy)
}

func (h *Handler) loadUserApprovalPolicyFromConfig() map[string]string {
	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		return map[string]string{}
	}
	if cfg.ApprovalPolicy == nil {
		return map[string]string{}
	}
	return cfg.ApprovalPolicy
}
