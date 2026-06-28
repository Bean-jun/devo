package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestSkill(t *testing.T, dir, name, frontmatterName, description, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("create skill dir: %v", err)
	}
	fmName := frontmatterName
	if fmName == "" {
		fmName = name
	}
	fmDesc := description
	if fmDesc == "" {
		fmDesc = name
	}
	fullContent := "---\nname: " + fmName + "\ndescription: " + fmDesc + "\n---\n\n" + content
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(fullContent), 0644); err != nil {
		t.Fatalf("write skill file: %v", err)
	}
}

func TestManager_ScanGlobalSkills(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	createTestSkill(t, globalDir, "python-expert", "Python Expert", "Python coding best practices", "# Python Expert\n\nAlways use type hints.")
	createTestSkill(t, globalDir, "go-expert", "Go Expert", "Go coding conventions", "# Go Expert\n\nUse tab indentation.")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	skills := mgr.GetAllSkills()
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}

	names := make(map[string]bool)
	for _, s := range skills {
		names[s.Name] = true
	}
	if !names["Python Expert"] {
		t.Error("expected 'Python Expert' skill")
	}
	if !names["Go Expert"] {
		t.Error("expected 'Go Expert' skill")
	}
}

func TestManager_ProjectSkillsOverrideGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")
	projectDir := filepath.Join(tmpDir, ".devo", "skills")

	createTestSkill(t, globalDir, "common", "Common", "Global common skill", "# Global Common\n\nGlobal instructions.")
	createTestSkill(t, projectDir, "common", "Common", "Project common skill", "# Project Common\n\nProject-specific instructions.")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	skill, err := mgr.GetSkill("Common")
	if err != nil {
		t.Fatalf("GetSkill: %v", err)
	}

	if skill.Source != SourceProject {
		t.Errorf("expected source 'project', got '%s'", skill.Source)
	}
	if !strings.Contains(skill.Instructions, "Project-specific instructions") {
		t.Errorf("expected project instructions, got '%s'", skill.Instructions)
	}
}

func TestManager_GetActiveSkillsPrompt(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	createTestSkill(t, globalDir, "python-expert", "Python Expert", "Python coding best practices", "# Python Expert\n\nAlways use type hints.")
	createTestSkill(t, globalDir, "go-expert", "Go Expert", "Go coding conventions", "# Go Expert\n\nUse tab indentation.")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	prompt := mgr.GetActiveSkillsPrompt()
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "## Available Skills") {
		t.Error("expected '## Available Skills' section in prompt")
	}
	if !strings.Contains(prompt, "use_skill") {
		t.Error("expected 'use_skill' hint in prompt")
	}
	if !strings.Contains(prompt, "Python Expert") {
		t.Error("expected 'Python Expert' in prompt")
	}
	if !strings.Contains(prompt, "Go Expert") {
		t.Error("expected 'Go Expert' in prompt")
	}
	if !strings.Contains(prompt, "Python coding best practices") {
		t.Error("expected description in catalog section")
	}
	if strings.Contains(prompt, "## Active Skills") {
		t.Error("should not contain '## Active Skills' section (instructions loaded via use_skill)")
	}
	if strings.Contains(prompt, "Use type hints") {
		t.Error("should not contain full instructions in catalog")
	}
}

func TestManager_GetActiveSkillsPrompt_Filtered(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	createTestSkill(t, globalDir, "python-expert", "Python Expert", "Python coding best practices", "# Python Expert\n\nUse type hints.")
	createTestSkill(t, globalDir, "go-expert", "Go Expert", "Go coding conventions", "# Go Expert\n\nUse tab indentation.")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	if err := mgr.DisableSkill("Go Expert"); err != nil {
		t.Fatalf("DisableSkill: %v", err)
	}

	prompt := mgr.GetActiveSkillsPrompt()
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "## Available Skills") {
		t.Error("expected '## Available Skills' section in prompt")
	}
	if !strings.Contains(prompt, "Python Expert") {
		t.Error("expected 'Python Expert' in prompt")
	}
	if strings.Contains(prompt, "Go Expert") {
		t.Error("did not expect 'Go Expert' when disabled")
	}
	if strings.Contains(prompt, "Use tab indentation") {
		t.Error("did not expect 'Go Expert' description when disabled")
	}
}

func TestManager_GetActiveSkillsPrompt_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	prompt := mgr.GetActiveSkillsPrompt()
	if prompt != "" {
		t.Errorf("expected empty prompt, got '%s'", prompt)
	}
}

func TestManager_InstallSkill(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")
	sourceDir := filepath.Join(tmpDir, "source-skill")

	createTestSkill(t, sourceDir, "test-skill", "Test Skill", "Test skill description", "# Test Skill\n\nTest instructions.")

	mgr := NewManager(globalDir)

	skillSourceDir := filepath.Join(sourceDir, "test-skill")
	skill, err := mgr.InstallSkill(skillSourceDir)
	if err != nil {
		t.Fatalf("InstallSkill: %v", err)
	}

	if skill.Name != "Test Skill" {
		t.Errorf("expected name 'Test Skill', got '%s'", skill.Name)
	}
	if skill.Description != "Test skill description" {
		t.Errorf("expected description 'Test skill description', got '%s'", skill.Description)
	}
	if skill.Source != SourceGlobal {
		t.Errorf("expected source 'global', got '%s'", skill.Source)
	}

	destFile := filepath.Join(globalDir, "Test Skill", "SKILL.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("installed skill file not found")
	}
}

func TestManager_SaveSkill(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	mgr := NewManager(globalDir)

	skill, err := mgr.SaveSkill("new-skill", "# New Skill\n\nNew instructions.")
	if err != nil {
		t.Fatalf("SaveSkill: %v", err)
	}

	if skill.Name != "new-skill" {
		t.Errorf("expected name 'new-skill', got '%s'", skill.Name)
	}
	if skill.Source != SourceGlobal {
		t.Errorf("expected source 'global', got '%s'", skill.Source)
	}

	destFile := filepath.Join(globalDir, "new-skill", "SKILL.md")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("saved skill file not found")
	}

	data, _ := os.ReadFile(destFile)
	content := string(data)
	if !strings.Contains(content, "name: new-skill") {
		t.Errorf("expected frontmatter 'name: new-skill' in file, got: %s", content)
	}
	if !strings.Contains(content, "---") {
		t.Errorf("expected frontmatter in file, got: %s", content)
	}
}

func TestManager_SaveSkill_WithFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	mgr := NewManager(globalDir)

	fullContent := "---\nname: cool-skill\ndescription: A very cool skill\n---\n\n# Cool Skill\n\nInstructions here."
	skill, err := mgr.SaveSkill("cool-skill", fullContent)
	if err != nil {
		t.Fatalf("SaveSkill: %v", err)
	}

	if skill.Description != "A very cool skill" {
		t.Errorf("expected description 'A very cool skill', got '%s'", skill.Description)
	}

	destFile := filepath.Join(globalDir, "cool-skill", "SKILL.md")
	data, _ := os.ReadFile(destFile)
	if string(data) != fullContent {
		t.Errorf("content should be preserved, got: %s", string(data))
	}
}

func TestManager_DeleteSkill(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	createTestSkill(t, globalDir, "to-delete", "To Delete", "Temporary skill", "# To Delete\n\nTemporary skill.")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	if err := mgr.DeleteSkill("To Delete"); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}

	_, err := mgr.GetSkill("To Delete")
	if err != ErrSkillNotFound {
		t.Errorf("expected ErrSkillNotFound, got %v", err)
	}

	destDir := filepath.Join(globalDir, "To Delete")
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		t.Error("skill directory should be deleted")
	}
}

func TestManager_Rescan(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	skills := mgr.GetAllSkills()
	if len(skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(skills))
	}

	createTestSkill(t, globalDir, "new-skill", "New Skill", "New skill", "# New Skill\n\nContent.")

	if err := mgr.Rescan(); err != nil {
		t.Fatalf("Rescan: %v", err)
	}

	skills = mgr.GetAllSkills()
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill after rescan, got %d", len(skills))
	}
}

func TestManager_PriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")
	projectDir := filepath.Join(tmpDir, ".devo", "skills")

	createTestSkill(t, globalDir, "skill-a", "Skill A", "Skill A", "Global.")
	createTestSkill(t, globalDir, "skill-b", "Skill B", "Skill B", "Global.")
	createTestSkill(t, projectDir, "skill-c", "Skill C", "Skill C", "Project.")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	skills := mgr.GetAllSkills()
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(skills))
	}

	if skills[0].Priority < skills[1].Priority {
		t.Error("skills should be sorted by priority descending")
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		content  string
		wantName string
		wantDesc string
	}{
		{
			"---\nname: test-skill\ndescription: A test skill\n---\n\n# Body",
			"test-skill",
			"A test skill",
		},
		{
			"---\nname: multi-word skill\ndescription: Does something cool\n---\n\nContent",
			"multi-word skill",
			"Does something cool",
		},
		{
			"# No frontmatter\n\nJust content",
			"",
			"",
		},
		{
			"---\nname:only-name\n---\n\nBody",
			"only-name",
			"",
		},
		{
			"",
			"",
			"",
		},
	}

	for _, tt := range tests {
		fm := parseFrontmatter(tt.content)
		if fm["name"] != tt.wantName {
			t.Errorf("parseFrontmatter(%q) name = %q, want %q", tt.content, fm["name"], tt.wantName)
		}
		if fm["description"] != tt.wantDesc {
			t.Errorf("parseFrontmatter(%q) description = %q, want %q", tt.content, fm["description"], tt.wantDesc)
		}
	}
}

func TestExtractSkillName(t *testing.T) {
	tests := []struct {
		content  string
		fallback string
		expected string
	}{
		{"# My Skill\n\nContent", "fallback", "My Skill"},
		{"No heading here", "fallback", "fallback"},
		{"#  First\n# Second", "fallback", "First"},
		{"", "fallback", "fallback"},
	}

	for _, tt := range tests {
		result := extractSkillName(tt.content, tt.fallback)
		if result != tt.expected {
			t.Errorf("extractSkillName(%q, %q) = %q, want %q", tt.content, tt.fallback, result, tt.expected)
		}
	}
}

func TestManager_ListSkillResources(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")

	scriptsDir := filepath.Join(skillDir, "scripts")
	refsDir := filepath.Join(skillDir, "references")
	assetsDir := filepath.Join(skillDir, "assets")

	os.MkdirAll(scriptsDir, 0755)
	os.MkdirAll(refsDir, 0755)
	os.MkdirAll(assetsDir, 0755)

	os.WriteFile(filepath.Join(scriptsDir, "helper.py"), []byte("print('hello')"), 0644)
	os.WriteFile(filepath.Join(refsDir, "guide.md"), []byte("# Guide"), 0644)
	os.WriteFile(filepath.Join(assetsDir, "template.py"), []byte("def main(): pass"), 0644)

	mgr := NewManager(tmpDir)
	scripts, refs, assets := mgr.ListSkillResources(skillDir)

	if len(scripts) != 1 || scripts[0] != "scripts/helper.py" {
		t.Errorf("expected scripts to contain 'scripts/helper.py', got %v", scripts)
	}
	if len(refs) != 1 || refs[0] != "references/guide.md" {
		t.Errorf("expected references to contain 'references/guide.md', got %v", refs)
	}
	if len(assets) != 1 || assets[0] != "assets/template.py" {
		t.Errorf("expected assets to contain 'assets/template.py', got %v", assets)
	}
}

func TestManager_ListSkillResources_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "empty-skill")
	os.MkdirAll(skillDir, 0755)

	mgr := NewManager(tmpDir)
	scripts, refs, assets := mgr.ListSkillResources(skillDir)

	if len(scripts) != 0 || len(refs) != 0 || len(assets) != 0 {
		t.Error("expected empty resources for skill with no subdirectories")
	}
}

func TestManager_ListSkillResources_IgnoresHidden(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "my-skill")

	hiddenDir := filepath.Join(skillDir, ".git")
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "config"), []byte("ignored"), 0644)

	dotDir := filepath.Join(skillDir, ".hidden")
	os.MkdirAll(dotDir, 0755)
	os.WriteFile(filepath.Join(dotDir, "secret.txt"), []byte("ignored"), 0644)

	mgr := NewManager(tmpDir)
	scripts, refs, assets := mgr.ListSkillResources(skillDir)

	if len(scripts) != 0 || len(refs) != 0 || len(assets) != 0 {
		t.Error("expected hidden/dot directories to be ignored")
	}
}

func TestManager_MultipleSkillsWithActiveFilter(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	createTestSkill(t, globalDir, "python-expert", "Python Expert", "Python coding", "# Python\n\nUse type hints.")
	createTestSkill(t, globalDir, "go-expert", "Go Expert", "Go coding", "# Go\n\nUse tabs.")
	createTestSkill(t, globalDir, "code-review", "Code Review", "Security review", "# Review\n\nCheck for bugs.")

	mgr := NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	skills := mgr.GetAllSkills()
	if len(skills) != 3 {
		t.Fatalf("expected 3 skills, got %d", len(skills))
	}

	prompt := mgr.GetActiveSkillsPrompt()
	if !strings.Contains(prompt, "Python coding") {
		t.Error("expected 'Python Expert' description in catalog")
	}
	if !strings.Contains(prompt, "Security review") {
		t.Error("expected 'Code Review' description in catalog")
	}
	if !strings.Contains(prompt, "Go coding") {
		t.Error("expected 'Go Expert' description in catalog")
	}
	if strings.Contains(prompt, "Use type hints") {
		t.Error("should not contain full instructions in catalog")
	}
}
