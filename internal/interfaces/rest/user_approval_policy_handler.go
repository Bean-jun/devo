package rest

import (
	"encoding/json"
	"net/http"

	"devo/internal/config"
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

	cfg, _ := config.LoadGlobal()
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.ApprovalPolicy = req

	if err := config.SaveGlobalConfig(cfg); err != nil {
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
	cfg, err := config.LoadGlobal()
	if err != nil || cfg == nil {
		return map[string]string{}
	}
	if cfg.ApprovalPolicy == nil {
		return map[string]string{}
	}
	return cfg.ApprovalPolicy
}
