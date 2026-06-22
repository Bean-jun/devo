package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestGenerateDirTree_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Project"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "internal"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "internal", "app.go"), []byte("package internal"), 0644)

	config := DirTreeConfig{MaxDepth: 3, MaxFiles: 200}
	tree, err := GenerateDirTree(tmpDir, config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(tree, "main.go") {
		t.Error("expected tree to contain main.go")
	}
	if !strings.Contains(tree, "README.md") {
		t.Error("expected tree to contain README.md")
	}
	if !strings.Contains(tree, "internal/") {
		t.Error("expected tree to contain internal/")
	}
	if !strings.Contains(tree, "app.go") {
		t.Error("expected tree to contain app.go")
	}
}

func TestGenerateDirTree_ExcludesGit(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, ".git"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".git", "config"), []byte("git config"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	config := DirTreeConfig{MaxDepth: 3, MaxFiles: 200}
	tree, err := GenerateDirTree(tmpDir, config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if strings.Contains(tree, ".git") {
		t.Error("expected tree to exclude .git directory")
	}
	if !strings.Contains(tree, "main.go") {
		t.Error("expected tree to contain main.go")
	}
}

func TestGenerateDirTree_ExcludesDotFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("secret"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	config := DirTreeConfig{MaxDepth: 3, MaxFiles: 200}
	tree, err := GenerateDirTree(tmpDir, config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if strings.Contains(tree, ".env") {
		t.Error("expected tree to exclude hidden .env file")
	}
	if !strings.Contains(tree, "main.go") {
		t.Error("expected tree to contain main.go")
	}
}

func TestGenerateDirTree_IncludesDevo(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, ".devo"), 0755)
	os.WriteFile(filepath.Join(tmpDir, ".devo", "sessions"), []byte(""), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	config := DirTreeConfig{MaxDepth: 3, MaxFiles: 200}
	tree, err := GenerateDirTree(tmpDir, config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(tree, ".devo/") {
		t.Error("expected tree to include .devo directory")
	}
}

func TestGenerateDirTree_MaxDepth(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "level1", "level2", "level3"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "level1", "level2", "level3", "deep.txt"), []byte("deep"), 0644)

	config := DirTreeConfig{MaxDepth: 1, MaxFiles: 200}
	tree, err := GenerateDirTree(tmpDir, config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if strings.Contains(tree, "deep.txt") {
		t.Error("with max_depth=1, deep.txt should not appear")
	}
	if !strings.Contains(tree, "level1/") {
		t.Error("expected tree to contain level1/")
	}
}

func TestGenerateDirTree_MaxFiles(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(tmpDir, string(rune('a'+i))+".txt"), []byte("data"), 0644)
	}

	config := DirTreeConfig{MaxDepth: 3, MaxFiles: 3}
	tree, err := GenerateDirTree(tmpDir, config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(tree, "输出被截断") {
		t.Error("expected tree to be truncated at max_files=3")
	}
}

func TestGenerateDirTree_DirsBeforeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "z.txt"), []byte("z"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "a_dir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("a"), 0644)

	config := DirTreeConfig{MaxDepth: 3, MaxFiles: 200}
	tree, err := GenerateDirTree(tmpDir, config)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	aDirIdx := strings.Index(tree, "a_dir/")
	aTxtIdx := strings.Index(tree, "a.txt")
	if aDirIdx < 0 || aTxtIdx < 0 {
		t.Fatal("expected both a_dir/ and a.txt in tree")
	}
	if aDirIdx > aTxtIdx {
		t.Error("expected directories to appear before files")
	}
}

func TestIsDirTreeChanged_NilSummary(t *testing.T) {
	tmpDir := t.TempDir()

	changed, err := IsDirTreeChanged(tmpDir, nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when cached summary is nil")
	}
}

func TestIsDirTreeChanged_InvalidSummary(t *testing.T) {
	tmpDir := t.TempDir()

	summary := &session.DirectorySummary{
		Content:     "old tree",
		GeneratedAt: time.Now(),
		Valid:       false,
	}

	changed, err := IsDirTreeChanged(tmpDir, summary)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when cached summary is invalid")
	}
}

func TestIsDirTreeChanged_ValidCachedSummary(t *testing.T) {
	tmpDir := t.TempDir()

	summary := &session.DirectorySummary{
		Content:     "tree content",
		GeneratedAt: time.Now().Add(1 * time.Hour),
		Valid:       true,
	}

	changed, err := IsDirTreeChanged(tmpDir, summary)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if changed {
		t.Error("expected changed=false when cached summary is newer than directory")
	}
}

func TestIsDirTreeChanged_StaleCachedSummary(t *testing.T) {
	tmpDir := t.TempDir()

	summary := &session.DirectorySummary{
		Content:     "tree content",
		GeneratedAt: time.Now().Add(-1 * time.Hour),
		Valid:       true,
	}

	time.Sleep(10 * time.Millisecond)

	os.WriteFile(filepath.Join(tmpDir, "new.txt"), []byte("new"), 0644)

	changed, err := IsDirTreeChanged(tmpDir, summary)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !changed {
		t.Error("expected changed=true when directory modified after cached summary")
	}
}

func TestNewDirectorySummary(t *testing.T) {
	content := "test directory tree"
	summary := NewDirectorySummary(content)

	if summary.Content != content {
		t.Errorf("expected content %q, got %q", content, summary.Content)
	}
	if !summary.Valid {
		t.Error("expected Valid to be true")
	}
	if summary.GeneratedAt.IsZero() {
		t.Error("expected GeneratedAt to be set")
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
	result := assembler.Assemble(sess, true)

	if !strings.Contains(result, "You are a helpful coding assistant") {
		t.Error("expected base prompt in assembled result")
	}
	if !strings.Contains(result, "sess-test-1") {
		t.Error("expected session ID in dynamic info")
	}
	if !strings.Contains(result, tmpDir) {
		t.Error("expected working directory in dynamic info")
	}
	if !strings.Contains(result, "main.go") {
		t.Error("expected directory tree in assembled result")
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
	result := assembler.Assemble(sess, true)

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
	result := assembler.Assemble(sess, true)

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
	result := assembler.Assemble(sess, true)

	if strings.Contains(result, "agents.md") {
		t.Error("did not expect agents.md reference when file does not exist")
	}
}

func TestAssembler_Assemble_DirTreeCaching(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()

	result1 := assembler.Assemble(sess, true)
	if !strings.Contains(result1, "main.go") {
		t.Error("first call should generate fresh dir tree")
	}

	if sess.CachedDirectorySummary == nil {
		t.Fatal("expected cached summary to be set after first call")
	}
	if !sess.CachedDirectorySummary.Valid {
		t.Error("expected cached summary to be valid")
	}

	result2 := assembler.Assemble(sess, false)
	if !strings.Contains(result2, "main.go") {
		t.Error("cached result should still contain main.go")
	}

	if result1 != result2 {
		t.Error("expected same result when using cache (no file change)")
	}
}

func TestAssembler_Assemble_DirTreeRegenerationOnFileChange(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)

	sess := &session.Session{
		ID:               "sess-test-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()

	result1 := assembler.Assemble(sess, true)

	os.WriteFile(filepath.Join(tmpDir, "newfile.go"), []byte("package newfile"), 0644)

	result2 := assembler.Assemble(sess, true)

	if !strings.Contains(result1, "main.go") {
		t.Error("first result should contain main.go")
	}
	if strings.Contains(result1, "newfile.go") {
		t.Error("first result should not contain newfile.go")
	}
	if !strings.Contains(result2, "newfile.go") {
		t.Error("second result should contain newfile.go after file change")
	}
}

func TestAssembler_Assemble_DynamicInfo(t *testing.T) {
	tmpDir := t.TempDir()

	sess := &session.Session{
		ID:               "sess-dynamic",
		WorkingDirectory: tmpDir,
		TrustLevel:       "full_trust",
		ToolCallLimit:    100,
		ToolCallCount:    5,
		ApprovalPolicy: map[string]string{
			"file_write": "always_ask",
			"exec_cmd":   "session_trust",
		},
	}

	assembler := NewAssembler()
	result := assembler.Assemble(sess, true)

	if !strings.Contains(result, "sess-dynamic") {
		t.Error("expected session ID in dynamic info")
	}
	if !strings.Contains(result, "full_trust") {
		t.Error("expected trust level in dynamic info")
	}
	if !strings.Contains(result, "100") {
		t.Error("expected tool call limit in dynamic info")
	}
	if !strings.Contains(result, "5") {
		t.Error("expected tool call count in dynamic info")
	}
	if !strings.Contains(result, "always_ask") {
		t.Error("expected approval policy in dynamic info")
	}
	if !strings.Contains(result, "session_trust") {
		t.Error("expected approval policy in dynamic info")
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
	result := assembler.Assemble(sess, true)

	baseIdx := strings.Index(result, "You are a helpful coding assistant")
	overrideIdx := strings.Index(result, "OVERRIDE_TEXT")
	agentsIdx := strings.Index(result, "AGENTS_MD_RULE")
	dirTreeIdx := strings.Index(result, "工作目录文件结构")
	dynamicIdx := strings.Index(result, "会话 ID")

	if baseIdx < 0 || overrideIdx < 0 || agentsIdx < 0 || dirTreeIdx < 0 || dynamicIdx < 0 {
		t.Fatal("expected all sections to be present")
	}

	if !(baseIdx < overrideIdx) {
		t.Error("base prompt should come before override")
	}
	if !(overrideIdx < agentsIdx) {
		t.Error("override should come before agents.md")
	}
	if !(agentsIdx < dirTreeIdx) {
		t.Error("agents.md should come before directory tree")
	}
	if !(dirTreeIdx < dynamicIdx) {
		t.Error("directory tree should come before dynamic info")
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

	result := assembler.Assemble(sess, true)

	if !strings.Contains(result, "You are a Python expert") {
		t.Error("expected custom base prompt in assembled result")
	}
	if strings.Contains(result, "You are a helpful coding assistant") {
		t.Error("expected default base prompt to be replaced")
	}
}

func TestAssembler_SetDirTreeConfig(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "level1", "level2"), 0755)

	sess := &session.Session{
		ID:               "sess-1",
		WorkingDirectory: tmpDir,
		TrustLevel:       "session_trust",
		ToolCallLimit:    50,
	}

	assembler := NewAssembler()
	assembler.SetDirTreeConfig(DirTreeConfig{MaxDepth: 1, MaxFiles: 200})

	result := assembler.Assemble(sess, true)

	if strings.Contains(result, "level2") {
		t.Error("with max_depth=1, level2 should not appear")
	}
	if !strings.Contains(result, "level1/") {
		t.Error("expected level1 to appear")
	}
}

type mockSkillsProvider struct {
	prompt string
}

func (m *mockSkillsProvider) GetActiveSkillsPrompt() string {
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

	result := assembler.Assemble(sess, true)

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

	result := assembler.Assemble(sess, true)

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

	result := assembler.Assemble(sess, true)

	if strings.Contains(result, "Composition API") {
		t.Error("empty skills should not appear in result")
	}
	if strings.Contains(result, "TypeScript") {
		t.Error("empty memory should not appear in result")
	}
}
