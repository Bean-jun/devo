package prompt

import "devo/internal/core/skills"

type SkillsProvider interface {
	GetActiveSkillsPrompt() string
	IsSkillAllowed(name string) bool
	GetSkill(name string) (*skills.Skill, error)
}

type MemoryProvider interface {
	GetRelevantMemories(workingDir, sessionID string) string
}
