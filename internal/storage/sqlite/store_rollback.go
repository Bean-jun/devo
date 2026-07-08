package sqlite

import (
	"time"

	"devo/internal/core/session"

	"gorm.io/gorm"
)

func (s *GormStore) GetMessageByID(sessionID string, messageID string) (*session.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var model MessageModel
	if err := s.db.Where("session_id = ? AND id = ?", sessionID, messageID).First(&model).Error; err != nil {
		return nil, session.ErrMessageNotFound
	}

	msg := model.ToDomain()
	return &msg, nil
}

func (s *GormStore) DeleteMessagesAfter(sessionID string, messageID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var target MessageModel
	if err := s.db.Where("session_id = ? AND id = ?", sessionID, messageID).First(&target).Error; err != nil {
		return 0, session.ErrMessageNotFound
	}

	result := s.db.Where("session_id = ? AND seq >= ?", sessionID, target.Seq).Delete(&MessageModel{})
	if result.Error != nil {
		return 0, result.Error
	}

	deleted := int(result.RowsAffected)
	if deleted > 0 {
		s.db.Model(&SessionModel{}).Where("id = ?", sessionID).
			Update("message_count", gorm.Expr("message_count - ?", deleted))
	}

	return deleted, nil
}

func (s *GormStore) RecordFileModification(record session.FileModificationRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	model := FileModificationLogModel{
		SessionID:         record.SessionID,
		FilePath:          record.FilePath,
		ModifiedAt:        record.ModifiedAt,
		CausedByMessageID: record.CausedByMessageID,
	}

	return s.db.Create(&model).Error
}

func (s *GormStore) GetFileModifications(sessionID string) ([]session.FileModificationRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var models []FileModificationLogModel
	if err := s.db.Where("session_id = ?", sessionID).Order("modified_at ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	records := make([]session.FileModificationRecord, len(models))
	for i, m := range models {
		records[i] = session.FileModificationRecord{
			SessionID:         m.SessionID,
			FilePath:          m.FilePath,
			ModifiedAt:        m.ModifiedAt,
			CausedByMessageID: m.CausedByMessageID,
		}
	}

	return records, nil
}

func (s *GormStore) DeleteFileModificationsAfter(sessionID string, afterTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Where("session_id = ? AND modified_at > ?", sessionID, afterTime).Delete(&FileModificationLogModel{}).Error
}
