package prompt

import (
	"fmt"
	"strings"
	"time"

	"devo/internal/core/session"
)

const defaultBasePrompt = "You are a helpful coding assistant. Respond concisely and helpfully."

type Assembler struct {
	basePrompt     string
	skillsProvider SkillsProvider
	memoryProvider MemoryProvider
	dirTreeConfig  DirTreeConfig
}

func NewAssembler() *Assembler {
	return &Assembler{
		basePrompt:    defaultBasePrompt,
		dirTreeConfig: DefaultDirTreeConfig(),
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

func (a *Assembler) SetDirTreeConfig(cfg DirTreeConfig) {
	a.dirTreeConfig = cfg
}

func (a *Assembler) Assemble(sess *session.Session, hasFileChange bool) string {
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

	if dirTree := a.buildDirTree(sess, hasFileChange); dirTree != "" {
		parts = append(parts, dirTree)
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

func (a *Assembler) buildDirTree(sess *session.Session, hasFileChange bool) string {
	if !hasFileChange && sess.CachedDirectorySummary != nil && sess.CachedDirectorySummary.Valid {
		return sess.CachedDirectorySummary.Content
	}

	changed, err := IsDirTreeChanged(sess.WorkingDirectory, sess.CachedDirectorySummary)
	if err != nil || changed {
		tree, err := GenerateDirTree(sess.WorkingDirectory, a.dirTreeConfig)
		if err != nil {
			return ""
		}
		sess.CachedDirectorySummary = &session.DirectorySummary{
			Content:     tree,
			GeneratedAt: time.Now(),
			Valid:       true,
		}
		return tree
	}

	if sess.CachedDirectorySummary != nil {
		return sess.CachedDirectorySummary.Content
	}

	return ""
}

func (a *Assembler) buildDynamicInfo(sess *session.Session) string {
	parts := []string{
		fmt.Sprintf("会话 ID: %s", sess.ID),
		fmt.Sprintf("工作目录: %s", sess.WorkingDirectory),
		fmt.Sprintf("信任级别: %s", sess.TrustLevel),
		fmt.Sprintf("工具调用上限: %d", sess.ToolCallLimit),
		fmt.Sprintf("当前工具调用计数: %d", sess.ToolCallCount),
	}

	if len(sess.ApprovalPolicy) > 0 {
		var policies []string
		for opType, policy := range sess.ApprovalPolicy {
			policies = append(policies, fmt.Sprintf("  %s: %s", opType, policy))
		}
		parts = append(parts, "审批策略:\n"+strings.Join(policies, "\n"))
	}

	return strings.Join(parts, "\n")
}
