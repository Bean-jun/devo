package skills

import (
	"context"
	"fmt"
	"strings"
	"time"

	"devo/internal/core/session"
	"devo/internal/taskexec/llmclient"
)

const solidifyPrompt = `You are an expert at analyzing completed coding sessions and extracting reusable skills. 

Given the full conversation history of a coding session, analyze it and produce a SKILL.md file that captures the reusable knowledge.

The SKILL.md MUST follow this EXACT format:

---
name: <Skill Name - a short, descriptive name, one or two words>
description: <Brief description of when this skill should be used - one sentence>
---

# <Skill Name>

## Description
<Describe what this skill does and when it should be activated>

## Instructions
<Step-by-step instructions for the AI agent to follow when this skill is activated>

## Usage
<What situations trigger this skill - what problems does it solve>

## Examples (optional)
<Input/output examples to show expected behavior>

Rules:
- MUST start with --- YAML frontmatter containing name and description.
- Be specific and actionable. The skill should be directly usable by an AI agent.
- Focus on the reusable patterns, not the specific task details.
- Keep it concise but complete.
- If the session doesn't contain any reusable skill, respond with "NO_SKILL".
- Use the same language as the conversation.

Here is the conversation history:`

type Solidifier struct {
	llmClient     llmclient.Client
	skillsManager *Manager
	sessionStore  session.SessionStore
}

func NewSolidifier(llmClient llmclient.Client, skillsManager *Manager, sessionStore session.SessionStore) *Solidifier {
	return &Solidifier{
		llmClient:     llmClient,
		skillsManager: skillsManager,
		sessionStore:  sessionStore,
	}
}

func (s *Solidifier) SolidifySession(ctx context.Context, sessionID string) (*SolidifyResult, error) {
	msgs, _, err := s.sessionStore.GetMessages(sessionID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	if len(msgs) == 0 {
		return &SolidifyResult{NoSkill: true}, nil
	}

	convText := buildConversationText(msgs)
	if len(convText) < 100 {
		return &SolidifyResult{NoSkill: true}, nil
	}

	prompt := solidifyPrompt + "\n\n" + convText

	systemMsg := []session.Message{
		{
			ID:        session.GenerateID("msg"),
			Role:      session.RoleUser,
			Content:   prompt,
			CreatedAt: time.Now(),
		},
	}

	result, err := s.llmClient.Complete(ctx, systemMsg, "You are a skill extraction assistant.")
	if err != nil {
		return nil, fmt.Errorf("llm complete: %w", err)
	}

	rawContent := result.Text
	if strings.Contains(rawContent, "NO_SKILL") {
		return &SolidifyResult{NoSkill: true}, nil
	}

	skillContent := cleanSkillContent(rawContent)
	fm := parseFrontmatter(skillContent)
	skillName := fm["name"]
	if skillName == "" {
		skillName = extractSkillName(skillContent, fmt.Sprintf("skill-%s", sessionID[:8]))
	}

	return &SolidifyResult{
		SkillName: skillName,
		Content:   skillContent,
		NoSkill:   false,
	}, nil
}

func buildConversationText(msgs []session.Message) string {
	var parts []string
	for _, msg := range msgs {
		switch msg.Role {
		case session.RoleUser:
			parts = append(parts, fmt.Sprintf("User: %s", msg.Content))
		case session.RoleAssistant:
			if msg.Content != "" {
				parts = append(parts, fmt.Sprintf("Assistant: %s", msg.Content))
			}
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					parts = append(parts, fmt.Sprintf("Tool Call: %s(%v)", tc.ToolName, tc.Params))
				}
			}
		case session.RoleTool:
			parts = append(parts, fmt.Sprintf("Tool Result: %s", msg.Content))
		case session.RoleSystem:
			parts = append(parts, fmt.Sprintf("System: %s", msg.Content))
		}
	}
	return strings.Join(parts, "\n")
}

func cleanSkillContent(raw string) string {
	raw = strings.TrimSpace(raw)

	if idx := strings.Index(raw, "```"); idx >= 0 {
		start := strings.Index(raw[idx:], "\n")
		if start >= 0 {
			raw = raw[idx+start+1:]
		}
		end := strings.LastIndex(raw, "```")
		if end >= 0 {
			raw = raw[:end]
		}
	}

	return strings.TrimSpace(raw)
}

type SolidifyResult struct {
	SkillName string `json:"skill_name,omitempty"`
	Content   string `json:"content,omitempty"`
	NoSkill   bool   `json:"no_skill"`
}
