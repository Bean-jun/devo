package tools

import (
	"context"
	"fmt"
	"strings"

	"devo/internal/core/skills"
)

type UseSkillTool struct {
	manager *skills.Manager
	loaded  map[string]bool
}

func NewUseSkillTool(manager *skills.Manager) *UseSkillTool {
	return &UseSkillTool{
		manager: manager,
		loaded:  make(map[string]bool),
	}
}

func (t *UseSkillTool) Name() string {
	return "use_skill"
}

func (t *UseSkillTool) Description() string {
	return `Execute a skill within the main conversation.

Usage:
- Invoke skills using this tool with the skill name only (no arguments).
- When you invoke a skill, the skill's prompt will expand and provide detailed instructions.
- Examples:
    command: "pdf" - invoke the pdf skill
    command: "xlsx" - invoke the xlsx skill
- MANDATORY for creating skills - MUST be invoked IMMEDIATELY when user wants to create/add any skill.
- Only use skills listed in <available_skills>.
- Do not invoke a skill that is already running.`
}

func (t *UseSkillTool) RiskLevel() RiskLevel {
	return RiskLevelNone
}

func (t *UseSkillTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"skill_name": map[string]interface{}{
				"type":        "string",
				"description": "The name of the skill to load (e.g., 'Python Expert', 'Go Expert'). Use the exact name from the Available Skills catalog.",
			},
		},
		"required": []string{"skill_name"},
	}
}

func (t *UseSkillTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	skillName, ok := params["skill_name"].(string)
	if !ok || skillName == "" {
		return fmt.Errorf("missing required parameter: skill_name")
	}

	if t.loaded[skillName] {
		w.WriteDone(true, fmt.Sprintf("Skill '%s' is already loaded. Its instructions are already in the conversation context.", skillName))
		return nil
	}

	skill, err := t.manager.GetSkill(skillName)
	if err != nil {
		return fmt.Errorf("skill not found: %s. Make sure the skill name matches exactly one of the available skills.", skillName)
	}

	if !skill.Enabled {
		return fmt.Errorf("skill '%s' is currently disabled", skillName)
	}

	t.loaded[skillName] = true

	var result strings.Builder

	result.WriteString(fmt.Sprintf("[Skill Loaded]\nName: %s\nLocation: %s\n\n", skill.Name, skill.Location))

	if skill.Instructions != "" {
		result.WriteString("[Instructions]\n\n")
		result.WriteString(skill.Instructions)
		result.WriteString("\n")
	}

	scripts, references, assets := t.manager.ListSkillResources(skill.Location)

	if len(scripts) > 0 || len(references) > 0 || len(assets) > 0 {
		result.WriteString("\n[Available Resources]\n")
		result.WriteString("Use the existing tools (read_file, exec_python, etc.) to access these:\n\n")

		if len(scripts) > 0 {
			result.WriteString("Scripts:\n")
			for _, s := range scripts {
				result.WriteString(fmt.Sprintf("  - %s/%s\n", skill.Location, s))
			}
			result.WriteString("\n")
		}
		if len(references) > 0 {
			result.WriteString("References:\n")
			for _, r := range references {
				result.WriteString(fmt.Sprintf("  - %s/%s\n", skill.Location, r))
			}
			result.WriteString("\n")
		}
		if len(assets) > 0 {
			result.WriteString("Assets:\n")
			for _, a := range assets {
				result.WriteString(fmt.Sprintf("  - %s/%s\n", skill.Location, a))
			}
			result.WriteString("\n")
		}
	}

	w.WriteDone(true, result.String())
	return nil
}

func (t *UseSkillTool) Reset() {
	t.loaded = make(map[string]bool)
}
