package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devo/internal/core/session"
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
		ToolCallLimit:    50,
		ToolCallCount:    0,
	}

	assembler := NewAssembler()
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

func TestAssembler_Assemble_WithSystemPromptOverride(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:                   "sess-test-1",
		WorkingDirectory:     tmpDir,
		SystemPromptOverride: "请使用中文回答所有问题",
		TrustLevel:           "session_trust",
		ToolCallLimit:        50,
	}

	assembler := NewAssembler()
	result := assembler.Assemble(sess)

	if !strings.Contains(result, "请使用中文回答所有问题") {
		t.Error("expected system prompt override in assembled result")
	}
}

func TestAssembler_Assemble_WithAgentsMD(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "agents.md"), []byte("所有 Python 文件必须包含文件头注释"), 0644)

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()
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
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()
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

	assembler := NewAssembler()
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
		ID:                   "sess-order",
		WorkingDirectory:     tmpDir,
		SystemPromptOverride: "OVERRIDE_TEXT",
		TrustLevel:           "session_trust",
		ToolCallLimit:        50,
	}

	assembler := NewAssembler()
	result := assembler.Assemble(sess)

	baseIdx := strings.Index(result, "You are Devo")
	overrideIdx := strings.Index(result, "OVERRIDE_TEXT")
	agentsIdx := strings.Index(result, "AGENTS_MD_RULE")
	dynamicIdx := strings.Index(result, "会话 ID")

	if baseIdx < 0 || overrideIdx < 0 || agentsIdx < 0 || dynamicIdx < 0 {
		t.Fatal("expected all sections to be present")
	}

	if !(baseIdx < overrideIdx) {
		t.Error("base prompt should come before override")
	}
	if !(overrideIdx < agentsIdx) {
		t.Error("override should come before agents.md")
	}
	if !(agentsIdx < dynamicIdx) {
		t.Error("agents.md should come before dynamic info")
	}
}

func TestAssembler_SetBasePrompt(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()
	assembler.SetBasePrompt("You are a Python expert. Write clean code.")

	result := assembler.Assemble(sess)

	if !strings.Contains(result, "You are a Python expert") {
		t.Error("expected custom base prompt in assembled result")
	}
	if strings.Contains(result, "You are Devo") {
		t.Error("expected default base prompt to be replaced")
	}
}

type mockSkillsProvider struct {
	prompt string
}

func (m *mockSkillsProvider) GetActiveSkillsPrompt(activeSkillNames []string) string {
	return m.prompt
}

func TestAssembler_Assemble_WithSkillsProvider(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()
	assembler.SetSkillsProvider(&mockSkillsProvider{prompt: "你是一个 Vue 专家，请使用 Composition API"})

	result := assembler.Assemble(sess)

	if !strings.Contains(result, "Composition API") {
		t.Error("expected skills prompt in assembled result")
	}
}

type mockMemoryProvider struct {
	memory string
}

func (m *mockMemoryProvider) GetRelevantMemories(workingDir, sessionID string) string {
	return m.memory
}

func TestAssembler_Assemble_WithMemoryProvider(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-mem",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()
	assembler.SetMemoryProvider(&mockMemoryProvider{memory: "用户偏好使用 TypeScript 而非 JavaScript"})

	result := assembler.Assemble(sess)

	if !strings.Contains(result, "TypeScript") {
		t.Error("expected memory content in assembled result")
	}
}

func TestAssembler_Assemble_EmptyProviders(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()
	assembler.SetSkillsProvider(&mockSkillsProvider{prompt: ""})
	assembler.SetMemoryProvider(&mockMemoryProvider{memory: ""})

	result := assembler.Assemble(sess)

	if strings.Contains(result, "Composition API") {
		t.Error("empty skills should not appear in result")
	}
	if strings.Contains(result, "TypeScript") {
		t.Error("empty memory should not appear in result")
	}
}
