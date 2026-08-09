package components

import (
	"strings"
	"testing"
)

func TestToast_ShowSuccess(t *testing.T) {
	toast := Toast{Width: 80}
	toast.Show("操作成功", false)
	if toast.Type != "success" {
		t.Error("非错误 Toast 类型应为 success")
	}
	if toast.Duration != 3 {
		t.Errorf("success Toast 持续时间应为 3, got %d", toast.Duration)
	}
}

func TestToast_ShowError(t *testing.T) {
	toast := Toast{Width: 80}
	toast.Show("操作失败", true)
	if toast.Type != "error" {
		t.Error("错误 Toast 类型应为 error")
	}
	if toast.Duration != 5 {
		t.Errorf("error Toast 持续时间应为 5, got %d", toast.Duration)
	}
}

func TestToast_Hide(t *testing.T) {
	toast := Toast{Width: 80}
	toast.Show("test", false)
	toast.Hide()
	if toast.Duration != 0 {
		t.Error("Hide() 后 Duration 应为 0")
	}
}

func TestToast_Tick(t *testing.T) {
	toast := Toast{Width: 80, Duration: 3}
	toast.Tick()
	if toast.Duration != 2 {
		t.Errorf("Tick() 后 Duration 应为 2, got %d", toast.Duration)
	}
}

func TestToast_Render(t *testing.T) {
	toast := Toast{Width: 80, Message: "测试消息", Duration: 3, Type: "success"}
	result := toast.Render()
	if !strings.Contains(result, "测试消息") {
		t.Error("Toast 渲染结果应包含消息内容")
	}
}

func TestToast_RenderHidden(t *testing.T) {
	toast := Toast{Width: 80, Duration: 0}
	result := toast.Render()
	if result != "" {
		t.Error("Duration 为 0 时 Render() 应返回空字符串")
	}
}
