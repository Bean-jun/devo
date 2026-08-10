package overlays

import (
	"strings"
	"testing"
)

func TestReasoningPicker_NewDisabled(t *testing.T) {
	rp := NewReasoningPicker(false, "medium")
	if rp.Selected != 0 {
		t.Errorf("关闭时 Selected 应为 0, got %d", rp.Selected)
	}
	if rp.Enabled {
		t.Error("关闭时 Enabled 应为 false")
	}
}

func TestReasoningPicker_NewEnabled(t *testing.T) {
	rp := NewReasoningPicker(true, "medium")
	if rp.Selected != 2 {
		t.Errorf("medium 时 Selected 应为 2, got %d", rp.Selected)
	}
	if !rp.Enabled {
		t.Error("启用时 Enabled 应为 true")
	}
}

func TestReasoningPicker_NewEnabledHigh(t *testing.T) {
	rp := NewReasoningPicker(true, "high")
	if rp.Selected != 3 {
		t.Errorf("high 时 Selected 应为 3, got %d", rp.Selected)
	}
}

func TestReasoningPicker_NewEnabledLow(t *testing.T) {
	rp := NewReasoningPicker(true, "low")
	if rp.Selected != 1 {
		t.Errorf("low 时 Selected 应为 1, got %d", rp.Selected)
	}
}

func TestReasoningPicker_CursorUp(t *testing.T) {
	rp := NewReasoningPicker(true, "medium")
	rp.CursorUp()
	if rp.Selected != 1 {
		t.Errorf("从 medium 向上后应为 1, got %d", rp.Selected)
	}
	rp.CursorUp()
	if rp.Selected != 0 {
		t.Errorf("从 low 向上后应为 0, got %d", rp.Selected)
	}
	rp.CursorUp()
	if rp.Selected != 0 {
		t.Error("在顶部 CursorUp() 不应超出边界")
	}
}

func TestReasoningPicker_CursorDown(t *testing.T) {
	rp := NewReasoningPicker(true, "medium")
	rp.CursorDown()
	if rp.Selected != 3 {
		t.Errorf("从 medium 向下后应为 3, got %d", rp.Selected)
	}
	rp.CursorDown()
	if rp.Selected != 3 {
		t.Error("在底部 CursorDown() 不应超出边界")
	}
}

func TestReasoningPicker_SelectedOption(t *testing.T) {
	rp := NewReasoningPicker(true, "low")
	opt := rp.SelectedOption()
	if opt.Value != "low" {
		t.Errorf("应选中 low, got %s", opt.Value)
	}
}

func TestReasoningPicker_Render(t *testing.T) {
	rp := NewReasoningPicker(true, "medium")
	rp.Width = 40
	result := rp.Render()

	if !strings.Contains(result, "思维链") {
		t.Error("渲染结果应包含标题")
	}
	if !strings.Contains(result, "当前: medium") {
		t.Error("渲染结果应包含当前状态")
	}
	if !strings.Contains(result, "关闭") {
		t.Error("渲染结果应包含关闭选项")
	}
	if !strings.Contains(result, "低") {
		t.Error("渲染结果应包含低选项")
	}
	if !strings.Contains(result, "中") {
		t.Error("渲染结果应包含中选项")
	}
	if !strings.Contains(result, "高") {
		t.Error("渲染结果应包含高选项")
	}
}

func TestReasoningPicker_RenderDisabled(t *testing.T) {
	rp := NewReasoningPicker(false, "medium")
	rp.Width = 40
	result := rp.Render()

	if !strings.Contains(result, "当前: 关闭") {
		t.Error("关闭时渲染结果应包含当前: 关闭")
	}
}

func TestReasoningPicker_Options(t *testing.T) {
	if len(ReasoningOptions) != 4 {
		t.Errorf("应有 4 个选项, got %d", len(ReasoningOptions))
	}
	if ReasoningOptions[0].Value != "off" {
		t.Error("第一个选项应为 off")
	}
	if ReasoningOptions[1].Value != "low" {
		t.Error("第二个选项应为 low")
	}
	if ReasoningOptions[2].Value != "medium" {
		t.Error("第三个选项应为 medium")
	}
	if ReasoningOptions[3].Value != "high" {
		t.Error("第四个选项应为 high")
	}
}
