package sqlite

import (
	"devo/internal/core/approval"
)

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
