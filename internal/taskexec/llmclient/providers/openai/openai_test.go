package openai

import (
	"testing"

	"devo/internal/taskexec/tools"
)

type mockTool struct {
	name        string
	description string
	params      map[string]interface{}
}

func (m *mockTool) Name() string                         { return m.name }
func (m *mockTool) Description() string                  { return m.description }
func (m *mockTool) RiskLevel() tools.RiskLevel           { return tools.RiskLevelNone }
func (m *mockTool) ParamsSchema() map[string]interface{} { return m.params }
func (m *mockTool) Execute(workingDir string, params map[string]interface{}) (string, error) {
	return "ok", nil
}

func TestBuildToolDefs_Empty(t *testing.T) {
	result := buildToolDefs(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result))
	}

	result = buildToolDefs([]tools.Tool{})
	if len(result) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result))
	}
}

func TestBuildToolDefs_Single(t *testing.T) {
	tl := &mockTool{
		name:        "use_skill",
		description: "Load a skill",
		params: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"skill_name": map[string]interface{}{
					"type":        "string",
					"description": "The skill name",
				},
			},
			"required": []string{"skill_name"},
		},
	}

	result := buildToolDefs([]tools.Tool{tl})
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	if result[0].Type != "function" {
		t.Errorf("expected type 'function', got %s", result[0].Type)
	}
	if result[0].Function.Name != "use_skill" {
		t.Errorf("expected name 'use_skill', got %s", result[0].Function.Name)
	}
	if result[0].Function.Description != "Load a skill" {
		t.Errorf("expected description 'Load a skill', got %s", result[0].Function.Description)
	}
	if result[0].Function.Parameters == nil {
		t.Error("expected non-nil parameters")
	}
}

func TestBuildToolDefs_Multiple(t *testing.T) {
	tools := []tools.Tool{
		&mockTool{name: "read_file", description: "Read a file"},
		&mockTool{name: "write_file", description: "Write a file"},
		&mockTool{name: "use_skill", description: "Load a skill"},
	}

	result := buildToolDefs(tools)
	if len(result) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(result))
	}

	names := make([]string, len(result))
	for i, r := range result {
		names[i] = r.Function.Name
		if r.Type != "function" {
			t.Errorf("tool %s: expected type 'function', got %s", r.Function.Name, r.Type)
		}
	}

	expected := map[string]bool{"read_file": true, "write_file": true, "use_skill": true}
	for _, name := range names {
		if !expected[name] {
			t.Errorf("unexpected tool name: %s", name)
		}
	}
}

func TestBuildToolDefs_RegistryIntegration(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(&mockTool{
		name:        "use_skill",
		description: "Load a skill with instructions and resources",
		params: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	})

	toolList := registry.ListTools()
	result := buildToolDefs(toolList)

	if len(result) != 1 {
		t.Fatalf("expected 1 tool from registry, got %d", len(result))
	}
	if result[0].Function.Name != "use_skill" {
		t.Errorf("expected 'use_skill', got %s", result[0].Function.Name)
	}
	if result[0].Function.Description != "Load a skill with instructions and resources" {
		t.Errorf("unexpected description: %s", result[0].Function.Description)
	}
}

func TestBuildToolDefs_RegistryDynamicUpdate(t *testing.T) {
	registry := tools.NewRegistry()

	registry.Register(&mockTool{name: "read_file", description: "Read a file"})
	result1 := buildToolDefs(registry.ListTools())
	if len(result1) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result1))
	}

	registry.Register(&mockTool{name: "use_skill", description: "Load a skill"})
	result2 := buildToolDefs(registry.ListTools())
	if len(result2) != 2 {
		t.Fatalf("expected 2 tools after registration, got %d", len(result2))
	}

	registry.Unregister("read_file")
	result3 := buildToolDefs(registry.ListTools())
	if len(result3) != 1 {
		t.Fatalf("expected 1 tool after unregister, got %d", len(result3))
	}
	if result3[0].Function.Name != "use_skill" {
		t.Errorf("expected 'use_skill', got %s", result3[0].Function.Name)
	}
}
