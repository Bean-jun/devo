package sqlite

import (
	"encoding/json"

	"devo/internal/core/session"
)

func (s *GormStore) Create(sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	if err := s.db.Model(&SessionModel{}).Where("id = ?", sess.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return session.ErrSessionConflict
	}

	model := fromDomain(sess)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	s.eventBus[sess.ID] = session.NewEventBus(session.DefaultEventHistorySize)
	return nil
}

func (s *GormStore) Get(id string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var model SessionModel
	if err := s.db.First(&model, "id = ?", id).Error; err != nil {
		return nil, session.ErrSessionNotFound
	}

	return model.ToDomain(), nil
}

func (s *GormStore) Update(sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var model SessionModel
	if err := s.db.First(&model, "id = ?", sess.ID).Error; err != nil {
		return session.ErrSessionNotFound
	}

	updates := map[string]interface{}{
		"title":                        sess.Title,
		"working_directory":            sess.WorkingDirectory,
		"state":                        string(sess.State),
		"last_active_at":               sess.LastActiveAt,
		"active_sse_connections":       sess.ActiveSSEConnections,
		"trust_level":                  sess.TrustLevel,
		"approval_timeout_seconds":     sess.ApprovalTimeoutSeconds,
		"tool_call_limit":              sess.ToolCallLimit,
		"tool_call_count":              sess.ToolCallCount,
		"last_loop_termination_reason": string(sess.LastLoopTerminationReason),
	}

	if sess.ApprovalPolicy != nil {
		data, _ := json.Marshal(sess.ApprovalPolicy)
		updates["approval_policy_json"] = string(data)
	}

	if err := s.db.Model(&model).Updates(updates).Error; err != nil {
		return err
	}

	return nil
}

func (s *GormStore) ListSessions(status, project string, limit, offset int) ([]session.Session, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := s.db.Model(&SessionModel{})

	if status != "" && status != "all" {
		query = query.Where("state = ?", status)
	}
	if project != "" {
		query = query.Where("working_directory = ?", project)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 20
	}

	var models []SessionModel
	if err := query.Offset(offset).Limit(limit).Order("last_active_at DESC").Find(&models).Error; err != nil {
		return nil, 0, err
	}

	sessions := make([]session.Session, len(models))
	for i, m := range models {
		sessions[i] = *m.ToDomain()
	}

	return sessions, int(total), nil
}
