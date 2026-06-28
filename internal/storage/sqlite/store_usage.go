package sqlite

import (
	"devo/internal/core/session"
)

func (s *GormStore) AddUsageStep(sessionID string, stepSeq int, inputTokens, outputTokens int, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	step := &TokenUsageStepModel{
		SessionID:    sessionID,
		StepSeq:      stepSeq,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Source:       source,
	}
	return s.db.Create(step).Error
}

func (s *GormStore) GetUsageSteps(sessionID string) ([]session.UsageStepRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var models []TokenUsageStepModel
	if err := s.db.Where("session_id = ?", sessionID).Order("step_seq ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	records := make([]session.UsageStepRecord, len(models))
	for i, m := range models {
		records[i] = session.UsageStepRecord{
			SessionID:    m.SessionID,
			StepSeq:      m.StepSeq,
			InputTokens:  m.InputTokens,
			OutputTokens: m.OutputTokens,
			Source:       m.Source,
			CreatedAt:    m.CreatedAt,
		}
	}
	return records, nil
}

func (s *GormStore) UpdateSessionUsage(sessionID string, inputTokens, outputTokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Model(&SessionModel{}).
		Where("id = ?", sessionID).
		Updates(map[string]interface{}{
			"token_usage_input":  s.db.Raw("token_usage_input + ?", inputTokens),
			"token_usage_output": s.db.Raw("token_usage_output + ?", outputTokens),
			"token_usage_total":  s.db.Raw("token_usage_input + token_usage_output + ?", inputTokens+outputTokens),
		}).Error
}

func (s *GormStore) GetUsageStats(groupBy, dateRange, project string) (*session.UsageStatsResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := &session.UsageStatsResult{
		Groups: make([]session.UsageGroup, 0),
	}

	type GroupResult struct {
		Key          string
		InputTokens  int
		OutputTokens int
		TotalTokens  int
	}

	var groupResults []GroupResult

	query := s.db.Model(&SessionModel{})

	if project != "" {
		query = query.Where("working_directory = ?", project)
	}

	switch groupBy {
	case "project":
		rows, err := query.Select("working_directory as key, SUM(token_usage_input) as input_tokens, SUM(token_usage_output) as output_tokens, SUM(token_usage_total) as total_tokens").Group("working_directory").Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var gr GroupResult
			if err := rows.Scan(&gr.Key, &gr.InputTokens, &gr.OutputTokens, &gr.TotalTokens); err != nil {
				return nil, err
			}
			groupResults = append(groupResults, gr)
		}
	case "session":
		rows, err := query.Select("id as key, token_usage_input as input_tokens, token_usage_output as output_tokens, token_usage_total as total_tokens").Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var gr GroupResult
			if err := rows.Scan(&gr.Key, &gr.InputTokens, &gr.OutputTokens, &gr.TotalTokens); err != nil {
				return nil, err
			}
			groupResults = append(groupResults, gr)
		}
	case "date":
		rows, err := query.Select("date(created_at) as key, SUM(token_usage_input) as input_tokens, SUM(token_usage_output) as output_tokens, SUM(token_usage_total) as total_tokens").Group("date(created_at)").Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var gr GroupResult
			if err := rows.Scan(&gr.Key, &gr.InputTokens, &gr.OutputTokens, &gr.TotalTokens); err != nil {
				return nil, err
			}
			groupResults = append(groupResults, gr)
		}
	default:
		rows, err := query.Select("'all' as key, SUM(token_usage_input) as input_tokens, SUM(token_usage_output) as output_tokens, SUM(token_usage_total) as total_tokens").Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var gr GroupResult
			if err := rows.Scan(&gr.Key, &gr.InputTokens, &gr.OutputTokens, &gr.TotalTokens); err != nil {
				return nil, err
			}
			groupResults = append(groupResults, gr)
		}
	}

	for _, gr := range groupResults {
		result.Groups = append(result.Groups, session.UsageGroup{
			Key:          gr.Key,
			InputTokens:  gr.InputTokens,
			OutputTokens: gr.OutputTokens,
			TotalTokens:  gr.TotalTokens,
		})
		result.Summary.Input += gr.InputTokens
		result.Summary.Output += gr.OutputTokens
		result.Summary.Total += gr.TotalTokens
	}

	return result, nil
}
