package prompt

type SkillsProvider interface {
	GetActiveSkillsPrompt(activeSkillNames []string) string
}

type MemoryProvider interface {
	GetRelevantMemories(workingDir, sessionID string) string
}
