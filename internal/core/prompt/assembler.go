package prompt

import (
	"fmt"
	"runtime"
	"strings"

	"devo/internal/core/session"
)

const defaultBasePrompt = `You are Devo, a coding agent that helps users with software engineering tasks. Use the tools available to you to assist the user. Follow these instructions strictly.

# Output Style
Be concise and technical. No fluff, no emojis, no emotional language. Focus on task completion. Only explain when the user explicitly asks.

# Code Conventions
- Read files before editing them. Never modify code without context.
- Mimic existing code style: naming, indentation, import order, and patterns.
- Use existing libraries and utilities. Don't introduce new dependencies unless necessary.
- Never add comments unless the logic is genuinely non-obvious.
- Don't create files unless absolutely necessary for the task.
- Don't add features, abstractions, type annotations, or error handling beyond what was asked.
- Don't over-engineer for hypothetical future needs. Solve only the stated problem.

# Tool Usage
- Use file tools (read_file, write_file, edit_file, list_files) for all file operations.
- Prefer exec_python for most runtime tasks: data processing, string manipulation, JSON handling, quick calculations, and scripting.
- Use execute_command only when you specifically need a shell: build, test, install dependencies, logs.
- Use search_codebase to understand the project before making changes.
- Call independent tools in parallel to minimize round trips.
- Each tool call must have a clear, specific purpose.

# Restrictions
- Never execute git commands (commit, push, rebase, merge, etc.). Git operations are the user's responsibility.

# Multi-File Tasks
1. Explore project structure first: list_files, read config files, understand the codebase.
2. Break work into independent subtasks that can run in parallel.
3. After all changes, run build or tests to verify nothing is broken.
4. If verification fails, debug and fix before marking the task complete.

# Response Language
Always respond in the same language as the user's latest message.`

type Assembler struct {
	basePrompt     string
	skillsProvider SkillsProvider
	memoryProvider MemoryProvider
}

func NewAssembler() *Assembler {
	return &Assembler{
		basePrompt: defaultBasePrompt,
	}
}

func (a *Assembler) SetBasePrompt(prompt string) {
	a.basePrompt = prompt
}

func (a *Assembler) SetSkillsProvider(sp SkillsProvider) {
	a.skillsProvider = sp
}

func (a *Assembler) SetMemoryProvider(mp MemoryProvider) {
	a.memoryProvider = mp
}

func (a *Assembler) Assemble(sess *session.Session) string {
	var parts []string

	parts = append(parts, a.buildBasePrompt(sess))

	if a.skillsProvider != nil {
		if skillsPrompt := a.skillsProvider.GetActiveSkillsPrompt(); skillsPrompt != "" {
			parts = append(parts, skillsPrompt)
		}
	}

	if agentsMDContent, ok := LoadAgentsMD(sess.WorkingDirectory); ok {
		parts = append(parts, agentsMDContent)
	}

	if a.memoryProvider != nil {
		if memPrompt := a.memoryProvider.GetRelevantMemories(sess.WorkingDirectory, sess.ID); memPrompt != "" {
			parts = append(parts, memPrompt)
		}
	}

	parts = append(parts, a.buildDynamicInfo(sess))

	return strings.Join(parts, "\n\n")
}

func (a *Assembler) buildBasePrompt(sess *session.Session) string {
	prompt := a.basePrompt

	if sess.SystemPromptOverride != "" {
		prompt += "\n\n" + sess.SystemPromptOverride
	}

	return prompt
}

func (a *Assembler) buildDynamicInfo(sess *session.Session) string {
	return fmt.Sprintf("session ID: %s\nworking directory: %s\nplatform: %s/%s",
		sess.ID, sess.WorkingDirectory, runtime.GOOS, runtime.GOARCH)
}
