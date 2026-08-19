package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"devo/internal/core/approval"
	"devo/internal/core/session"
	"devo/internal/core/skills"
)

type skillItem struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
	Location    string `json:"location,omitempty"`
	InstalledAt string `json:"installed_at,omitempty"`
}

type getSkillsResponse struct {
	Skills []skillItem `json:"skills"`
}

func (h *Handler) GetSkills(w http.ResponseWriter, r *http.Request) {
	if h.skillsManager == nil {
		writeJSON(w, http.StatusOK, getSkillsResponse{Skills: []skillItem{}})
		return
	}

	sourceFilter := r.URL.Query().Get("source")

	allSkills := h.skillsManager.GetAllSkills()
	items := make([]skillItem, 0, len(allSkills))
	for _, s := range allSkills {
		if sourceFilter != "" && string(s.Source) != sourceFilter {
			continue
		}

		installedAt := ""
		if !s.InstalledAt.IsZero() {
			installedAt = s.InstalledAt.Format(time.RFC3339)
		}

		items = append(items, skillItem{
			Name:        s.Name,
			Source:      string(s.Source),
			Priority:    s.Priority,
			Enabled:     s.Enabled,
			Location:    s.Location,
			InstalledAt: installedAt,
		})
	}

	writeJSON(w, http.StatusOK, getSkillsResponse{Skills: items})
}

func (h *Handler) ReloadSkills(w http.ResponseWriter, r *http.Request) {
	if h.skillsManager == nil {
		writeError(w, http.StatusInternalServerError, "skills manager not available")
		return
	}

	if err := h.skillsManager.ReloadSkills(); err != nil {
		writeError(w, http.StatusInternalServerError, "reload skills failed: "+err.Error())
		return
	}

	allSkills := h.skillsManager.GetAllSkills()
	items := make([]skillItem, 0, len(allSkills))
	for _, s := range allSkills {
		installedAt := ""
		if !s.InstalledAt.IsZero() {
			installedAt = s.InstalledAt.Format(time.RFC3339)
		}
		items = append(items, skillItem{
			Name:        s.Name,
			Source:      string(s.Source),
			Priority:    s.Priority,
			Enabled:     s.Enabled,
			Location:    s.Location,
			InstalledAt: installedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"skills":  items,
		"message": "skills reloaded",
	})
}

func (h *Handler) DeleteSkillByName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		writeError(w, http.StatusBadRequest, "skill name is required")
		return
	}

	if h.skillsManager == nil {
		writeError(w, http.StatusInternalServerError, "skills manager not available")
		return
	}

	if err := h.skillsManager.DeleteSkill(name); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"name":   name,
		"status": "removed",
	})
}

type setSkillsRequest struct {
	Enable  []string `json:"enable"`
	Disable []string `json:"disable"`
}

type setSkillsResponse struct {
	ActiveSkills []string `json:"active_skills"`
}

func (h *Handler) SetSessionSkills(w http.ResponseWriter, r *http.Request) {
	_ = r.PathValue("id")

	if h.skillsManager == nil {
		writeError(w, http.StatusInternalServerError, "skills manager not available")
		return
	}

	var req setSkillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	for _, name := range req.Enable {
		if err := h.skillsManager.EnableSkill(name); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to enable skill: "+err.Error())
			return
		}
	}
	for _, name := range req.Disable {
		if err := h.skillsManager.DisableSkill(name); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to disable skill: "+err.Error())
			return
		}
	}

	enabledSkills := h.skillsManager.GetEnabledSkills()
	activeSkillNames := make([]string, 0, len(enabledSkills))
	for _, s := range enabledSkills {
		activeSkillNames = append(activeSkillNames, s.Name)
	}

	writeJSON(w, http.StatusOK, setSkillsResponse{ActiveSkills: activeSkillNames})
}

type installSkillRequest struct {
	Source string `json:"source"`
	Value  string `json:"value"`
}

type installSkillResponse struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	InstalledAt string `json:"installed_at"`
}

func (h *Handler) InstallSkill(w http.ResponseWriter, r *http.Request) {
	if h.skillsManager == nil {
		writeError(w, http.StatusInternalServerError, "skills manager not available")
		return
	}

	var req installSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Value == "" {
		writeError(w, http.StatusBadRequest, "value is required")
		return
	}

	skill, err := h.skillsManager.InstallSkill(req.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, installSkillResponse{
		Name:        skill.Name,
		Source:      string(skill.Source),
		InstalledAt: skill.InstalledAt.Format(time.RFC3339),
	})
}

type solidifyResponse struct {
	ApprovalID string `json:"approval_id,omitempty"`
	SkillName  string `json:"skill_name,omitempty"`
	Location   string `json:"location,omitempty"`
	NoSkill    bool   `json:"no_skill"`
}

func (h *Handler) SolidifySession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	sess, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if sess.State != session.StateCompleted {
		writeError(w, http.StatusBadRequest, "session must be in completed state to solidify")
		return
	}

	if h.skillsManager == nil || h.agentRegistry == nil {
		writeError(w, http.StatusInternalServerError, "skills manager or agent registry not available")
		return
	}

	result, err := h.getAgent(sess).SolidifySession(context.Background(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "solidify failed: "+err.Error())
		return
	}

	if result.NoSkill {
		writeJSON(w, http.StatusOK, solidifyResponse{NoSkill: true})
		return
	}

	approvalManager := h.getAgent(sess).GetApprovalManager()
	opType := approval.OpSolidifySkill
	sessionPolicy := make(map[approval.OperationType]approval.PolicyLevel)
	for k, v := range sess.ApprovalPolicy {
		sessionPolicy[approval.OperationType(k)] = approval.PolicyLevel(v)
	}
	effectivePolicy := approvalManager.ResolveEffectivePolicy(sessionPolicy, nil, opType)

	if approvalManager.IsAutoApproved(effectivePolicy) {
		skill, err := h.skillsManager.SaveSkill(result.SkillName, result.Content)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save skill: "+err.Error())
			return
		}

		eventBus, _ := h.store.GetEventBus(id)
		if eventBus != nil {
			eventBus.Publish("skill_solidified", map[string]any{
				"skill_name": skill.Name,
				"location":   skill.Location,
			})
		}

		writeJSON(w, http.StatusOK, solidifyResponse{
			SkillName: skill.Name,
			Location:  skill.Location,
		})
		return
	}

	details := map[string]any{
		"skill_name":    result.SkillName,
		"draft_content": result.Content,
	}

	approvalReq := approvalManager.CreateRequest(id, "", opType, approval.RiskHigh, details)

	eventBus, _ := h.store.GetEventBus(id)
	if eventBus != nil {
		eventBus.Publish("approval_required", map[string]any{
			"approval_id":    approvalReq.ID,
			"operation_type": string(opType),
			"details":        details,
		})
	}

	writeJSON(w, http.StatusOK, solidifyResponse{
		ApprovalID: approvalReq.ID,
		SkillName:  result.SkillName,
	})
}

func (h *Handler) SolidifyApprove(sessionID, approvalID string, approved bool) (*skills.Skill, error) {
	if h.skillsManager == nil {
		return nil, errors.New("skills manager not available")
	}

	sess, err := h.store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	approvalManager := h.getAgent(sess).GetApprovalManager()
	req, ok := approvalManager.GetRequest(approvalID)
	if !ok {
		return nil, errors.New("approval request not found")
	}

	if req.OperationType != approval.OpSolidifySkill {
		return nil, errors.New("not a solidify approval request")
	}

	if approved {
		skillName, _ := req.Details["skill_name"].(string)
		content, _ := req.Details["draft_content"].(string)

		skill, err := h.skillsManager.SaveSkill(skillName, content)
		if err != nil {
			return nil, err
		}

		eventBus, _ := h.store.GetEventBus(sessionID)
		if eventBus != nil {
			eventBus.Publish("skill_solidified", map[string]any{
				"skill_name": skill.Name,
				"location":   skill.Location,
			})
		}

		return skill, nil
	}

	return nil, nil
}
