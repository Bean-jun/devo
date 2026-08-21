package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"devo/internal/core/skills"
)

func setupTestSkillsManager(t *testing.T) (*skills.Manager, string) {
	t.Helper()

	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	pyDir := filepath.Join(globalDir, "python-expert")
	os.MkdirAll(pyDir, 0755)
	os.WriteFile(filepath.Join(pyDir, "SKILL.md"), []byte("---\nname: Python Expert\ndescription: Python coding best practices\n---\n\n# Python Expert\n\nAlways use type hints.\nUse pathlib for paths."), 0644)

	scriptsDir := filepath.Join(pyDir, "scripts")
	os.MkdirAll(scriptsDir, 0755)
	os.WriteFile(filepath.Join(scriptsDir, "helper.py"), []byte("print('hello')"), 0644)

	refsDir := filepath.Join(pyDir, "references")
	os.MkdirAll(refsDir, 0755)
	os.WriteFile(filepath.Join(refsDir, "style-guide.md"), []byte("# Style Guide"), 0644)

	assetsDir := filepath.Join(pyDir, "assets")
	os.MkdirAll(assetsDir, 0755)
	os.WriteFile(filepath.Join(assetsDir, "template.py"), []byte("def main(): pass"), 0644)

	mgr := skills.NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	return mgr, tmpDir
}

func TestUseSkillTool_Name(t *testing.T) {
	tool := &UseSkillTool{}
	if tool.Name() != "use_skill" {
		t.Errorf("expected name 'use_skill', got '%s'", tool.Name())
	}
}

func TestUseSkillTool_RiskLevel(t *testing.T) {
	tool := &UseSkillTool{}
	if tool.RiskLevel() != RiskLevelNone {
		t.Errorf("expected RiskLevelNone, got '%s'", tool.RiskLevel())
	}
}

func TestUseSkillTool_ParamsSchema(t *testing.T) {
	tool := &UseSkillTool{}
	schema := tool.ParamsSchema()

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("expected 'required' field in schema")
	}
	if len(required) != 1 || required[0] != "skill_name" {
		t.Errorf("expected 'skill_name' in required params, got %v", required)
	}
}

func TestUseSkillTool_Execute_Success(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)
	tool := NewUseSkillTool(mgr)

	result, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "Python Expert",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Content, "[Skill Loaded]") {
		t.Error("expected '[Skill Loaded]' in result")
	}
	if !strings.Contains(result.Content, "Python Expert") {
		t.Error("expected 'Python Expert' in result")
	}
	if !strings.Contains(result.Content, "[Instructions]") {
		t.Error("expected '[Instructions]' section in result")
	}
	if !strings.Contains(result.Content, "Always use type hints") {
		t.Error("expected skill instructions in result")
	}
	if !strings.Contains(result.Content, "[Available Resources]") {
		t.Error("expected '[Available Resources]' section in result")
	}
	if !strings.Contains(result.Content, "scripts/helper.py") {
		t.Error("expected 'scripts/helper.py' in resources")
	}
	if !strings.Contains(result.Content, "references/style-guide.md") {
		t.Error("expected 'references/style-guide.md' in resources")
	}
	if !strings.Contains(result.Content, "assets/template.py") {
		t.Error("expected 'assets/template.py' in resources")
	}
}

func TestUseSkillTool_Execute_NotFound(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)
	tool := NewUseSkillTool(mgr)

	result, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "NonExistentSkill",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for non-existent skill")
	}
	if !strings.Contains(result.Error, "skill not found") {
		t.Errorf("expected 'skill not found' error, got: %v", result.Error)
	}
}

func TestUseSkillTool_Execute_MissingParam(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)
	tool := NewUseSkillTool(mgr)

	result, err := executeTool(t, tool, "/tmp", map[string]interface{}{})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for missing skill_name parameter")
	}
}

func TestUseSkillTool_Execute_EmptySkillName(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)
	tool := NewUseSkillTool(mgr)

	result, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for empty skill_name")
	}
}

func TestUseSkillTool_Execute_Deduplication(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)
	tool := NewUseSkillTool(mgr)

	result1, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "Python Expert",
	})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if !result1.Success {
		t.Fatalf("expected success, got error: %s", result1.Error)
	}
	if !strings.Contains(result1.Content, "[Instructions]") {
		t.Error("first call should include instructions")
	}

	result2, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "Python Expert",
	})
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !result2.Success {
		t.Fatalf("expected success, got error: %s", result2.Error)
	}
	if strings.Contains(result2.Content, "[Instructions]") {
		t.Error("second call should not include instructions (already loaded)")
	}
	if !strings.Contains(result2.Content, "already loaded") {
		t.Error("second call should indicate skill is already loaded")
	}
}

func TestUseSkillTool_Execute_Disabled(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)
	tool := NewUseSkillTool(mgr)

	if err := mgr.DisableSkill("Python Expert"); err != nil {
		t.Fatalf("DisableSkill: %v", err)
	}

	result, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "Python Expert",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.Success {
		t.Fatal("expected error for disabled skill")
	}
	if !strings.Contains(result.Error, "disabled") {
		t.Errorf("expected 'disabled' error, got: %v", result.Error)
	}
}

func TestUseSkillTool_Reset(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)
	tool := NewUseSkillTool(mgr)

	executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "Python Expert",
	})
	tool.Reset()

	result, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "Python Expert",
	})
	if err != nil {
		t.Fatalf("after reset: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if !strings.Contains(result.Content, "[Instructions]") {
		t.Error("after reset, instructions should be included again")
	}
}

func TestUseSkillTool_Execute_NoResources(t *testing.T) {
	tmpDir := t.TempDir()
	globalDir := filepath.Join(tmpDir, "global-skills")

	skillDir := filepath.Join(globalDir, "simple-skill")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: Simple Skill\ndescription: Simple\n---\n\n# Simple\n\nJust instructions."), 0644)

	mgr := skills.NewManager(globalDir)
	if err := mgr.SetProjectDir(tmpDir); err != nil {
		t.Fatalf("SetProjectDir: %v", err)
	}

	tool := NewUseSkillTool(mgr)
	result, err := executeTool(t, tool, tmpDir, map[string]interface{}{
		"skill_name": "Simple Skill",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if strings.Contains(result.Content, "[Available Resources]") {
		t.Error("should not contain '[Available Resources]' when no resources present")
	}
	if !strings.Contains(result.Content, "Just instructions") {
		t.Error("expected instructions in result")
	}
}

func TestUseSkillTool_SkillNotAllowed(t *testing.T) {
	mgr, _ := setupTestSkillsManager(t)

	deniedProvider := &deniedSkillsProvider{mgr}
	tool := NewUseSkillTool(deniedProvider)

	result, err := executeTool(t, tool, "/tmp", map[string]interface{}{
		"skill_name": "Python Expert",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Error == "" {
		t.Fatal("expected error for denied skill")
	}
	if !strings.Contains(result.Error, "not available") {
		t.Errorf("expected 'not available' error, got: %q", result.Error)
	}
}

type deniedSkillsProvider struct {
	*skills.Manager
}

func (d *deniedSkillsProvider) IsSkillAllowed(name string) bool {
	return false
}
