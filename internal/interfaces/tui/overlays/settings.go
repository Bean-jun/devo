package overlays

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"devo/internal/interfaces/tui/api"
	"devo/internal/interfaces/tui/components"
)

const (
	ApprovalLevelAlwaysAsk    = "always_ask"
	ApprovalLevelSessionTrust = "session_trust"
	ApprovalLevelFullTrust    = "full_trust"
	ApprovalLevelAutoApprove  = "auto_approve"
)

var ApprovalLevels = []string{
	ApprovalLevelAlwaysAsk,
	ApprovalLevelSessionTrust,
	ApprovalLevelFullTrust,
	ApprovalLevelAutoApprove,
}

var ApprovalLevelLabels = map[string]string{
	ApprovalLevelAlwaysAsk:    "始终询问",
	ApprovalLevelSessionTrust: "会话信任",
	ApprovalLevelFullTrust:    "永久信任",
	ApprovalLevelAutoApprove:  "自动批准",
}

type SettingsField struct {
	Label       string
	Group       string
	Key         string
	FieldType   string
	IntValue    *int
	EnumValue   string
	EnumOptions []string
	EnumKey     string
}

type SettingsPanel struct {
	Width         int
	Height        int
	Fields        []SettingsField
	Selected      int
	Editing       bool
	EditBuffer    string
	ProjectConfig *api.ProjectConfigInfo
	GlobalConfig  *api.GlobalConfigInfo
}

func NewSettingsPanel() SettingsPanel {
	return SettingsPanel{}
}

func (sp *SettingsPanel) BuildFields() {
	sp.Fields = nil
	sp.Selected = 0

	if sp.ProjectConfig != nil {
		pc := sp.ProjectConfig
		sp.Fields = append(sp.Fields, SettingsField{
			Label: "项目 · 工具调用上限", Group: "project", Key: "tool_call_limit",
			FieldType: "int", IntValue: pc.ToolCallLimit,
		})
		sp.Fields = append(sp.Fields, SettingsField{
			Label: "项目 · 最大上下文 tokens", Group: "project", Key: "max_context_tokens",
			FieldType: "int", IntValue: pc.MaxContextTokens,
		})
		sp.Fields = append(sp.Fields, SettingsField{
			Label: "项目 · 保留最近消息数", Group: "project", Key: "keep_recent",
			FieldType: "int", IntValue: pc.KeepRecent,
		})
		for k, v := range pc.ApprovalPolicy {
			sp.Fields = append(sp.Fields, SettingsField{
				Label: "项目 · 审批: " + k, Group: "project", Key: "approval_policy",
				FieldType: "enum", EnumValue: v, EnumOptions: ApprovalLevels, EnumKey: k,
			})
		}
	}

	if sp.GlobalConfig != nil {
		gc := sp.GlobalConfig
		sp.Fields = append(sp.Fields, SettingsField{
			Label: "全局 · 工具调用上限", Group: "global", Key: "tool_call_limit",
			FieldType: "int", IntValue: gc.ToolCallLimit,
		})
		sp.Fields = append(sp.Fields, SettingsField{
			Label: "全局 · 最大上下文 tokens", Group: "global", Key: "max_context_tokens",
			FieldType: "int", IntValue: gc.MaxContextTokens,
		})
		sp.Fields = append(sp.Fields, SettingsField{
			Label: "全局 · 保留最近消息数", Group: "global", Key: "keep_recent",
			FieldType: "int", IntValue: gc.KeepRecent,
		})
		if gc.LLM != nil {
			sp.Fields = append(sp.Fields, SettingsField{
				Label: "全局 · LLM 最大 tokens", Group: "global", Key: "llm_max_tokens",
				FieldType: "int", IntValue: gc.LLM.MaxTokens,
			})
		}
		for k, v := range gc.ApprovalPolicy {
			sp.Fields = append(sp.Fields, SettingsField{
				Label: "全局 · 审批: " + k, Group: "global", Key: "approval_policy",
				FieldType: "enum", EnumValue: v, EnumOptions: ApprovalLevels, EnumKey: k,
			})
		}
	}
}

func (sp *SettingsPanel) CursorUp() {
	if sp.Selected > 0 {
		sp.Selected--
	}
}

func (sp *SettingsPanel) CursorDown() {
	if sp.Selected < len(sp.Fields)-1 {
		sp.Selected++
	}
}

func (sp *SettingsPanel) SelectedField() *SettingsField {
	if sp.Selected < 0 || sp.Selected >= len(sp.Fields) {
		return nil
	}
	return &sp.Fields[sp.Selected]
}

func (sp *SettingsPanel) StartEditing() {
	sp.Editing = true
	sp.EditBuffer = ""
}

func (sp *SettingsPanel) CancelEditing() {
	sp.Editing = false
	sp.EditBuffer = ""
}

func (sp *SettingsPanel) ConfirmEditing() (*SettingsField, int, bool) {
	sp.Editing = false
	f := sp.SelectedField()
	if f == nil || f.FieldType != "int" {
		return nil, 0, false
	}
	val, err := strconv.Atoi(sp.EditBuffer)
	if err != nil || val < 0 {
		sp.EditBuffer = ""
		return nil, 0, false
	}
	sp.EditBuffer = ""
	f.IntValue = &val
	return f, val, true
}

func (sp *SettingsPanel) CycleEnum() *SettingsField {
	f := sp.SelectedField()
	if f == nil || f.FieldType != "enum" {
		return nil
	}
	for i, opt := range f.EnumOptions {
		if opt == f.EnumValue {
			next := (i + 1) % len(f.EnumOptions)
			f.EnumValue = f.EnumOptions[next]
			return f
		}
	}
	f.EnumValue = f.EnumOptions[0]
	return f
}

func (sp *SettingsPanel) BuildProjectSaveBody() map[string]interface{} {
	body := map[string]interface{}{}
	approvalPolicy := map[string]string{}
	for _, f := range sp.Fields {
		if f.Group != "project" {
			continue
		}
		if f.FieldType == "int" && f.IntValue != nil {
			body[f.Key] = *f.IntValue
		}
		if f.FieldType == "enum" && f.EnumKey != "" {
			approvalPolicy[f.EnumKey] = f.EnumValue
		}
	}
	if len(approvalPolicy) > 0 {
		body["approval_policy"] = approvalPolicy
	}
	return body
}

func (sp *SettingsPanel) BuildGlobalSaveBody() map[string]interface{} {
	body := map[string]interface{}{}
	approvalPolicy := map[string]string{}
	llm := map[string]interface{}{}
	for _, f := range sp.Fields {
		if f.Group != "global" {
			continue
		}
		if f.FieldType == "int" {
			if f.Key == "llm_max_tokens" {
				if f.IntValue != nil {
					llm["max_tokens"] = *f.IntValue
				}
			} else if f.IntValue != nil {
				body[f.Key] = *f.IntValue
			}
		}
		if f.FieldType == "enum" && f.EnumKey != "" {
			approvalPolicy[f.EnumKey] = f.EnumValue
		}
	}
	if len(llm) > 0 {
		body["llm"] = llm
	}
	if len(approvalPolicy) > 0 {
		body["approval_policy"] = approvalPolicy
	}
	return body
}

func (sp *SettingsPanel) Render() string {
	w := sp.Width
	if w < 40 {
		w = 40
	}
	innerW := w - 4

	label := lipgloss.NewStyle().Foreground(components.ColorAccent()).Bold(true)
	accent := lipgloss.NewStyle().Foreground(components.ColorAccent())
	muted := lipgloss.NewStyle().Foreground(components.ColorMuted())
	text := lipgloss.NewStyle().Foreground(components.ColorText())
	editing := lipgloss.NewStyle().Foreground(components.ColorSuccess()).Bold(true)

	var lines []string
	lines = append(lines, components.PanelHeaderStyle(innerW).Render(" Settings"))

	lastGroup := ""
	for i, f := range sp.Fields {
		groupName := "项目设置"
		if f.Group == "global" {
			groupName = "全局设置"
		}
		if groupName != lastGroup {
			if lastGroup != "" {
				lines = append(lines, "")
			}
			lines = append(lines, "  "+label.Render(groupName))
			lastGroup = groupName
		}

		prefix := "  "
		valStyle := muted
		if i == sp.Selected {
			prefix = accent.Render(" \u25b8")
			valStyle = accent
		}

		valStr := ""
		if f.FieldType == "int" {
			if f.IntValue != nil {
				valStr = fmt.Sprintf("%d", *f.IntValue)
			} else {
				valStr = "未设置"
			}
		} else {
			label := ApprovalLevelLabels[f.EnumValue]
			if label == "" {
				label = f.EnumValue
			}
			valStr = label
		}

		labelText := text.Render(f.Label)
		valText := valStyle.Render(valStr)
		line := fmt.Sprintf("%s%s: %s", prefix, labelText, valText)
		lines = append(lines, line)
	}

	if sp.Editing {
		lines = append(lines, "")
		lines = append(lines, "  "+editing.Render("输入新值: "+sp.EditBuffer+"\u2588"))
		lines = append(lines, "  "+muted.Render("[Enter] 确认  [Esc] 取消"))
	} else {
		footer := "[\u2191\u2193] 导航  [Enter] 编辑数字  [Space] 切换审批  [Esc] 关闭"
		lines = append(lines, components.PanelFooterStyle(innerW).Render(footer))
	}

	return strings.Join(lines, "\n")
}
