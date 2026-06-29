package rest

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"devo/internal/core/approval"
)

func (h *Handler) LoadUserApprovalPolicy() {
	raw := h.loadUserApprovalPolicyFromConfig()
	policy := make(map[approval.OperationType]approval.PolicyLevel)
	for k, v := range raw {
		policy[approval.OperationType(k)] = approval.PolicyLevel(v)
	}
	h.loop.GetApprovalManager().SetUserGlobalPolicy(policy)
}

func (h *Handler) GetUserApprovalPolicy(w http.ResponseWriter, r *http.Request) {
	policy := h.loadUserApprovalPolicyFromConfig()
	writeJSON(w, http.StatusOK, policy)
}

func (h *Handler) SetUserApprovalPolicy(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	configDir := h.userConfigDir
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "cannot determine user config directory")
			return
		}
		configDir = filepath.Join(homeDir, ".devo")
	}

	configPath := filepath.Join(configDir, "config.json")

	existing := make(map[string]json.RawMessage)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, &existing)
	}

	approvalData, err := json.Marshal(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal policy")
		return
	}
	existing["approval_policy"] = approvalData

	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to marshal config")
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save config: "+err.Error())
		return
	}

	policy := make(map[approval.OperationType]approval.PolicyLevel)
	for k, v := range req {
		policy[approval.OperationType(k)] = approval.PolicyLevel(v)
	}
	h.loop.GetApprovalManager().SetUserGlobalPolicy(policy)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) loadUserApprovalPolicyFromConfig() map[string]string {
	configDir := h.userConfigDir
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return map[string]string{}
		}
		configDir = filepath.Join(homeDir, ".devo")
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return map[string]string{}
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return map[string]string{}
	}

	approvalRaw, ok := raw["approval_policy"]
	if !ok {
		return map[string]string{}
	}

	var policy map[string]string
	if err := json.Unmarshal(approvalRaw, &policy); err != nil {
		return map[string]string{}
	}

	return policy
}
