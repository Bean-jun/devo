package session

import (
	"sort"
	"sync"
	"time"
)

type InMemoryStore struct {
	mu                sync.RWMutex
	sessions          map[string]*Session
	usageSteps        map[string][]UsageStepRecord
	fileModifications map[string][]FileModificationRecord
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		sessions:   make(map[string]*Session),
		usageSteps: make(map[string][]UsageStepRecord),
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
	sess.MessageCount = len(sess.Messages)
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

func (s *InMemoryStore) ListUniqueWorkspaces() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	var dirs []string
	for _, sess := range s.sessions {
		if sess.WorkingDirectory == "" || seen[sess.WorkingDirectory] {
			continue
		}
		seen[sess.WorkingDirectory] = true
		dirs = append(dirs, sess.WorkingDirectory)
	}
	return dirs, nil
}

func (s *InMemoryStore) DeleteByWorkspace(path string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for id, sess := range s.sessions {
		if sess.WorkingDirectory == path {
			delete(s.sessions, id)
			count++
		}
	}
	return count, nil
}

func (s *InMemoryStore) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
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

func (s *InMemoryStore) AddUsageStep(sessionID string, stepSeq int, inputTokens, outputTokens int, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.usageSteps[sessionID] = append(s.usageSteps[sessionID], UsageStepRecord{
		SessionID:    sessionID,
		StepSeq:      stepSeq,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Source:       source,
		CreatedAt:    time.Now(),
	})
	return nil
}

func (s *InMemoryStore) GetUsageSteps(sessionID string) ([]UsageStepRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	steps := s.usageSteps[sessionID]
	result := make([]UsageStepRecord, len(steps))
	copy(result, steps)
	return result, nil
}

func (s *InMemoryStore) UpdateSessionUsage(sessionID string, inputTokens, outputTokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	sess.TokenUsage.Input += inputTokens
	sess.TokenUsage.Output += outputTokens
	sess.TokenUsage.Total = sess.TokenUsage.Input + sess.TokenUsage.Output
	return nil
}

func (s *InMemoryStore) GetUsageStats(groupBy, dateRange, project string) (*UsageStatsResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &UsageStatsResult{
		Groups: make([]UsageGroup, 0),
	}

	groupMap := make(map[string]*TokenUsage)

	for _, sess := range s.sessions {
		if project != "" && sess.WorkingDirectory != project {
			continue
		}

		var key string
		switch groupBy {
		case "project":
			key = sess.WorkingDirectory
		case "session":
			key = sess.ID
		case "date":
			key = sess.CreatedAt.Format("2006-01-02")
		default:
			key = "all"
		}

		if _, ok := groupMap[key]; !ok {
			groupMap[key] = &TokenUsage{}
		}
		groupMap[key].Input += sess.TokenUsage.Input
		groupMap[key].Output += sess.TokenUsage.Output
		groupMap[key].Total += sess.TokenUsage.Total
	}

	keys := make([]string, 0, len(groupMap))
	for k := range groupMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		u := groupMap[k]
		result.Groups = append(result.Groups, UsageGroup{
			Key:          k,
			InputTokens:  u.Input,
			OutputTokens: u.Output,
			TotalTokens:  u.Total,
		})
		result.Summary.Input += u.Input
		result.Summary.Output += u.Output
		result.Summary.Total += u.Total
	}

	return result, nil
}

func (s *InMemoryStore) Close() error {
	return nil
}

func (s *InMemoryStore) GetMessageByID(sessionID string, messageID string) (*Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	for i := range sess.Messages {
		if sess.Messages[i].ID == messageID {
			cp := sess.Messages[i]
			return &cp, nil
		}
	}

	return nil, ErrMessageNotFound
}

func (s *InMemoryStore) DeleteMessagesAfter(sessionID string, messageID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return 0, ErrSessionNotFound
	}

	cutoffIdx := -1
	for i := range sess.Messages {
		if sess.Messages[i].ID == messageID {
			cutoffIdx = i
			break
		}
	}

	if cutoffIdx == -1 {
		return 0, ErrMessageNotFound
	}

	deletedCount := len(sess.Messages) - cutoffIdx - 1
	if deletedCount > 0 {
		sess.Messages = sess.Messages[:cutoffIdx+1]
		sess.MessageCount = len(sess.Messages)
	}

	return deletedCount, nil
}

func (s *InMemoryStore) RecordFileModification(record FileModificationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[record.SessionID]; !ok {
		return ErrSessionNotFound
	}

	if s.fileModifications == nil {
		s.fileModifications = make(map[string][]FileModificationRecord)
	}

	s.fileModifications[record.SessionID] = append(s.fileModifications[record.SessionID], record)
	return nil
}

func (s *InMemoryStore) GetFileModifications(sessionID string) ([]FileModificationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.fileModifications == nil {
		return nil, nil
	}

	records := s.fileModifications[sessionID]
	result := make([]FileModificationRecord, len(records))
	copy(result, records)
	return result, nil
}

func (s *InMemoryStore) DeleteFileModificationsAfter(sessionID string, afterTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.fileModifications == nil {
		return nil
	}

	records := s.fileModifications[sessionID]
	var filtered []FileModificationRecord
	for _, r := range records {
		if !r.ModifiedAt.After(afterTime) {
			filtered = append(filtered, r)
		}
	}
	s.fileModifications[sessionID] = filtered
	return nil
}
