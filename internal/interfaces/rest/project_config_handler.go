package rest

import (
	"encoding/json"
	"net/http"

	"devo/internal/core/config"
)

type getProjectConfigResponse struct {
	Skills []string `json:"skills"`
	MCP    []string `json:"mcp"`
}

func (h *Handler) GetProjectConfig(w http.ResponseWriter, r *http.Request) {
	wd := h.projectDir
	if wd == "" {
		writeJSON(w, http.StatusOK, getProjectConfigResponse{Skills: []string{}, MCP: []string{}})
		return
	}

	cfg, err := config.Load(wd)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config: "+err.Error())
		return
	}

	if cfg == nil {
		writeJSON(w, http.StatusOK, getProjectConfigResponse{Skills: []string{}, MCP: []string{}})
		return
	}

	writeJSON(w, http.StatusOK, getProjectConfigResponse{
		Skills: cfg.Skills,
		MCP:    cfg.MCP,
	})
}

type setProjectConfigRequest struct {
	Skills []string `json:"skills"`
	MCP    []string `json:"mcp"`
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
		Skills: req.Skills,
		MCP:    req.MCP,
	}

	if err := config.Save(wd, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	if err := h.skillsManager.RescanWithConfig(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reload skills: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
