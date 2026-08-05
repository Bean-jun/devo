package sqlite

import (
	"devo/internal/core/session"
	"time"
)

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

	if err := s.db.Model(&SessionModel{}).Where("id = ?", sessionID).
		Update("message_count", maxSeq+1).Error; err != nil {
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

type lastMessageRow struct {
	SessionID string
	Content   string
	Role      string
	CreatedAt time.Time
}

func (s *GormStore) GetLastMessages(sessionIDs []string) (map[string]session.LastMessageInfo, error) {
	if len(sessionIDs) == 0 {
		return map[string]session.LastMessageInfo{}, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var rows []lastMessageRow
	err := s.db.Model(&MessageModel{}).
		Select("message_models.session_id, message_models.content, message_models.role, message_models.created_at").
		Joins("INNER JOIN (SELECT session_id, MAX(seq) AS max_seq FROM message_models WHERE session_id IN ? AND role = 'user' GROUP BY session_id) sub ON message_models.session_id = sub.session_id AND message_models.seq = sub.max_seq", sessionIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]session.LastMessageInfo, len(rows))
	for _, row := range rows {
		result[row.SessionID] = session.LastMessageInfo{
			SessionID: row.SessionID,
			Content:   row.Content,
			Role:      row.Role,
			CreatedAt: row.CreatedAt,
		}
	}
	return result, nil
}
