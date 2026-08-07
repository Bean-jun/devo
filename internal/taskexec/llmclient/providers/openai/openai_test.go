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

func TestConvertMessages_ImageWithText(t *testing.T) {
	msgs := []session.Message{
		{
			ID:      "m1",
			Role:    session.RoleUser,
			Content: "这是什么东西",
			ContentParts: []session.ContentPart{
				{Type: "image_url", URL: "data:image/jpeg;base64,/9j/test"},
				{Type: "text", Text: "这是什么东西"},
			},
		},
	}

	result := convertMessages(msgs, "")

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	userMsg := result[0]
	if userMsg.Role != "user" {
		t.Fatalf("expected user role, got %s", userMsg.Role)
	}

	parts, ok := userMsg.Content.([]openaiContentPart)
	if !ok {
		t.Fatalf("expected Content to be []openaiContentPart, got %T", userMsg.Content)
	}
	if len(parts) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(parts))
	}

	if parts[0].Type != "image_url" {
		t.Errorf("expected first part type 'image_url', got %s", parts[0].Type)
	}
	if parts[0].ImageURL == nil {
		t.Error("expected non-nil ImageURL")
	} else if parts[0].ImageURL.URL != "data:image/jpeg;base64,/9j/test" {
		t.Errorf("expected image URL, got %s", parts[0].ImageURL.URL)
	}

	if parts[1].Type != "text" {
		t.Errorf("expected second part type 'text', got %s", parts[1].Type)
	}
	if parts[1].Text != "这是什么东西" {
		t.Errorf("expected text '这是什么东西', got %s", parts[1].Text)
	}

	body, err := json.Marshal(userMsg)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `"image_url"`) {
		t.Errorf("expected image_url in JSON, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"text"`) {
		t.Errorf("expected text in JSON, got: %s", bodyStr)
	}
}

func TestConvertMessages_ImageOnly(t *testing.T) {
	msgs := []session.Message{
		{
			ID:      "m1",
			Role:    session.RoleUser,
			Content: "",
			ContentParts: []session.ContentPart{
				{Type: "image_url", URL: "data:image/png;base64,iVBORw0KGgo="},
			},
		},
	}

	result := convertMessages(msgs, "")

	if len(result) != 1 {
		t.Fatalf("expected 1 message, got %d", len(result))
	}

	parts, ok := result[0].Content.([]openaiContentPart)
	if !ok {
		t.Fatalf("expected Content to be []openaiContentPart, got %T", result[0].Content)
	}
	if len(parts) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(parts))
	}
	if parts[0].Type != "image_url" {
		t.Errorf("expected type 'image_url', got %s", parts[0].Type)
	}

	body, err := json.Marshal(result[0])
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, `"image_url"`) {
		t.Errorf("expected image_url in JSON, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, `"url":"data:image/png;base64,iVBORw0KGgo="`) {
		t.Errorf("expected image URL in JSON, got: %s", bodyStr)
	}
}

func TestConvertMessages_MultipleImages(t *testing.T) {
	msgs := []session.Message{
		{
			ID:      "m1",
			Role:    session.RoleUser,
			Content: "compare these two images",
			ContentParts: []session.ContentPart{
				{Type: "image_url", URL: "data:image/jpeg;base64,img1"},
				{Type: "image_url", URL: "data:image/jpeg;base64,img2"},
				{Type: "text", Text: "compare these two images"},
			},
		},
	}

	result := convertMessages(msgs, "")

	parts, ok := result[0].Content.([]openaiContentPart)
	if !ok {
		t.Fatalf("expected Content to be []openaiContentPart, got %T", result[0].Content)
	}
	if len(parts) != 3 {
		t.Fatalf("expected 3 content parts, got %d", len(parts))
	}

	imageCount := 0
	textCount := 0
	for _, p := range parts {
		switch p.Type {
		case "image_url":
			imageCount++
			if p.ImageURL == nil {
				t.Error("expected non-nil ImageURL for image part")
			}
		case "text":
			textCount++
		}
	}
	if imageCount != 2 {
		t.Errorf("expected 2 image parts, got %d", imageCount)
	}
	if textCount != 1 {
		t.Errorf("expected 1 text part, got %d", textCount)
	}
}

func TestConvertMessages_ContentPartsBackwardCompatible(t *testing.T) {
	msgs := []session.Message{
		{ID: "m1", Role: session.RoleUser, Content: "plain text message"},
		{
			ID:      "m2",
			Role:    session.RoleAssistant,
			Content: "",
			ToolCalls: []session.ToolCall{
				{ID: "call_1", ToolName: "read_file", Params: map[string]interface{}{"path": "a.go"}},
			},
		},
		{ID: "m3", Role: session.RoleTool, Content: "", ToolCallID: "call_1"},
	}

	result := convertMessages(msgs, "system")

	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	if result[1].Content != "plain text message" {
		t.Errorf("expected plain text, got %v", result[1].Content)
	}

	assistantMsg := result[2]
	if assistantMsg.Content != nil {
		t.Errorf("expected nil content for assistant with tool_calls, got %v", assistantMsg.Content)
	}

	toolMsg := result[3]
	if toolMsg.Content != "(无输出)" {
		t.Errorf("expected placeholder, got %v", toolMsg.Content)
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
