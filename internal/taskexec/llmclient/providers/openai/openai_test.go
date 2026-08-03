package openai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"devo/internal/core/session"
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
func (m *mockTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w tools.StreamWriter) error {
	return nil
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

func TestConvertMessages_AssistantWithToolCallsHasContent(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "hi"},
		{
			ID:      "m2",
			Role:    session.RoleAssistant,
			Content: "",
			ToolCalls: []session.ToolCall{
				{ID: "call_1", ToolName: "read_file", Params: map[string]interface{}{"path": "a.go"}},
			},
		},
		{ID: "m3", Role: session.RoleTool, Content: "file contents", ToolCallID: "call_1"},
	}

	result := convertMessages(msgs, "system")

	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	assistantMsg := result[2]
	if assistantMsg.Role != "assistant" {
		t.Fatalf("expected assistant role, got %s", assistantMsg.Role)
	}
	if assistantMsg.Content != nil {
		t.Errorf("expected nil content for assistant with tool_calls, got %v", assistantMsg.Content)
	}

	body, err := json.Marshal(assistantMsg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `"content":null`) {
		t.Errorf("expected content:null in JSON, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"tool_calls"`) {
		t.Errorf("expected tool_calls field in JSON, got: %s", bodyStr)
	}
}

func TestConvertMessages_EmptyToolResultHasContent(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "hi"},
		{
			ID:      "m2",
			Role:    session.RoleAssistant,
			Content: "",
			ToolCalls: []session.ToolCall{
				{ID: "call_1", ToolName: "list_files", Params: map[string]interface{}{"path": "."}},
			},
		},
		{ID: "m3", Role: session.RoleTool, Content: "", ToolCallID: "call_1"},
	}

	result := convertMessages(msgs, "system")

	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	toolMsg := result[3]
	if toolMsg.Role != "tool" {
		t.Fatalf("expected tool role, got %s", toolMsg.Role)
	}
	if toolMsg.Content != "(无输出)" {
		t.Errorf("expected placeholder content, got %v", toolMsg.Content)
	}

	body, err := json.Marshal(toolMsg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `"content":"`) {
		t.Errorf("expected non-empty content field in JSON, got: %s", bodyStr)
	}
}
