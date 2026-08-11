package overlays

import (
	"strings"
	"testing"
)

func TestApprovalModal_Render(t *testing.T) {
	am := ApprovalModal{
		Width:     80,
		Operation: "delete",
		Risk:      "HIGH",
		Diff:      "- line1\n+ line2",
	}
	result := am.Render()
	if !strings.Contains(result, "Approval Required") {
		t.Error("渲染结果应包含标题 'Approval Required'")
	}
	if !strings.Contains(result, "delete") {
		t.Error("渲染结果应包含操作类型")
	}
	if !strings.Contains(result, "HIGH") {
		t.Error("渲染结果应包含风险等级")
	}
}
