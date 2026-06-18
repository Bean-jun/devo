package sqlite

import (
	"encoding/json"
	"sync"
	"time"

	"gorm.io/gorm"

	"devo/internal/core/approval"
	"devo/internal/core/session"
)

type GormStore struct {
	mu       sync.RWMutex
	db       *gorm.DB
	eventBus map[string]*session.EventBus
}

func NewGormStore(db *gorm.DB) (*GormStore, error) {
	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	store := &GormStore{
		db:       db,
		eventBus: make(map[string]*session.EventBus),
	}

	if err := store.loadEventBuses(); err != nil {
		return nil, err
	}

	return store, nil
}

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
		"title":                    sess.Title,
		"working_directory":        sess.WorkingDirectory,
		"state":                    string(sess.State),
		"last_active_at":           sess.LastActiveAt,
		"active_sse_connections":   sess.ActiveSSEConnections,
		"trust_level":              sess.TrustLevel,
		"approval_timeout_seconds": sess.ApprovalTimeoutSeconds,
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

func (s *GormStore) AddMessage(sessionID string, msg session.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	if err := s.db.Model(&SessionModel{}).Where("id = ?", sessionID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return session.ErrSessionNotFound
	}

	var maxSeq int
	s.db.Model(&MessageModel{}).Where("session_id = ?", sessionID).Select("COALESCE(MAX(seq), 0)").Scan(&maxSeq)

	model := fromMessage(sessionID, maxSeq+1, msg)
	if err := s.db.Create(model).Error; err != nil {
		return err
	}

	return nil
}

func (s *GormStore) GetMessages(sessionID string, limit, offset int) ([]session.Message, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := s.db.Model(&MessageModel{}).Where("session_id = ?", sessionID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = int(total)
		if limit == 0 {
			limit = 1
		}
	}

	var models []MessageModel
	if err := query.Order("seq ASC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}

	msgs := make([]session.Message, len(models))
	for i, m := range models {
		msgs[i] = m.ToDomain()
	}

	return msgs, int(total), nil
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

func (s *GormStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

func (s *GormStore) DB() *gorm.DB {
	return s.db
}

const fullTrustKeyPrefix = "full_trust:"

func (s *GormStore) GetFullTrust(operationType approval.OperationType) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var model UserConfigModel
	key := fullTrustKeyPrefix + string(operationType)
	if err := s.db.First(&model, "key = ?", key).Error; err != nil {
		return false, nil
	}

	return model.Value == "true", nil
}

func (s *GormStore) SetFullTrust(operationType approval.OperationType, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fullTrustKeyPrefix + string(operationType)
	value := "true"
	if !enabled {
		value = "false"
	}

	var existing UserConfigModel
	if err := s.db.First(&existing, "key = ?", key).Error; err != nil {
		return s.db.Create(&UserConfigModel{Key: key, Value: value}).Error
	}

	return s.db.Model(&existing).Update("value", value).Error
}

func (s *GormStore) GetAllFullTrust() (map[approval.OperationType]bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var models []UserConfigModel
	if err := s.db.Where("key LIKE ?", fullTrustKeyPrefix+"%").Find(&models).Error; err != nil {
		return nil, err
	}

	result := make(map[approval.OperationType]bool)
	for _, m := range models {
		opType := approval.OperationType(m.Key[len(fullTrustKeyPrefix):])
		result[opType] = m.Value == "true"
	}

	return result, nil
}

func init() {
	time.Local = time.UTC
}
