package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/session"
)

type approveRequest struct {
	Decision string `json:"decision"`
}

type approveResponse struct {
	ApprovalID string `json:"approval_id"`
	Decision   string `json:"decision"`
	ResolvedAt string `json:"resolved_at"`
}

func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	approvalID := r.PathValue("approval_id")

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	var req approveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Decision != "approve" && req.Decision != "reject" {
		writeError(w, http.StatusBadRequest, "decision must be 'approve' or 'reject'")
		return
	}

	if sess.State != session.StateAwaitingApproval {
		writeError(w, http.StatusConflict, "session is not in AwaitingApproval state")
		return
	}

	approvalManager := h.getAgent(sess).GetApprovalManager()
	approvalReq, exists := approvalManager.GetRequest(approvalID)
	if !exists {
		writeError(w, http.StatusNotFound, "approval request not found")
		return
	}

	if approvalReq.OperationType == approval.OpSolidifySkill {
		approved := req.Decision == "approve"
		skill, err := h.SolidifyApprove(id, approvalID, approved)
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}

		approvalManager.Resolve(approvalID, approval.StatusApproved)

		eventBus, _ := h.store.GetEventBus(id)
		if eventBus != nil {
			eventBus.Publish("approval_resolved", map[string]any{
				"approval_id":    approvalID,
				"decision":       req.Decision,
				"operation_type": string(approval.OpSolidifySkill),
			})
		}

		response := map[string]interface{}{
			"approval_id": approvalID,
			"decision":    req.Decision,
			"resolved_at": time.Now().Format(time.RFC3339),
		}
		if skill != nil {
			response["skill_name"] = skill.Name
			response["location"] = skill.Location
		}
		writeJSON(w, http.StatusOK, response)
		return
	}

	if err := h.getAgent(sess).ResolveApproval(id, approvalID, req.Decision); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "has expired") {
			writeError(w, http.StatusConflict, errMsg)
			return
		}
		writeError(w, http.StatusConflict, errMsg)
		return
	}

	writeJSON(w, http.StatusOK, approveResponse{
		ApprovalID: approvalID,
		Decision:   req.Decision,
		ResolvedAt: time.Now().Format(time.RFC3339),
	})
}

type setTrustLevelRequest struct {
	TrustLevel string `json:"trust_level"`
}

func (h *Handler) SetTrustLevel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req setTrustLevelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !approval.IsValidTrustLevel(req.TrustLevel) {
		writeError(w, http.StatusBadRequest, "trust_level must be one of: low, normal, elevated")
		return
	}

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	sess.TrustLevel = req.TrustLevel
	sess.LastActiveAt = time.Now()

	if err := h.store.Update(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"trust_level": req.TrustLevel})
}

type setApprovalPolicyRequest map[string]string

func (h *Handler) SetApprovalPolicy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req setApprovalPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	for opType, policyLevel := range req {
		if !approval.IsValidOperationType(opType) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid operation_type: %s", opType))
			return
		}
		if !approval.IsValidPolicyLevel(policyLevel) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid policy_level: %s", policyLevel))
			return
		}
	}

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if sess.ApprovalPolicy == nil {
		sess.ApprovalPolicy = make(map[string]string)
	}

	for opType, policyLevel := range req {
		sess.ApprovalPolicy[opType] = policyLevel
	}

	sess.LastActiveAt = time.Now()

	if err := h.store.Update(sess); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update session")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"approval_policy": sess.ApprovalPolicy,
	})
}
