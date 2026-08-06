package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"devo/internal/config"
	"devo/internal/core/concurrency"
	"devo/internal/core/session"
)

type ArchiveManager struct {
	store           session.SessionStore
	pathLockManager *concurrency.PathLockManager
}

func NewArchiveManager(store session.SessionStore, pathLockManager *concurrency.PathLockManager) *ArchiveManager {
	return &ArchiveManager{
		store:           store,
		pathLockManager: pathLockManager,
	}
}

func (am *ArchiveManager) ArchivePath(sess *session.Session) string {
	if sess.ArchivePath != "" {
		return sess.ArchivePath
	}
	return filepath.Join(config.ProjectSessionsDir(sess.WorkingDirectory), sess.ID+".md")
}

func (am *ArchiveManager) EnsureArchivePath(sess *session.Session) error {
	dir := filepath.Dir(am.ArchivePath(sess))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create archive directory: %w", err)
	}
	return nil
}

func (am *ArchiveManager) AppendUserMessage(sessionID string, content string) error {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)
	am.pathLockManager.Lock(archivePath)
	defer am.pathLockManager.Unlock(archivePath)

	if err := am.EnsureArchivePath(sess); err != nil {
		return err
	}

	entry := fmt.Sprintf("\n### 用户\n%s\n", content)
	return am.appendToFile(archivePath, entry)
}

func (am *ArchiveManager) AppendAssistantMessage(sessionID string, content string) error {
	return am.AppendAssistantMessageWithReasoning(sessionID, content, "")
}

func (am *ArchiveManager) AppendAssistantMessageWithReasoning(sessionID string, content string, reasoning string) error {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)
	am.pathLockManager.Lock(archivePath)
	defer am.pathLockManager.Unlock(archivePath)

	if err := am.EnsureArchivePath(sess); err != nil {
		return err
	}

	var b strings.Builder
	if reasoning != "" {
		b.WriteString("\n### 助手\n")
		b.WriteString("<details>\n<summary>💭 思考过程</summary>\n\n")
		b.WriteString(reasoning)
		b.WriteString("\n\n</details>\n\n")
		b.WriteString(content)
		b.WriteString("\n")
	} else {
		b.WriteString(fmt.Sprintf("\n### 助手\n%s\n", content))
	}
	return am.appendToFile(archivePath, b.String())
}

func (am *ArchiveManager) AppendToolCall(sessionID string, toolName string, params map[string]interface{}) error {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)
	am.pathLockManager.Lock(archivePath)
	defer am.pathLockManager.Unlock(archivePath)

	if err := am.EnsureArchivePath(sess); err != nil {
		return err
	}

	paramsStr := ""
	if params != nil {
		parts := make([]string, 0, len(params))
		for k, v := range params {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
		paramsStr = " (" + strings.Join(parts, ", ") + ")"
	}

	entry := fmt.Sprintf("\n**[工具调用: %s]**%s\n", toolName, paramsStr)
	return am.appendToFile(archivePath, entry)
}

func (am *ArchiveManager) AppendToolResult(sessionID string, toolName string, success bool, summary string) error {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)
	am.pathLockManager.Lock(archivePath)
	defer am.pathLockManager.Unlock(archivePath)

	if err := am.EnsureArchivePath(sess); err != nil {
		return err
	}

	status := "成功"
	if !success {
		status = "失败"
	}

	entry := fmt.Sprintf("> 结果(%s): %s\n", status, summary)
	return am.appendToFile(archivePath, entry)
}

func (am *ArchiveManager) AppendSystemMessage(sessionID string, content string) error {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)
	am.pathLockManager.Lock(archivePath)
	defer am.pathLockManager.Unlock(archivePath)

	if err := am.EnsureArchivePath(sess); err != nil {
		return err
	}

	entry := fmt.Sprintf("\n### 系统\n%s\n", content)
	return am.appendToFile(archivePath, entry)
}

func (am *ArchiveManager) appendToFile(path string, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open archive file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write archive file: %w", err)
	}

	return nil
}

func (am *ArchiveManager) SyncArchive(sessionID string) (string, error) {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return "", fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)
	am.pathLockManager.Lock(archivePath)
	defer am.pathLockManager.Unlock(archivePath)

	if err := am.EnsureArchivePath(sess); err != nil {
		return "", err
	}

	msgs, _, err := am.store.GetMessages(sessionID, 0, 0)
	if err != nil {
		return "", fmt.Errorf("get messages: %w", err)
	}

	content := am.renderArchive(sess, msgs)

	if err := os.WriteFile(archivePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write archive file: %w", err)
	}

	var lastMessageID string
	if len(msgs) > 0 {
		lastMessageID = msgs[len(msgs)-1].ID
	}

	return lastMessageID, nil
}

func (am *ArchiveManager) renderArchive(sess *session.Session, msgs []session.Message) string {
	var b strings.Builder

	b.WriteString("# 会话存档\n\n")

	b.WriteString(fmt.Sprintf("- **会话 ID**: %s\n", sess.ID))
	b.WriteString(fmt.Sprintf("- **创建时间**: %s\n", sess.CreatedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- **最后活跃时间**: %s\n", sess.LastActiveAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- **状态**: %s\n", sess.State))
	b.WriteString(fmt.Sprintf("- **工作目录**: %s\n", sess.WorkingDirectory))
	b.WriteString(fmt.Sprintf("- **Token 消耗**: 输入 %d | 输出 %d | 合计 %d\n", sess.TokenUsage.Input, sess.TokenUsage.Output, sess.TokenUsage.Total))
	b.WriteString(fmt.Sprintf("- **压缩次数**: %d\n", sess.CompressionCount))

	b.WriteString("\n---\n\n## 对话历史\n")

	for _, msg := range msgs {
		if msg.Role == session.RoleSystem {
			b.WriteString(fmt.Sprintf("\n### 系统\n%s\n", msg.Content))
		} else if msg.Role == session.RoleUser {
			b.WriteString(fmt.Sprintf("\n### 用户\n%s\n", msg.Content))
		} else if msg.Role == session.RoleAssistant {
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					paramsStr := ""
					if tc.Params != nil {
						parts := make([]string, 0, len(tc.Params))
						for k, v := range tc.Params {
							parts = append(parts, fmt.Sprintf("%s=%v", k, v))
						}
						paramsStr = " (" + strings.Join(parts, ", ") + ")"
					}
					b.WriteString(fmt.Sprintf("\n**[工具调用: %s]**%s\n", tc.ToolName, paramsStr))
				}
			}
			if msg.Reasoning != "" {
				b.WriteString("\n### 助手\n")
				b.WriteString("<details>\n<summary>💭 思考过程</summary>\n\n")
				b.WriteString(msg.Reasoning)
				b.WriteString("\n\n</details>\n\n")
				if msg.Content != "" {
					b.WriteString(msg.Content)
					b.WriteString("\n")
				}
			} else if msg.Content != "" {
				b.WriteString(fmt.Sprintf("\n### 助手\n%s\n", msg.Content))
			}
		} else if msg.Role == session.RoleTool {
			if msg.Content != "" {
				b.WriteString(fmt.Sprintf("> 结果: %s\n", msg.Content))
			}
		}
	}

	return b.String()
}

func (am *ArchiveManager) GetArchiveContent(sessionID string) ([]byte, error) {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)

	data, err := os.ReadFile(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read archive file: %w", err)
	}

	return data, nil
}

func (am *ArchiveManager) ArchiveExists(sessionID string) (bool, error) {
	sess, err := am.store.Get(sessionID)
	if err != nil {
		return false, fmt.Errorf("get session: %w", err)
	}

	archivePath := am.ArchivePath(sess)
	_, err = os.Stat(archivePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (am *ArchiveManager) GetLastMessageID(sessionID string) (string, error) {
	msgs, _, err := am.store.GetMessages(sessionID, 0, 0)
	if err != nil {
		return "", fmt.Errorf("get messages: %w", err)
	}
	if len(msgs) == 0 {
		return "", nil
	}
	return msgs[len(msgs)-1].ID, nil
}
