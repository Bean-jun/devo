package session

import (
	"sync"
)

type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]*Session),
	}
}

func (s *InMemoryStore) Create(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sess.ID]; exists {
		return ErrSessionConflict
	}

	cp := *sess
	cp.Messages = make([]Message, 0)
	cp.EventBus = NewEventBus(DefaultEventHistorySize)
	s.sessions[sess.ID] = &cp
	return nil
}

func (s *InMemoryStore) Get(id string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}

	cp := *sess
	return &cp, nil
}

func (s *InMemoryStore) Update(sess *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.sessions[sess.ID]
	if !ok {
		return ErrSessionNotFound
	}

	cp := *sess
	cp.Messages = existing.Messages
	cp.EventBus = existing.EventBus
	cp.ActiveSSEConnections = existing.ActiveSSEConnections
	s.sessions[sess.ID] = &cp
	return nil
}

func (s *InMemoryStore) AddMessage(sessionID string, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	sess.Messages = append(sess.Messages, msg)
	return nil
}

func (s *InMemoryStore) GetMessages(sessionID string, limit, offset int) ([]Message, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, 0, ErrSessionNotFound
	}

	msgs := sess.Messages
	total := len(msgs)

	if offset >= total {
		return []Message{}, total, nil
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	result := make([]Message, end-offset)
	copy(result, msgs[offset:end])
	return result, total, nil
}

func (s *InMemoryStore) GetEventBus(sessionID string) (*EventBus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	return sess.EventBus, nil
}

func (s *InMemoryStore) IncrementSSEConnections(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	sess.ActiveSSEConnections++
	return nil
}

func (s *InMemoryStore) DecrementSSEConnections(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	if sess.ActiveSSEConnections > 0 {
		sess.ActiveSSEConnections--
	}
	return nil
}

func (s *InMemoryStore) ListSessions(status, project string, limit, offset int) ([]Session, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Session
	for _, sess := range s.sessions {
		if status != "" && status != "all" && string(sess.State) != status {
			continue
		}
		if project != "" && sess.WorkingDirectory != project {
			continue
		}
		result = append(result, *sess)
	}

	total := len(result)

	if offset >= total {
		return []Session{}, total, nil
	}

	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}

	return result[offset:end], total, nil
}

func (s *InMemoryStore) AddEvent(sessionID string, event Event) error {
	s.mu.RLock()
	sess, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return ErrSessionNotFound
	}
	sess.EventBus.Publish(event.Type, event.Data)
	return nil
}

func (s *InMemoryStore) GetEvents(sessionID string, sinceID int64) ([]Event, error) {
	s.mu.RLock()
	sess, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrSessionNotFound
	}
	return sess.EventBus.GetHistory(sinceID), nil
}

func (s *InMemoryStore) Close() error {
	return nil
}
