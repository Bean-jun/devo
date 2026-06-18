package sqlite

import (
	"devo/internal/core/session"
)

func (s *GormStore) IncrementSSEConnections(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var model SessionModel
	if err := s.db.First(&model, "id = ?", sessionID).Error; err != nil {
		return session.ErrSessionNotFound
	}

	model.ActiveSSEConnections++
	if err := s.db.Model(&model).Update("active_sse_connections", model.ActiveSSEConnections).Error; err != nil {
		return err
	}

	return nil
}

func (s *GormStore) DecrementSSEConnections(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var model SessionModel
	if err := s.db.First(&model, "id = ?", sessionID).Error; err != nil {
		return session.ErrSessionNotFound
	}

	if model.ActiveSSEConnections > 0 {
		model.ActiveSSEConnections--
	}
	if err := s.db.Model(&model).Update("active_sse_connections", model.ActiveSSEConnections).Error; err != nil {
		return err
	}

	return nil
}
