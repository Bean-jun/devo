package main

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// ─── 消息渲染测试 ───

func TestRenderUserMessage(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	msg := Message{
		Role:    RoleUser,
		Content: "帮我修复空指针问题",
		Time:    "14:32",
	}

	result := r.renderUser(msg)

	if !strings.Contains(result, "⏺") {
		t.Error("用户消息应包含 ⏺ 前缀")
	}
	if !strings.Contains(result, "帮我修复空指针问题") {
		t.Error("用户消息应包含原文内容")
	}
	if !strings.Contains(result, "14:32") {
		t.Error("用户消息应包含时间戳")
	}
}

func TestRenderAssistantMessage(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	msg := Message{
		Role:     RoleAssistant,
		Content:  "我来分析代码",
		Thinking: "需要先读取文件\n找到具体位置",
		ToolCalls: []ToolCall{
			{Name: "read_file", Summary: "utils.go", Status: "success", Duration: "0.3s"},
		},
		Time: "14:31",
	}

	result := r.renderAssistant(msg)

	if !strings.Contains(result, "⏺") {
		t.Error("助手消息应包含 ⏺ 前缀")
	}
	if !strings.Contains(result, "我来分析代码") {
		t.Error("助手消息应包含原文内容")
	}
	if !strings.Contains(result, "14:31") {
		t.Error("助手消息应包含时间戳")
	}
}

func TestRenderSystemMessage(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	msg := Message{
		Role:    RoleSystem,
		Content: "session 已创建",
		Time:    "14:30",
	}

	result := r.renderSystem(msg)

	if !strings.Contains(result, "session 已创建") {
		t.Error("系统消息应包含原文内容")
	}
}

func TestRenderThinking(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	msg := Message{
		Role:     RoleAssistant,
		Content:  "回复内容",
		Thinking: "需要先读取文件\n找到具体位置\n添加 nil 检查",
		Time:     "14:31",
	}

	result := r.renderAssistant(msg)

	if !strings.Contains(result, "·") {
		t.Error("思考过程应包含 · 前缀")
	}
	if !strings.Contains(result, "需要先读取文件") {
		t.Error("思考过程应包含思考内容")
	}
}

func TestRenderToolCallSuccess(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	tc := ToolCall{
		Name: "read_file", Summary: "utils.go",
		Status: "success", Duration: "0.3s",
		Diff: "+ new line\n- old line",
	}

	result := r.renderToolCall(tc)

	if !strings.Contains(result, "✓") {
		t.Error("成功的工具调用应包含 ✓ 前缀")
	}
	if !strings.Contains(result, "read_file") {
		t.Error("工具调用应包含工具名")
	}
	if !strings.Contains(result, "utils.go") {
		t.Error("工具调用应包含摘要信息")
	}
}

func TestRenderToolCallError(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	tc := ToolCall{
		Name: "execute_command", Summary: "ping",
		Status: "error", Duration: "5.1s",
	}

	result := r.renderToolCall(tc)

	if !strings.Contains(result, "✗") {
		t.Error("失败的工具调用应包含 ✗ 前缀")
	}
}

func TestRenderToolCallPending(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	tc := ToolCall{
		Name: "write_file", Summary: "config.yaml",
		Status: "pending", Duration: "",
	}

	result := r.renderToolCall(tc)

	if !strings.Contains(result, "⏺") {
		t.Error("待审批的工具调用应包含 ⏺ 前缀")
	}
}

func TestRenderToolCallExecuting(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	tc := ToolCall{
		Name: "write_file", Summary: "config.yaml",
		Status: "executing", Duration: "",
	}

	result := r.renderToolCall(tc)

	if !strings.Contains(result, "⏺") {
		t.Error("执行中的工具调用应包含 ⏺ 前缀")
	}
}

// ─── CJK 换行测试 ───

func TestWrapCJK_Ascii(t *testing.T) {
	result := wrapCJK("Hello World", 5)
	if !strings.Contains(result, "Hello") && !strings.Contains(result, "World") {
		t.Errorf("ASCII 换行应包含原文: got %q", result)
	}
	for _, line := range strings.Split(result, "\n") {
		if displayWidth(line) > 5 {
			t.Errorf("行宽超出限制: %q (宽度 %d)", line, displayWidth(line))
		}
	}
}

func TestWrapCJK_Chinese(t *testing.T) {
	result := wrapCJK("你好世界测试", 4)
	expected := "你好\n世界\n测试"
	if result != expected {
		t.Errorf("中文换行错误:\n   got: %q\n  want: %q", result, expected)
	}
}

func TestWrapCJK_Mixed(t *testing.T) {
	result := wrapCJK("Hello你好World世界", 10)
	if !strings.Contains(result, "Hello") && !strings.Contains(result, "World") {
		t.Errorf("中英混合换行应包含原文: got %q", result)
	}
	for _, line := range strings.Split(result, "\n") {
		if displayWidth(line) > 10 {
			t.Errorf("行宽超出限制: %q (宽度 %d)", line, displayWidth(line))
		}
	}
}

func TestWrapCJK_LongChinese(t *testing.T) {
	longText := "这是一段很长的中文文本，用来测试自动换行功能是否正常工作。"
	result := wrapCJK(longText, 10)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		w := displayWidth(line)
		if w > 10 {
			t.Errorf("行宽超出限制: %q (宽度 %d > 10)", line, w)
		}
	}
}

func TestWrapCJK_EnglishLongWord(t *testing.T) {
	result := wrapCJK("supercalifragilisticexpialidocious", 10)
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		w := displayWidth(line)
		if w > 10 {
			t.Errorf("行宽超出限制: %q (宽度 %d > 10)", line, w)
		}
	}
}

func TestWrapCJK_Empty(t *testing.T) {
	result := wrapCJK("", 10)
	if result != "" {
		t.Errorf("空字符串换行应返回空: got %q", result)
	}
}

func TestWrapCJK_NewlineHandling(t *testing.T) {
	result := wrapCJK("line1\nline2\nline3", 10)
	if !strings.Contains(result, "\n") {
		t.Error("应保留原有换行符")
	}
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("应保留 3 行: got %d lines", len(lines))
	}
}

// ─── 显示宽度测试 ───

func TestDisplayWidth_Ascii(t *testing.T) {
	w := displayWidth("Hello")
	if w != 5 {
		t.Errorf("ASCII 宽度应为 5: got %d", w)
	}
}

func TestDisplayWidth_Chinese(t *testing.T) {
	w := displayWidth("你好")
	if w != 4 {
		t.Errorf("中文宽度应为 4: got %d", w)
	}
}

func TestDisplayWidth_Mixed(t *testing.T) {
	w := displayWidth("Hello你好")
	if w != 9 {
		t.Errorf("混合宽度应为 9: got %d", w)
	}
}

func TestDisplayWidth_Empty(t *testing.T) {
	w := displayWidth("")
	if w != 0 {
		t.Errorf("空字符串宽度应为 0: got %d", w)
	}
}

// ─── 渲染缓存测试 ───

func TestRenderCache_Invalidate(t *testing.T) {
	rc := newRenderCache()
	if rc.dirty != 0 {
		t.Error("新缓存应标记为 dirty=0")
	}

	rc.invalidate(3)
	if rc.dirty != 0 {
		t.Errorf("dirty=0 时 invalidate(3) 应保持 0: got %d", rc.dirty)
	}

	rc.invalidate(5)
	if rc.dirty != 0 {
		t.Errorf("dirty=0 时 invalidate(5) 应保持 0: got %d", rc.dirty)
	}

	rc.dirty = 5
	rc.invalidate(3)
	if rc.dirty != 3 {
		t.Errorf("dirty=5 时 invalidate(3) 应更新为 3: got %d", rc.dirty)
	}

	rc.dirty = -1
	rc.invalidate(2)
	if rc.dirty != 2 {
		t.Errorf("dirty=-1 时 invalidate(2) 应设置为 2: got %d", rc.dirty)
	}
}

func TestRenderCache_InvalidateAll(t *testing.T) {
	rc := newRenderCache()
	rc.invalidate(5)
	rc.invalidateAll()
	if rc.dirty != 0 {
		t.Errorf("invalidateAll 后 dirty 应为 0: got %d", rc.dirty)
	}
}

func TestRenderCache_Clean(t *testing.T) {
	rc := newRenderCache()
	if rc.isClean() {
		t.Error("新缓存不应是 clean 的")
	}
	rc.dirty = -1
	if !rc.isClean() {
		t.Error("dirty=-1 时应该是 clean 的")
	}
}

// ─── 消息渲染器集成测试 ───

func TestRenderer_RenderMultipleMessages(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	messages := []Message{
		{Role: RoleSystem, Content: "session started", Time: "14:00"},
		{Role: RoleUser, Content: "Hello", Time: "14:01"},
		{Role: RoleAssistant, Content: "Hi there", Time: "14:01"},
	}

	result := stripANSI(r.render(messages))

	if !strings.Contains(result, "session started") {
		t.Error("应包含系统消息")
	}
	if !strings.Contains(result, "Hello") {
		t.Error("应包含用户消息")
	}
	if !strings.Contains(result, "Hi there") {
		t.Errorf("应包含助手消息内容，实际渲染结果:\n%s", result)
	}
	if !strings.Contains(result, "14:01") {
		t.Error("应包含时间戳")
	}
}

func TestRenderer_CacheReuse(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	messages := []Message{
		{Role: RoleUser, Content: "Test", Time: "14:00"},
	}

	result1 := r.render(messages)
	result2 := r.render(messages)

	if result1 != result2 {
		t.Error("缓存命中时两次渲染结果应相同")
	}
}

func TestRenderer_CacheInvalidation(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	messages := []Message{
		{Role: RoleUser, Content: "Test", Time: "14:00"},
	}

	result1 := r.render(messages)
	r.cache.invalidate(0)
	result2 := r.render(messages)

	if result1 != result2 {
		t.Error("缓存失效后重新渲染结果应相同")
	}
}

func TestRenderer_EmptyMessages(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)
	result := r.render([]Message{})
	if result != "" {
		t.Errorf("空消息列表应返回空字符串: got %q", result)
	}
}

func TestRenderer_LargeMessageSet(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	messages := make([]Message, 100)
	for i := 0; i < 100; i++ {
		messages[i] = Message{
			Role:    RoleUser,
			Content: "测试消息 " + strings.Repeat("x", i%20),
			Time:    "14:00",
		}
	}

	result := r.render(messages)
	if !strings.Contains(result, "测试消息") {
		t.Error("大批量消息渲染应包含消息内容")
	}

	cached := r.render(messages)
	if result != cached {
		t.Error("缓存命中时大批量渲染结果应相同")
	}
}

// ─── 用户消息左对齐测试 ───

func TestRenderUser_LeftAligned(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	msg := Message{
		Role:    RoleUser,
		Content: "测试消息",
		Time:    "14:32",
	}

	result := r.renderUser(msg)

	if strings.HasPrefix(result, " ") {
		t.Error("用户消息不应有前导空格（左对齐）")
	}
	if !strings.Contains(result, "14:32") {
		t.Error("用户消息应包含时间戳")
	}
	if !strings.Contains(result, "⏺") {
		t.Error("用户消息应包含前缀符号")
	}
}

func TestRenderUser_NoRightPad(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	msg := Message{
		Role:    RoleUser,
		Content: "短消息",
		Time:    "14:32",
	}

	result := r.renderUser(msg)

	lines := strings.Split(result, "\n")
	for _, line := range lines {
		trimmed := strings.TrimRight(line, " ")
		if len(line)-len(trimmed) > 10 {
			t.Errorf("用户消息不应有大量尾部空格: line=%q", line)
		}
	}
}

// ─── 长文本换行测试 ───

func TestRenderUser_WordWrap(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(40)

	longText := "这是一段很长的消息文本，需要测试是否能够正确换行显示"
	msg := Message{
		Role:    RoleUser,
		Content: longText,
		Time:    "14:32",
	}

	result := stripANSI(r.renderUser(msg))
	lines := strings.Split(result, "\n")

	// 去掉时间戳行
	contentLines := lines[:len(lines)-1]

	if len(contentLines) < 2 {
		t.Errorf("长文本应换行，实际只有 %d 行:\n%s", len(contentLines), result)
	}

	// 后续行应有缩进，第一行以 ⏺ 开头
	if !strings.Contains(contentLines[0], "⏺") {
		t.Error("第一行应包含 ⏺ 前缀")
	}
	for i := 1; i < len(contentLines); i++ {
		if !strings.HasPrefix(contentLines[i], "  ") {
			t.Errorf("第 %d 行应有缩进: %q", i+1, contentLines[i])
		}
	}
}

func TestRenderUser_ShortTextNoWrap(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(80)

	msg := Message{
		Role:    RoleUser,
		Content: "短消息",
		Time:    "14:32",
	}

	result := stripANSI(r.renderUser(msg))
	lines := strings.Split(result, "\n")

	// 只有内容行 + 时间戳行
	if len(lines) != 2 {
		t.Errorf("短消息不应换行，实际 %d 行:\n%s", len(lines), result)
	}
}

func TestRenderAssistant_WordWrap(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(30)

	longText := "这是一段非常非常长的助手回复消息文本内容，需要测试换行功能是否正确工作"
	msg := Message{
		Role:    RoleAssistant,
		Content: longText,
		Time:    "14:33",
	}

	result := stripANSI(r.renderAssistant(msg))
	lines := strings.Split(result, "\n")

	contentLines := lines[:len(lines)-1]

	if len(contentLines) < 2 {
		t.Errorf("长文本应换行，实际只有 %d 行:\n%s", len(contentLines), result)
	}

	if !strings.Contains(contentLines[0], "⏺") {
		t.Error("第一行应包含 ⏺ 前缀")
	}
	for i := 1; i < len(contentLines); i++ {
		if !strings.HasPrefix(contentLines[i], "  ") {
			t.Errorf("第 %d 行应有缩进: %q", i+1, contentLines[i])
		}
	}
}

func TestRenderUser_EnglishWordWrap(t *testing.T) {
	r := newMsgRenderer()
	r.setWidth(30)

	longText := "This is a very long English message that should wrap across multiple lines properly"
	msg := Message{
		Role:    RoleUser,
		Content: longText,
		Time:    "14:32",
	}

	result := stripANSI(r.renderUser(msg))
	lines := strings.Split(result, "\n")
	contentLines := lines[:len(lines)-1]

	if len(contentLines) < 3 {
		t.Errorf("英文长文本应换行，实际只有 %d 行:\n%s", len(contentLines), result)
	}
}
