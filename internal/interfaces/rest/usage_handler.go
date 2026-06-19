package rest

import (
	"net/http"
	"strconv"

	"devo/internal/core/session"
)

func (h *Handler) GetSessionUsage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "missing session id")
		return
	}

	sess, err := h.store.Get(sessionID)
	if err != nil {
		if err == session.ErrSessionNotFound {
			writeError(w, http.StatusNotFound, "session not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := map[string]any{
		"total_input_tokens":  sess.TokenUsage.Input,
		"total_output_tokens": sess.TokenUsage.Output,
		"total_tokens":        sess.TokenUsage.Total,
		"compression_count":   0,
	}

	steps, err := h.store.GetUsageSteps(sessionID)
	if err == nil && len(steps) > 0 {
		stepList := make([]map[string]any, len(steps))
		for i, s := range steps {
			stepList[i] = map[string]any{
				"step_seq":      s.StepSeq,
				"input_tokens":  s.InputTokens,
				"output_tokens": s.OutputTokens,
				"created_at":    s.CreatedAt,
			}
		}
		response["steps"] = stepList
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetUsageStats(w http.ResponseWriter, r *http.Request) {
	groupBy := r.URL.Query().Get("group_by")
	dateRange := r.URL.Query().Get("date_range")
	project := r.URL.Query().Get("project")

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	limit := 20
	offset := 0
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	stats, err := h.store.GetUsageStats(groupBy, dateRange, project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	groups := stats.Groups
	if offset >= len(groups) {
		groups = []session.UsageGroup{}
	} else {
		end := offset + limit
		if end > len(groups) {
			end = len(groups)
		}
		groups = groups[offset:end]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"groups":  groups,
		"summary": stats.Summary,
	})
}
