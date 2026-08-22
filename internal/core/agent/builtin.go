package agent

import "devo/internal/core/prompt"

var BuiltinAgents = []Config{
	{
		ID:           "devo-default",
		Name:         "Devo",
		Description:  "General-purpose coding assistant with full code read/write and execution capabilities",
		SystemPrompt: prompt.DefaultSystemPrompt(),
		ModelID:      "",
		Tools:        nil,
		Skills:       nil,
		Builtin:      true,
	},
	{
		ID:           "code-reviewer",
		Name:         "Code Reviewer",
		Description:  "Code review expert, read-only analysis, no code modifications",
		SystemPrompt: prompt.CodeReviewerPrompt(),
		ModelID:      "",
		Tools:        []string{"read_file", "glob", "list_files", "search_codebase"},
		Skills:       nil,
		Builtin:      true,
	},
	{
		ID:           "architect",
		Name:         "Architect",
		Description:  "Architecture analysis and technical design, no code execution",
		SystemPrompt: prompt.ArchitectPrompt(),
		ModelID:      "",
		Tools:        []string{"read_file", "glob", "list_files", "search_codebase"},
		Skills:       nil,
		Builtin:      true,
	},
	{
		ID:           "test-writer",
		Name:         "Test Writer",
		Description:  "Test case writing expert, focused on code quality and edge cases",
		SystemPrompt: prompt.TestWriterPrompt(),
		ModelID:      "",
		Tools:        []string{"read_file", "write_file", "edit_file", "glob", "list_files", "search_codebase", "exec_python"},
		Skills:       nil,
		Builtin:      true,
	},
}
