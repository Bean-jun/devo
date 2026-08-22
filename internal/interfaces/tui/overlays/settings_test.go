package overlays

import (
	"strings"
	"testing"

	"devo/internal/interfaces/tui/api"
)

func TestSettingsPanel_BuildFields(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{
		ToolCallLimit:    intPtr(20),
		MaxContextTokens: intPtr(100000),
		KeepRecent:       intPtr(50),
		ApprovalPolicy: map[string]string{
			"file_write_new": "always_ask",
		},
	}
	sp.GlobalConfig = &api.GlobalConfigInfo{
		ToolCallLimit: intPtr(100),
		ApprovalPolicy: map[string]string{
			"exec_python": "session_trust",
		},
	}

	sp.BuildFields()

	if len(sp.Fields) == 0 {
		t.Error("BuildFields() 应创建字段列表")
	}
	if sp.Selected != 0 {
		t.Error("初始 Selected 应为 0")
	}

	hasProject := false
	hasGlobal := false
	for _, f := range sp.Fields {
		if f.Group == "project" {
			hasProject = true
		}
		if f.Group == "global" {
			hasGlobal = true
		}
	}
	if !hasProject {
		t.Error("应包含项目设置字段")
	}
	if !hasGlobal {
		t.Error("应包含全局设置字段")
	}
}

func TestSettingsPanel_CursorNavigation(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{
		ToolCallLimit: intPtr(20),
		ApprovalPolicy: map[string]string{
			"file_write_new": "always_ask",
		},
	}
	sp.GlobalConfig = &api.GlobalConfigInfo{
		ToolCallLimit: intPtr(100),
	}
	sp.BuildFields()

	if len(sp.Fields) < 2 {
		t.Skip("需要至少 2 个字段")
	}

	sp.CursorDown()
	if sp.Selected != 1 {
		t.Errorf("CursorDown() 后 Selected 应为 1, got %d", sp.Selected)
	}
	sp.CursorUp()
	if sp.Selected != 0 {
		t.Errorf("CursorUp() 后 Selected 应为 0, got %d", sp.Selected)
	}
}

func TestSettingsPanel_CursorUpBoundary(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()

	sp.CursorUp()
	if sp.Selected != 0 {
		t.Error("在顶部 CursorUp() 不应超出边界")
	}
}

func TestSettingsPanel_CursorDownBoundary(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()

	for i := 0; i < len(sp.Fields)+10; i++ {
		sp.CursorDown()
	}
	if sp.Selected >= len(sp.Fields) {
		t.Error("CursorDown() 不应超出列表范围")
	}
}

func TestSettingsPanel_EditIntField(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()
	sp.Selected = 0

	sp.StartEditing()
	if !sp.Editing {
		t.Error("StartEditing() 后 Editing 应为 true")
	}
	if sp.EditBuffer != "" {
		t.Error("StartEditing() 后 EditBuffer 应为空")
	}

	sp.EditBuffer = "50"
	f, val, ok := sp.ConfirmEditing()
	if !ok {
		t.Error("ConfirmEditing() 应返回 ok=true")
	}
	if val != 50 {
		t.Errorf("ConfirmEditing() 值应为 50, got %d", val)
	}
	if f == nil || f.IntValue == nil || *f.IntValue != 50 {
		t.Error("字段值应更新为 50")
	}
	if sp.Editing {
		t.Error("ConfirmEditing() 后 Editing 应为 false")
	}
}

func TestSettingsPanel_EditInvalidInput(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()
	sp.Selected = 0

	sp.StartEditing()
	sp.EditBuffer = "abc"
	_, _, ok := sp.ConfirmEditing()
	if ok {
		t.Error("无效输入 abc 应返回 ok=false")
	}
}

func TestSettingsPanel_EditNegativeInput(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()
	sp.Selected = 0

	sp.StartEditing()
	sp.EditBuffer = "-1"
	_, _, ok := sp.ConfirmEditing()
	if ok {
		t.Error("负数输入应返回 ok=false")
	}
}

func TestSettingsPanel_CancelEditing(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()

	sp.StartEditing()
	sp.EditBuffer = "999"
	sp.CancelEditing()
	if sp.Editing {
		t.Error("CancelEditing() 后 Editing 应为 false")
	}
	if sp.EditBuffer != "" {
		t.Error("CancelEditing() 后 EditBuffer 应为空")
	}
}

func TestSettingsPanel_CycleEnum(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{
		ApprovalPolicy: map[string]string{
			"file_write_new": "always_ask",
		},
	}
	sp.BuildFields()

	sp.Selected = 3
	f := sp.CycleEnum()
	if f == nil {
		t.Fatal("CycleEnum() 应返回字段")
	}
	if f.EnumValue != ApprovalLevelSessionTrust {
		t.Errorf("枚举值应从 always_ask 切换到 session_trust, got %s", f.EnumValue)
	}

	f = sp.CycleEnum()
	if f.EnumValue != ApprovalLevelAutoApprove {
		t.Errorf("枚举值应切换到 auto_approve, got %s", f.EnumValue)
	}

	f = sp.CycleEnum()
	if f.EnumValue != ApprovalLevelAlwaysAsk {
		t.Errorf("枚举值应循环回到 always_ask, got %s", f.EnumValue)
	}
}

func TestSettingsPanel_CycleEnumOnIntField(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()

	sp.Selected = 0
	f := sp.CycleEnum()
	if f != nil {
		t.Error("在 int 字段上调用 CycleEnum() 应返回 nil")
	}
}

func TestSettingsPanel_BuildProjectSaveBody(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{
		ToolCallLimit:    intPtr(20),
		MaxContextTokens: intPtr(100000),
		KeepRecent:       intPtr(50),
		ApprovalPolicy: map[string]string{
			"file_write_new": "always_ask",
		},
	}
	sp.BuildFields()

	body := sp.BuildProjectSaveBody()
	if body["tool_call_limit"] != 20 {
		t.Errorf("tool_call_limit 应为 20, got %v", body["tool_call_limit"])
	}
	if body["max_context_tokens"] != 100000 {
		t.Errorf("max_context_tokens 应为 100000")
	}
	if body["keep_recent"] != 50 {
		t.Errorf("keep_recent 应为 50")
	}
	ap, ok := body["approval_policy"].(map[string]string)
	if !ok {
		t.Fatal("approval_policy 应为 map[string]string")
	}
	if ap["file_write_new"] != "always_ask" {
		t.Errorf("approval_policy[file_write_new] 应为 always_ask, got %s", ap["file_write_new"])
	}
}

func TestSettingsPanel_BuildGlobalSaveBody(t *testing.T) {
	sp := NewSettingsPanel()
	sp.GlobalConfig = &api.GlobalConfigInfo{
		ToolCallLimit: intPtr(100),
		ApprovalPolicy: map[string]string{
			"exec_python": "session_trust",
		},
	}
	sp.BuildFields()

	body := sp.BuildGlobalSaveBody()
	if body["tool_call_limit"] != 100 {
		t.Errorf("tool_call_limit 应为 100, got %v", body["tool_call_limit"])
	}
	ap, ok := body["approval_policy"].(map[string]string)
	if !ok {
		t.Fatal("approval_policy 应为 map[string]string")
	}
	if ap["exec_python"] != "session_trust" {
		t.Errorf("approval_policy[exec_python] 应为 session_trust, got %s", ap["exec_python"])
	}
}

func TestSettingsPanel_BuildGlobalSaveBodyWithLLM(t *testing.T) {
	sp := NewSettingsPanel()
	sp.GlobalConfig = &api.GlobalConfigInfo{
		ToolCallLimit: intPtr(100),
		LLM: &api.LLMConfigInfo{
			MaxTokens: intPtr(4096),
		},
	}
	sp.BuildFields()

	body := sp.BuildGlobalSaveBody()
	llm, ok := body["llm"].(map[string]interface{})
	if !ok {
		t.Fatal("llm 应为 map")
	}
	if llm["max_tokens"] != 4096 {
		t.Errorf("llm.max_tokens 应为 4096, got %v", llm["max_tokens"])
	}
}

func TestSettingsPanel_Render(t *testing.T) {
	sp := NewSettingsPanel()
	sp.Width = 80
	sp.ProjectConfig = &api.ProjectConfigInfo{
		ToolCallLimit: intPtr(20),
		ApprovalPolicy: map[string]string{
			"file_write_new": "always_ask",
		},
	}
	sp.GlobalConfig = &api.GlobalConfigInfo{
		ToolCallLimit: intPtr(100),
	}
	sp.BuildFields()

	result := sp.Render()
	if !strings.Contains(result, "Settings") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "项目设置") {
		t.Error("渲染结果应包含 '项目设置'")
	}
	if !strings.Contains(result, "全局设置") {
		t.Error("渲染结果应包含 '全局设置'")
	}
}

func TestSettingsPanel_RenderEditing(t *testing.T) {
	sp := NewSettingsPanel()
	sp.Width = 80
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()
	sp.StartEditing()
	sp.EditBuffer = "123"

	result := sp.Render()
	if !strings.Contains(result, "123") {
		t.Error("渲染结果应包含编辑缓冲区内容")
	}
	if !strings.Contains(result, "确认") {
		t.Error("渲染结果应包含确认提示")
	}
}

func TestSettingsPanel_SelectedField(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{ToolCallLimit: intPtr(20)}
	sp.BuildFields()
	sp.Selected = 0

	f := sp.SelectedField()
	if f == nil {
		t.Fatal("SelectedField() 应返回字段")
	}
	if f.Key != "tool_call_limit" {
		t.Errorf("第一个项目字段应为 tool_call_limit, got %s", f.Key)
	}
}

func TestSettingsPanel_SelectedFieldNil(t *testing.T) {
	sp := NewSettingsPanel()
	sp.Selected = -1
	f := sp.SelectedField()
	if f != nil {
		t.Error("Selected 为 -1 时应返回 nil")
	}
}

func TestSettingsPanel_CycleEnumUnknownValue(t *testing.T) {
	sp := NewSettingsPanel()
	sp.ProjectConfig = &api.ProjectConfigInfo{
		ApprovalPolicy: map[string]string{
			"file_write_new": "unknown_level",
		},
	}
	sp.BuildFields()

	sp.Selected = 3
	f := sp.CycleEnum()
	if f == nil {
		t.Fatal("CycleEnum() 应返回字段")
	}
	if f.EnumValue != ApprovalLevels[0] {
		t.Errorf("未知值应重置为第一个选项, got %s", f.EnumValue)
	}
}

func intPtr(i int) *int {
	return &i
}
