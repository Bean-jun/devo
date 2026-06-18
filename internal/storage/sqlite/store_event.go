package sqlite

import (
	"devo/internal/core/session"
)

func (s *GormStore) loadEventBuses() error {
	var models []SessionModel
	if err := s.db.Find(&models).Error; err != nil {
		return err
	}

	for _, m := range models {
		eb := session.NewEventBus(session.DefaultEventHistorySize)
		s.eventBus[m.ID] = eb

		var events []EventModel
		if err := s.db.Where("session_id = ?", m.ID).Order("event_id").Find(&events).Error; err != nil {
			return err
		}

		for _, em := range events {
			evt, err := em.ToDomain()
			if err != nil {
				continue
			}
			eb.Publish(evt.Type, evt.Data)
		}
	}

	return nil
}

func (s *GormStore) GetEventBus(sessionID string) (*session.EventBus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	eb, ok := s.eventBus[sessionID]
	if !ok {
		return nil, session.ErrSessionNotFound
	}

	return eb, nil
}

func (s *GormStore) AddEvent(sessionID string, event session.Event) error {
	s.mu.RLock()
	eb, ok := s.eventBus[sessionID]
	s.mu.RUnlock()

	if !ok {
		return session.ErrSessionNotFound
	}

	model, err := fromEvent(sessionID, event)
	if err != nil {
		return err
	}

	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	eb.Publish(event.Type, event.Data)
	return nil
}

func (s *GormStore) GetEvents(sessionID string, sinceID int64) ([]session.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var models []EventModel
	if err := s.db.Where("session_id = ? AND event_id > ?", sessionID, sinceID).Order("event_id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	events := make([]session.Event, 0, len(models))
	for _, m := range models {
		evt, err := m.ToDomain()
		if err != nil {
			continue
		}
		events = append(events, evt)
	}

	return events, nil
}
