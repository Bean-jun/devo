package rest

import (
	"encoding/json"
	"net/http"

	"devo/internal/config"
	"devo/internal/taskexec/llmclient/providers"
	"devo/internal/taskexec/llmclient/providers/openai"
	"devo/internal/taskexec/tools"
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

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}

type onboardRequest struct {
	Name            string `json:"name"`
	APIKey          string `json:"api_key"`
	BaseURL         string `json:"base_url,omitempty"`
	Model           string `json:"model,omitempty"`
	EnableReasoning bool   `json:"enable_reasoning,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	MaxTokens       int    `json:"max_tokens,omitempty"`
}

func (h *Handler) OnboardLLM(w http.ResponseWriter, r *http.Request) {
	var req onboardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.APIKey == "" {
		writeError(w, http.StatusBadRequest, "name and api_key are required")
		return
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	id := config.Slugify(req.Name)
	baseURL := req.BaseURL
	if baseURL == "" {
		baseURL = config.DefaultLLMBaseURL
	}
	model := req.Model
	if model == "" {
		model = config.DefaultLLMModel
	}

	newModel := config.ModelConfig{
		ID:              id,
		Name:            req.Name,
		APIKey:          req.APIKey,
		BaseURL:         baseURL,
		Model:           model,
		EnableReasoning: req.EnableReasoning,
		ReasoningEffort: req.ReasoningEffort,
		MaxTokens:       req.MaxTokens,
	}

	cfg.LLM.Models = append(cfg.LLM.Models, newModel)
	cfg.LLM.ActiveModelID = id

	if err := config.SaveGlobalConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	h.cfg = cfg
	h.llmConfigured = true

	h.rebuildLLMClient(cfg)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"model_id": id,
		"model":    newModel,
	})
}

func (h *Handler) GetModels(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	models := make([]map[string]interface{}, 0, len(cfg.LLM.Models))
	for _, m := range cfg.LLM.Models {
		models = append(models, map[string]interface{}{
			"id":               m.ID,
			"name":             m.Name,
			"provider":         m.Provider,
			"api_key":          maskAPIKey(m.APIKey),
			"base_url":         m.BaseURL,
			"model":            m.Model,
			"enable_reasoning": m.EnableReasoning,
			"reasoning_effort": m.ReasoningEffort,
			"max_tokens":       m.MaxTokens,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"models":          models,
		"active_model_id": cfg.LLM.ActiveModelID,
	})
}

func (h *Handler) AddModel(w http.ResponseWriter, r *http.Request) {
	var req config.ModelConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.APIKey == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "name, api_key, and model are required")
		return
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	if req.ID == "" {
		req.ID = config.Slugify(req.Name)
	}

	for _, m := range cfg.LLM.Models {
		if m.ID == req.ID {
			writeError(w, http.StatusConflict, "model with id "+req.ID+" already exists")
			return
		}
	}

	if req.BaseURL == "" {
		req.BaseURL = config.DefaultLLMBaseURL
	}

	cfg.LLM.Models = append(cfg.LLM.Models, req)
	if cfg.LLM.ActiveModelID == "" {
		cfg.LLM.ActiveModelID = req.ID
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	h.cfg = cfg
	h.llmConfigured = true

	writeJSON(w, http.StatusCreated, req)
}

func (h *Handler) UpdateModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "model id is required")
		return
	}

	var req config.ModelConfig
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	found := false
	for i, m := range cfg.LLM.Models {
		if m.ID == id {
			if req.Name != "" {
				cfg.LLM.Models[i].Name = req.Name
			}
			if req.APIKey != "" {
				cfg.LLM.Models[i].APIKey = req.APIKey
			}
			if req.BaseURL != "" {
				cfg.LLM.Models[i].BaseURL = req.BaseURL
			}
			if req.Model != "" {
				cfg.LLM.Models[i].Model = req.Model
			}
			if req.ExtraHeaders != nil {
				cfg.LLM.Models[i].ExtraHeaders = req.ExtraHeaders
			}
			cfg.LLM.Models[i].EnableReasoning = req.EnableReasoning
			if req.ReasoningEffort != "" {
				cfg.LLM.Models[i].ReasoningEffort = req.ReasoningEffort
			}
			if req.MaxTokens > 0 {
				cfg.LLM.Models[i].MaxTokens = req.MaxTokens
			}
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}

	if err := config.SaveGlobalConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	h.cfg = cfg

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) DeleteModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "model id is required")
		return
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	found := false
	for i, m := range cfg.LLM.Models {
		if m.ID == id {
			cfg.LLM.Models = append(cfg.LLM.Models[:i], cfg.LLM.Models[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}

	if cfg.LLM.ActiveModelID == id {
		if len(cfg.LLM.Models) > 0 {
			cfg.LLM.ActiveModelID = cfg.LLM.Models[0].ID
		} else {
			cfg.LLM.ActiveModelID = ""
		}
	}

	llmConfigured := len(cfg.LLM.Models) > 0

	if err := config.SaveGlobalConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	h.cfg = cfg
	h.llmConfigured = llmConfigured

	if llmConfigured && id == cfg.LLM.ActiveModelID {
		h.rebuildLLMClient(cfg)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) ActivateModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "model id is required")
		return
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}

	found := false
	for _, m := range cfg.LLM.Models {
		if m.ID == id {
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}

	cfg.LLM.ActiveModelID = id

	if err := config.SaveGlobalConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	h.cfg = cfg
	h.rebuildLLMClient(cfg)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":         true,
		"active_model_id": id,
	})
}

func (h *Handler) TestModelConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "model id is required")
		return
	}

	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	var model *config.ModelConfig
	for i := range cfg.LLM.Models {
		if cfg.LLM.Models[i].ID == id {
			model = &cfg.LLM.Models[i]
			break
		}
	}

	if model == nil {
		writeError(w, http.StatusNotFound, "model not found")
		return
	}

	llmCfg := &config.LLMConfig{
		APIKey:          model.APIKey,
		BaseURL:         model.BaseURL,
		Model:           model.Model,
		ExtraHeaders:    model.ExtraHeaders,
		EnableReasoning: model.EnableReasoning,
		ReasoningEffort: model.ReasoningEffort,
		MaxTokens:       model.MaxTokens,
	}

	client := openai.New(openai.Config{
		LLMConfig: llmCfg,
	}, tools.NewRegistry())

	err = client.TestConnection(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
	})
}

func (h *Handler) rebuildLLMClient(cfg *config.Config) {
	if h.agentRegistry == nil {
		return
	}
	client := providers.NewClient(cfg, nil)
	h.getDefaultAgent().UpdateLLMClient(client)
}
