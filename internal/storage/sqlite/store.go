package sqlite

import (
	"sync"

	"gorm.io/gorm"

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
