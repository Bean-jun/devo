package overlays

import (
	"strings"
	"testing"
)

func TestSkillsPanel_Render(t *testing.T) {
	sp := NewSkillsPanel()
	sp.Skills = []SkillEntry{
		{Name: "code-reviewer", Description: "代码审查", Enabled: true},
	}
	sp.Width = 80
	result := sp.Render()
	if !strings.Contains(result, "技能管理") {
		t.Error("渲染结果应包含标题 '技能管理'")
	}
	if !strings.Contains(result, "code-reviewer") {
		t.Error("渲染结果应包含技能 code-reviewer")
	}
}

func TestSkillsPanel_CursorNavigation(t *testing.T) {
	sp := NewSkillsPanel()
	sp.Skills = []SkillEntry{{Name: "a"}, {Name: "b"}}
	sp.CursorDown()
	if sp.Selected != 1 {
		t.Error("CursorDown() 后 Selected 应为 1")
	}
	sp.CursorUp()
	if sp.Selected != 0 {
		t.Error("CursorUp() 后 Selected 应为 0")
	}
}

func TestSkillsPanel_Toggle(t *testing.T) {
	sp := NewSkillsPanel()
	sp.Skills = []SkillEntry{{Name: "a", Enabled: false}}
	sp.Toggle()
	if !sp.Skills[0].Enabled {
		t.Error("Toggle() 后技能应启用")
	}
	sp.Toggle()
	if sp.Skills[0].Enabled {
		t.Error("再次 Toggle() 后技能应停用")
	}
}

func TestSkillsPanel_EditMode(t *testing.T) {
	sp := NewSkillsPanel()
	sp.StartEditing()
	if !sp.Editing {
		t.Error("StartEditing() 后 Editing 应为 true")
	}
	if sp.EditBuffer != "" {
		t.Error("StartEditing() 后 EditBuffer 应为空")
	}

	sp.EditBuffer = "my-skill"
	v := sp.ConfirmEditing()
	if v != "my-skill" {
		t.Errorf("ConfirmEditing() 应返回 my-skill, got %s", v)
	}
	if sp.Editing {
		t.Error("ConfirmEditing() 后 Editing 应为 false")
	}
	if sp.EditBuffer != "" {
		t.Error("ConfirmEditing() 后 EditBuffer 应为空")
	}
}

func TestSkillsPanel_CancelEditing(t *testing.T) {
	sp := NewSkillsPanel()
	sp.StartEditing()
	sp.EditBuffer = "test"
	sp.CancelEditing()
	if sp.Editing {
		t.Error("CancelEditing() 后 Editing 应为 false")
	}
	if sp.EditBuffer != "" {
		t.Error("CancelEditing() 后 EditBuffer 应为空")
	}
}

func TestSkillsPanel_RenderEditing(t *testing.T) {
	sp := NewSkillsPanel()
	sp.Width = 80
	sp.StartEditing()
	sp.EditBuffer = "test-skill"
	result := sp.Render()
	if !strings.Contains(result, "test-skill") {
		t.Error("渲染结果应包含编辑缓冲区内容")
	}
	if !strings.Contains(result, "确认安装") {
		t.Error("渲染结果应包含确认提示")
	}
}
