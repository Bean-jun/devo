package agentloop

import (
	"fmt"
	"time"

	"devo/internal/core/session"
)

type RollbackResult struct {
	ActualRollbackMessageID string `json:"actual_rollback_message_id"`
	Adjusted                bool   `json:"adjusted"`
	AdjustmentReason        string `json:"adjustment_reason,omitempty"`
	DeletedCount            int    `json:"deleted_count"`
}

func (l *Loop) Rollback(sessionID string, targetMessageID string) (*RollbackResult, error) {
	sess, err := l.store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	if sess.State == session.StateArchived {
		return nil, fmt.Errorf("%w: session is archived", session.ErrSessionArchived)
	}

	msgs, _, err := l.store.GetMessages(sessionID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	targetIdx := -1
	for i := range msgs {
		if msgs[i].ID == targetMessageID {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return nil, fmt.Errorf("%w: %s", session.ErrMessageNotFound, targetMessageID)
	}

	targetMsg := msgs[targetIdx]

	adjusted := false
	adjustmentReason := ""
	actualRollbackMessageID := targetMessageID

	if targetMsg.Role == session.RoleAssistant && len(targetMsg.ToolCalls) > 0 {
		if targetIdx+1 < len(msgs) && msgs[targetIdx+1].Role == session.RoleTool {
			adjusted = true
			adjustmentReason = fmt.Sprintf(
				"目标消息是一条包含工具调用的助手消息，其后存在对应的工具结果（消息ID: %s），回滚点已自动后移至工具结果之后，以保留完整的工具调用回合。",
				msgs[targetIdx+1].ID,
			)
			actualRollbackMessageID = msgs[targetIdx+1].ID
		}
	}

	if targetMsg.Role == session.RoleTool {
		if targetIdx > 0 && msgs[targetIdx-1].Role == session.RoleAssistant && len(msgs[targetIdx-1].ToolCalls) > 0 {
			adjusted = true
			adjustmentReason = fmt.Sprintf(
				"目标消息是一条工具结果消息，其前存在对应的工具调用请求（消息ID: %s），回滚点已自动前移至工具调用请求之前，以删除不完整的工具调用回合。",
				msgs[targetIdx-1].ID,
			)
			if targetIdx > 1 {
				actualRollbackMessageID = msgs[targetIdx-2].ID
			} else {
				actualRollbackMessageID = msgs[targetIdx-1].ID
			}
		}
	}

	rollbackTime := time.Now()

	deletedCount, err := l.store.DeleteMessagesAfter(sessionID, actualRollbackMessageID)
	if err != nil {
		return nil, fmt.Errorf("delete messages: %w", err)
	}

	sysMsg := session.Message{
		ID:        session.GenerateID("msg"),
		Role:      session.RoleSystem,
		Content:   buildRollbackSystemMessage(targetMessageID, rollbackTime, adjusted, adjustmentReason),
		CreatedAt: rollbackTime,
	}
	if err := l.store.AddMessage(sessionID, sysMsg); err != nil {
		return nil, fmt.Errorf("add rollback system message: %w", err)
	}

	fileWarnings := l.checkFileConsistency(sessionID, actualRollbackMessageID, msgs)
	if len(fileWarnings) > 0 {
		eventBus, err := l.store.GetEventBus(sessionID)
		if err == nil {
			eventBus.Publish("file_state_warning", map[string]any{
				"message":        buildFileWarningMessage(fileWarnings),
				"affected_files": fileWarnings,
			})
		}
	}

	oldState := string(sess.State)
	sess.State = session.StateIdle
	sess.CancelRequested = false
	sess.PauseRequested = false
	sess.CompressionState = nil
	sess.LastActiveAt = rollbackTime
	if err := l.store.Update(sess); err != nil {
		return nil, fmt.Errorf("update session state: %w", err)
	}

	eventBus, err := l.store.GetEventBus(sessionID)
	if err == nil {
		eventBus.Publish("session_state_change", map[string]any{
			"old_state": oldState,
			"new_state": string(session.StateIdle),
			"reason":    "rollback",
		})
	}

	return &RollbackResult{
		ActualRollbackMessageID: actualRollbackMessageID,
		Adjusted:                adjusted,
		AdjustmentReason:        adjustmentReason,
		DeletedCount:            deletedCount,
	}, nil
}

func (l *Loop) checkFileConsistency(sessionID string, rollbackMessageID string, msgs []session.Message) []string {
	rollbackIdx := -1
	for i := range msgs {
		if msgs[i].ID == rollbackMessageID {
			rollbackIdx = i
			break
		}
	}
	if rollbackIdx == -1 {
		return nil
	}

	modifications, err := l.store.GetFileModifications(sessionID)
	if err != nil || len(modifications) == 0 {
		return nil
	}

	rollbackTime := msgs[rollbackIdx].CreatedAt

	var warnings []string
	for _, mod := range modifications {
		if mod.ModifiedAt.After(rollbackTime) {
			warnings = append(warnings, mod.FilePath)
		}
	}

	return warnings
}

func buildRollbackSystemMessage(targetID string, rollbackTime time.Time, adjusted bool, reason string) string {
	base := fmt.Sprintf("用户于 %s 将对话回滚至消息 %s。", rollbackTime.Format(time.RFC3339), targetID)
	if adjusted && reason != "" {
		base += " " + reason
	}
	return base
}

func buildFileWarningMessage(files []string) string {
	msg := fmt.Sprintf("回滚完成，但检测到以下 %d 个文件可能在回滚点之后被工具修改过，文件状态可能与当前对话不一致：\n", len(files))
	for _, f := range files {
		msg += fmt.Sprintf("  - %s\n", f)
	}
	msg += "建议使用 Git 检查文件变更。"
	return msg
}
