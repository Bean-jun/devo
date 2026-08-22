package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devo/internal/core/session"
	"devo/internal/core/skills"
)

func TestLoadAgentsMD_Found(t *testing.T) {
	tmpDir := t.TempDir()
	content := "所有新建的 Python 文件必须包含文件头注释"
	os.WriteFile(filepath.Join(tmpDir, "agents.md"), []byte(content), 0644)

	result, ok := LoadAgentsMD(tmpDir)
	if !ok {
		t.Fatal("expected agents.md to be found")
	}
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestLoadAgentsMD_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, ok := LoadAgentsMD(tmpDir)
	if ok {
		t.Fatal("expected agents.md to not be found")
	}
}

func TestLoadAgentsMD_DevoRules(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, ".devo"), 0755)
	content := "所有 Go 文件必须使用 tab 缩进"
	os.WriteFile(filepath.Join(tmpDir, ".devo", "rules.md"), []byte(content), 0644)

	result, ok := LoadAgentsMD(tmpDir)
	if !ok {
		t.Fatal("expected .devo/rules.md to be found")
	}
	if result != content {
		t.Errorf("expected %q, got %q", content, result)
	}
}

func TestLoadAgentsMD_AgentsMDPriority(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "agents.md"), []byte("agents.md content"), 0644)
	os.Mkdir(filepath.Join(tmpDir, ".devo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".devo", "rules.md"), []byte("rules.md content"), 0644)

	result, ok := LoadAgentsMD(tmpDir)
	if !ok {
		t.Fatal("expected a file to be found")
	}
	if result != "agents.md content" {
		t.Errorf("expected agents.md to take priority, got %q", result)
	}
}

func TestAssembler_Assemble_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
	}

	assembler := NewAssembler(DefaultSystemPrompt())
	result := assembler.Assemble(sess)

	if !strings.Contains(result, "You are Devo") {
		t.Error("expected base prompt in assembled result")
	}
	if !strings.Contains(result, "sess-test-1") {
		t.Error("expected session ID in dynamic info")
	}
	if !strings.Contains(result, tmpDir) {
		t.Error("expected working directory in dynamic info")
	}
}

func TestAssembler_Assemble_WithCustomSystemPrompt(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
	}

	assembler := NewAssembler("请使用中文回答所有问题")
	result := assembler.Assemble(sess)

	if !strings.Contains(result, "请使用中文回答所有问题") {
		t.Error("expected custom system prompt in assembled result")
	}
}

func TestAssembler_Assemble_WithAgentsMD(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "agents.md"), []byte("所有 Python 文件必须包含文件头注释"), 0644)

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
	}

	assembler := NewAssembler(DefaultSystemPrompt())
	result := assembler.Assemble(sess)

	if !strings.Contains(result, "所有 Python 文件必须包含文件头注释") {
		t.Error("expected agents.md content in assembled result")
	}
}

func TestAssembler_Assemble_WithoutAgentsMD(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
	}

	assembler := NewAssembler(DefaultSystemPrompt())
	result := assembler.Assemble(sess)

	if strings.Contains(result, "agents.md") {
		t.Error("did not expect agents.md reference when file does not exist")
	}
}

func TestAssembler_Assemble_DynamicInfo(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-dynamic",
		WorkingDirectory: tmpDir,
	}

	assembler := NewAssembler(DefaultSystemPrompt())
	result := assembler.Assemble(sess)

	if !strings.Contains(result, "sess-dynamic") {
		t.Error("expected session ID in dynamic info")
	}
	if !strings.Contains(result, tmpDir) {
		t.Error("expected working directory in dynamic info")
	}
}

func TestAssembler_Assemble_Ordering(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "agents.md"), []byte("AGENTS_MD_RULE"), 0644)

	sess := &session.Session{
		ID:               "sess-order",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
	}

	assembler := NewAssembler("CUSTOM_SYSTEM_PROMPT")
	result := assembler.Assemble(sess)

	baseIdx := strings.Index(result, "CUSTOM_SYSTEM_PROMPT")
	agentsIdx := strings.Index(result, "AGENTS_MD_RULE")
	dynamicIdx := strings.Index(result, "session ID")

	if baseIdx < 0 || agentsIdx < 0 || dynamicIdx < 0 {
		t.Fatal("expected all sections to be present")
	}

	if !(baseIdx < agentsIdx) {
		t.Error("system prompt should come before agents.md")
	}
	if !(agentsIdx < dynamicIdx) {
		t.Error("agents.md should come before dynamic info")
	}
}

func TestAssembler_Assemble_WithSkillsProvider(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
	}

	assembler := NewAssembler(DefaultSystemPrompt())
	assembler.SetSkillsProvider(&mockSkillsProvider{})

	result := assembler.Assemble(sess)

	if !strings.Contains(result, "ACTIVE_SKILLS_CONTENT") {
		t.Error("expected skills provider content in assembled result")
	}
}

func TestAssembler_Assemble_WithMemoryProvider(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
	}

	assembler := NewAssembler(DefaultSystemPrompt())
	assembler.SetMemoryProvider(&mockMemoryProvider{})

	result := assembler.Assemble(sess)

	if !strings.Contains(result, "RELEVANT_MEMORIES_CONTENT") {
		t.Error("expected memory provider content in assembled result")
	}
}

func TestDefaultSystemPrompt(t *testing.T) {
	prompt := DefaultSystemPrompt()
	if prompt == "" {
		t.Error("expected non-empty default system prompt")
	}
	if !strings.Contains(prompt, "You are Devo") {
		t.Error("expected default prompt to contain 'You are Devo'")
	}
}

func TestCodeReviewerPrompt(t *testing.T) {
	prompt := CodeReviewerPrompt()
	if prompt == "" {
		t.Error("expected non-empty code reviewer prompt")
	}
	if !strings.Contains(prompt, "Code Reviewer") {
		t.Error("expected prompt to contain 'Code Reviewer'")
	}
	if !strings.Contains(prompt, "read-only") {
		t.Error("expected prompt to mention read-only constraint")
	}
}

func TestArchitectPrompt(t *testing.T) {
	prompt := ArchitectPrompt()
	if prompt == "" {
		t.Error("expected non-empty architect prompt")
	}
	if !strings.Contains(prompt, "Architect") {
		t.Error("expected prompt to contain 'Architect'")
	}
	if !strings.Contains(prompt, "read-only") {
		t.Error("expected prompt to mention read-only constraint")
	}
}

func TestTestWriterPrompt(t *testing.T) {
	prompt := TestWriterPrompt()
	if prompt == "" {
		t.Error("expected non-empty test writer prompt")
	}
	if !strings.Contains(prompt, "Test Writer") {
		t.Error("expected prompt to contain 'Test Writer'")
	}
	if !strings.Contains(prompt, "test code") {
		t.Error("expected prompt to mention test code")
	}
}

func TestNewAssembler_EmptyPrompt(t *testing.T) {
	assembler := NewAssembler("")
	if assembler.systemPrompt != DefaultSystemPrompt() {
		t.Error("expected default system prompt when empty string is provided")
	}
}

func TestAssembler_SetSystemPrompt(t *testing.T) {
	assembler := NewAssembler(DefaultSystemPrompt())
	assembler.SetSystemPrompt("New prompt")

	if assembler.systemPrompt != "New prompt" {
		t.Errorf("expected 'New prompt', got %q", assembler.systemPrompt)
	}
}

type mockSkillsProvider struct{}

func (m *mockSkillsProvider) GetActiveSkillsPrompt() string {
	return "ACTIVE_SKILLS_CONTENT"
}

func (m *mockSkillsProvider) IsSkillAllowed(name string) bool {
	return true
}

func (m *mockSkillsProvider) GetSkill(name string) (*skills.Skill, error) {
	return nil, nil
}

type mockMemoryProvider struct{}

func (m *mockMemoryProvider) GetRelevantMemories(workingDir, sessionID string) string {
	return "RELEVANT_MEMORIES_CONTENT"
}
