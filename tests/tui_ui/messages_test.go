package main

import (
	"strings"
	"testing"
)

// ─── Mock 数据测试 ───

func TestMockMessages_NotEmpty(t *testing.T) {
	msgs := mockMessages()
	if len(msgs) == 0 {
		t.Error("mock 数据不应为空")
	}
}

func TestMockMessages_HasAllRoles(t *testing.T) {
	msgs := mockMessages()
	hasUser, hasAsst, hasSys := false, false, false
	for _, m := range msgs {
		switch m.Role {
		case RoleUser:
			hasUser = true
		case RoleAssistant:
			hasAsst = true
		case RoleSystem:
			hasSys = true
		}
	}
	if !hasUser {
		t.Error("mock 数据应包含用户消息")
	}
	if !hasAsst {
		t.Error("mock 数据应包含助手消息")
	}
	if !hasSys {
		t.Error("mock 数据应包含系统消息")
	}
}

func TestMockMessages_HasThinking(t *testing.T) {
	msgs := mockMessages()
	hasThinking := false
	for _, m := range msgs {
		if m.Thinking != "" {
			hasThinking = true
			break
		}
	}
	if !hasThinking {
		t.Error("mock 数据应包含思考过程")
	}
}

func TestMockMessages_HasToolCalls(t *testing.T) {
	msgs := mockMessages()
	hasTools := false
	for _, m := range msgs {
		if len(m.ToolCalls) > 0 {
			hasTools = true
			break
		}
	}
	if !hasTools {
		t.Error("mock 数据应包含工具调用")
	}
}

func TestMockMessages_HasMultipleToolStatuses(t *testing.T) {
	msgs := mockMessages()
	statuses := map[string]bool{}
	for _, m := range msgs {
		for _, tc := range m.ToolCalls {
			statuses[tc.Status] = true
		}
	}
	if len(statuses) < 2 {
		t.Errorf("mock 数据应包含多种工具状态，只有 %d 种", len(statuses))
	}
}

func TestMockMessages_ChineseContent(t *testing.T) {
	msgs := mockMessages()
	hasChinese := false
	for _, m := range msgs {
		for _, r := range m.Content {
			if r >= 0x4e00 && r <= 0x9fff {
				hasChinese = true
				break
			}
		}
	}
	if !hasChinese {
		t.Error("mock 数据应包含中文内容")
	}
}

func TestMockMessages_EnglishContent(t *testing.T) {
	msgs := mockMessages()
	hasEnglish := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "the") || strings.Contains(m.Content, "The") {
			hasEnglish = true
			break
		}
	}
	if !hasEnglish {
		t.Error("mock 数据应包含英文内容")
	}
}

func TestMockSessions_NotEmpty(t *testing.T) {
	sessions := mockSessions()
	if len(sessions) == 0 {
		t.Error("mock 会话数据不应为空")
	}
}

func TestMockSessions_HasActive(t *testing.T) {
	sessions := mockSessions()
	hasActive := false
	for _, s := range sessions {
		if s.Active {
			hasActive = true
			break
		}
	}
	if !hasActive {
		t.Error("mock 会话数据应包含活跃会话")
	}
}

// ─── Role 字符串测试 ───

func TestRoleString(t *testing.T) {
	if RoleUser.String() != "user" {
		t.Error("RoleUser.String() 应为 'user'")
	}
	if RoleAssistant.String() != "assistant" {
		t.Error("RoleAssistant.String() 应为 'assistant'")
	}
	if RoleSystem.String() != "system" {
		t.Error("RoleSystem.String() 应为 'system'")
	}
	if Role(99).String() != "unknown" {
		t.Error("未知 Role 应为 'unknown'")
	}
}
