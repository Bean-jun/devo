package overlays

import "testing"

func TestOverlayStack_OpenClose(t *testing.T) {
	os := OverlayStack{}
	if os.IsOpen() {
		t.Error("新 OverlayStack 应为关闭状态")
	}

	os.Open(OverlayHelp)
	if !os.IsOpen() {
		t.Error("Open() 后应为打开状态")
	}
	if os.Current != OverlayHelp {
		t.Error("Open(OverlayHelp) 后 Current 应为 OverlayHelp")
	}

	os.Close()
	if os.IsOpen() {
		t.Error("Close() 后应为关闭状态")
	}
}

func TestOverlayStack_CloseTwice(t *testing.T) {
	os := OverlayStack{}
	os.Open(OverlayHelp)
	if !os.Close() {
		t.Error("首次 Close() 应返回 true")
	}
	if os.Close() {
		t.Error("第二次 Close() 应返回 false")
	}
}

func TestOverlayStack_Nested(t *testing.T) {
	os := OverlayStack{}
	os.Open(OverlayHelp)
	os.Open(OverlayCommand)
	if os.Current != OverlayCommand {
		t.Error("打开命令面板后 Current 应为 OverlayCommand")
	}
	os.Close()
	if os.Current != OverlayHelp {
		t.Error("关闭命令面板后应回到帮助面板")
	}
	os.Close()
	if os.IsOpen() {
		t.Error("关闭所有面板后应为关闭状态")
	}
}
