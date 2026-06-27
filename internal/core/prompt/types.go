package prompt

type SkillsProvider interface {
	GetActiveSkillsPrompt() string
}

type MemoryProvider interface {
	GetRelevantMemories(workingDir, sessionID string) string
}
