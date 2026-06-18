package session

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type State string

const (
	StateIdle       State = "Idle"
	StateProcessing State = "Processing"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type Message struct {
	ID        string    `json:"id"`
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID                   string    `json:"id"`
	Title                string    `json:"title"`
	WorkingDirectory     string    `json:"working_directory"`
	State                State     `json:"state"`
	CreatedAt            time.Time `json:"created_at"`
	LastActiveAt         time.Time `json:"last_active_at"`
	Messages             []Message `json:"messages,omitempty"`
	ActiveSSEConnections int       `json:"active_sse_connections"`
	EventBus             *EventBus `json:"-"`
}

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionNotIdle  = errors.New("session is not idle")
	ErrSessionConflict = errors.New("session id already exists")
)

type SessionStore interface {
	Create(s *Session) error
	Get(id string) (*Session, error)
	Update(s *Session) error
	AddMessage(sessionID string, msg Message) error
	GetMessages(sessionID string, limit, offset int) ([]Message, int, error)
	GetEventBus(sessionID string) (*EventBus, error)
	IncrementSSEConnections(sessionID string) error
	DecrementSSEConnections(sessionID string) error
}

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

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%d-%04d", prefix, time.Now().UnixNano(), rng.Intn(10000))
}
