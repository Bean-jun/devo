package components

import (
	"strings"
	"testing"
)

func TestMessageViewport_AddStreamingChunk(t *testing.T) {
	mv := NewMessageViewport()

	mv.AddStreamingChunk("Hello ")
	mv.AddStreamingChunk("World")

	if !mv.StreamingActive {
		t.Error("StreamingActive should be true after AddStreamingChunk")
	}

	content := mv.StreamingBuffer.String()
	if content != "Hello World" {
		t.Errorf("StreamingBuffer = %q, want %q", content, "Hello World")
	}
}

func TestMessageViewport_FinalizeStreaming(t *testing.T) {
	mv := NewMessageViewport()

	mv.AddStreamingChunk("Hello ")
	mv.AddStreamingChunk("Streaming")

	result := mv.FinalizeStreaming()

	if result != "Hello Streaming" {
		t.Errorf("FinalizeStreaming = %q, want %q", result, "Hello Streaming")
	}

	if mv.StreamingActive {
		t.Error("StreamingActive should be false after FinalizeStreaming")
	}

	if mv.StreamingBuffer.Len() != 0 {
		t.Error("StreamingBuffer should be empty after FinalizeStreaming")
	}

	found := false
	for _, msg := range mv.Messages {
		if msg.Role == "assistant" && msg.Content == "Hello Streaming" {
			found = true
			break
		}
	}
	if !found {
		t.Error("streaming content should be added as assistant message after FinalizeStreaming")
	}
}

func TestMessageViewport_FinalizeStreamingEmpty(t *testing.T) {
	mv := NewMessageViewport()

	result := mv.FinalizeStreaming()

	if result != "" {
		t.Errorf("FinalizeStreaming on empty = %q, want empty", result)
	}

	if mv.StreamingActive {
		t.Error("StreamingActive should be false")
	}

	msgCount := len(mv.Messages)
	if msgCount != 0 {
		t.Errorf("no messages should be added, got %d", msgCount)
	}
}

func TestMessageViewport_FinalizeStreamingEmptyContent(t *testing.T) {
	mv := NewMessageViewport()
	mv.StreamingActive = true

	result := mv.FinalizeStreaming()

	if result != "" {
		t.Errorf("FinalizeStreaming with active but empty = %q, want empty", result)
	}

	msgCount := len(mv.Messages)
	if msgCount != 0 {
		t.Errorf("no messages should be added for empty content, got %d", msgCount)
	}
}

func TestMessageViewport_ToolCardStage(t *testing.T) {
	mv := NewMessageViewport()

	card := ToolCardData{
		ToolName: "exec_python",
		Params:   "print('hello')",
		Stage:    "pending",
	}
	mv.AddToolCard(card)

	mv.UpdateToolCardStage("exec_python", "executing")
	if mv.ToolCards[0].Stage != "executing" {
		t.Errorf("stage = %q, want %q", mv.ToolCards[0].Stage, "executing")
	}

	mv.UpdateToolCardStage("exec_python", "done")
	if mv.ToolCards[0].Stage != "done" {
		t.Errorf("stage = %q, want %q", mv.ToolCards[0].Stage, "done")
	}

	mv.UpdateToolCardStage("nonexistent", "executing")
	if mv.ToolCards[0].Stage != "done" {
		t.Error("stage should not change for nonexistent tool")
	}
}

func TestMessageViewport_AppendToolCardChunk(t *testing.T) {
	mv := NewMessageViewport()

	card := ToolCardData{
		ToolName: "exec_python",
		Params:   "print('hello')",
	}
	mv.AddToolCard(card)

	mv.AppendToolCardChunk("exec_python", "Line 1: Hello\n")
	if len(mv.ToolCards[0].OutputChunks) != 1 {
		t.Fatalf("OutputChunks length = %d, want 1", len(mv.ToolCards[0].OutputChunks))
	}
	if mv.ToolCards[0].OutputChunks[0] != "Line 1: Hello\n" {
		t.Errorf("chunk = %q, want %q", mv.ToolCards[0].OutputChunks[0], "Line 1: Hello\n")
	}

	mv.AppendToolCardChunk("exec_python", "Line 2: World\n")
	if len(mv.ToolCards[0].OutputChunks) != 2 {
		t.Fatalf("OutputChunks length = %d, want 2", len(mv.ToolCards[0].OutputChunks))
	}

	if mv.ToolCards[0].Stage != "executing" {
		t.Errorf("stage should be auto-set to 'executing', got %q", mv.ToolCards[0].Stage)
	}
}

func TestMessageViewport_AppendToolCardChunk_Nonexistent(t *testing.T) {
	mv := NewMessageViewport()

	mv.AppendToolCardChunk("nonexistent", "output")

	if len(mv.ToolCards) != 0 {
		t.Error("should not create tool card for nonexistent tool")
	}
}

func TestMessageViewport_StreamingWithToolCardsInterleaved(t *testing.T) {
	mv := NewMessageViewport()

	// Simulate: streaming starts, then tool call, then more streaming
	mv.AddStreamingChunk("I will run a Python script.\n")

	card := ToolCardData{
		ToolName: "exec_python",
		Params:   "print('hello')",
		Stage:    "executing",
	}
	mv.AddToolCard(card)
	mv.AppendToolCardChunk("exec_python", "hello\n")

	mv.AddStreamingChunk("The script ran successfully.\n")

	result := mv.FinalizeStreaming()
	if !strings.Contains(result, "I will run a Python script") {
		t.Error("streaming content should contain first part")
	}
	if !strings.Contains(result, "The script ran successfully") {
		t.Error("streaming content should contain second part after tool call")
	}

	if len(mv.ToolCards) != 1 {
		t.Errorf("ToolCards length = %d, want 1", len(mv.ToolCards))
	}
}

func TestMessageViewport_RenderIncludesStreamingContent(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	mv.AddStreamingChunk("Streaming text...")

	view := mv.View()
	if !strings.Contains(view, "Streaming text...") {
		t.Error("view should contain streaming content")
	}
}

func TestMessageViewport_RenderIncludesToolCardOutput(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	card := ToolCardData{
		ToolName:     "exec_python",
		Params:       "print('hello')",
		Stage:        "executing",
		Expanded:     true,
		OutputChunks: []string{"hello\n"},
	}
	mv.AddToolCard(card)

	view := mv.View()
	if !strings.Contains(view, "exec_python") {
		t.Error("view should contain tool name")
	}
	if !strings.Contains(view, "执行中") {
		t.Error("view should contain stage indicator")
	}
	if !strings.Contains(view, "hello") {
		t.Error("view should contain output chunk")
	}
}

func TestMessageViewport_RenderToolCardDone(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	card := ToolCardData{
		ToolName:     "exec_python",
		Params:       "print('hello')",
		Stage:        "done",
		Result:       "Execution completed",
		Success:      true,
		Expanded:     true,
		OutputChunks: []string{"hello\n", "world\n"},
	}
	mv.AddToolCard(card)

	view := mv.View()
	if !strings.Contains(view, "exec_python") {
		t.Error("view should contain tool name")
	}
	if !strings.Contains(view, "Execution completed") {
		t.Error("view should contain result")
	}

	if strings.Contains(view, "done") {
		t.Error("view should not show 'done' stage in header")
	}
}

func TestToolCard_Grouping(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	mv.AddToolCard(ToolCardData{ToolName: "tool_a", Stage: "executing"})
	mv.AddToolCard(ToolCardData{ToolName: "tool_b", Stage: "executing"})

	if mv.ToolCards[0].GroupID == "" {
		t.Error("consecutive tool cards without result should have GroupID")
	}
	if mv.ToolCards[1].GroupID != mv.ToolCards[0].GroupID {
		t.Error("consecutive tool cards should share the same GroupID")
	}
}

func TestToolCard_GroupRendering(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	mv.AddToolCard(ToolCardData{ToolName: "tool_a", Stage: "executing"})
	mv.AddToolCard(ToolCardData{ToolName: "tool_b", Stage: "executing"})
	mv.AddToolCard(ToolCardData{ToolName: "tool_c", Stage: "executing"})

	view := mv.View()
	if !strings.Contains(view, "工具调用 (3)") {
		t.Error("group of 3 tools should show header")
	}
}

func TestToolCard_NoGroupForSingle(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	mv.AddToolCard(ToolCardData{ToolName: "tool_a", Stage: "executing"})
	mv.UpdateToolCard("tool_a", true, "done", "1s")

	mv.AddToolCard(ToolCardData{ToolName: "tool_b", Stage: "executing"})

	if mv.ToolCards[1].GroupID != "" {
		t.Error("tool after completed one should not be grouped")
	}
}

func TestToolCard_ExpandCollapse(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	card := ToolCardData{
		ToolName: "exec_python",
		Params:   "print('hello')",
		Stage:    "executing",
	}
	mv.AddToolCard(card)

	if mv.ToolCards[0].Expanded {
		t.Error("card should not be expanded by default")
	}

	mv.ToggleToolCardExpanded(0)
	if !mv.ToolCards[0].Expanded {
		t.Error("card should be expanded after toggle")
	}

	mv.ToggleToolCardExpanded(0)
	if mv.ToolCards[0].Expanded {
		t.Error("card should be collapsed after second toggle")
	}
}

func TestToolCard_Duration(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	card := ToolCardData{
		ToolName: "exec_python",
		Stage:    "executing",
	}
	mv.AddToolCard(card)
	mv.UpdateToolCard("exec_python", true, "done", "3.5s")

	view := mv.View()
	if !strings.Contains(view, "3.5s") {
		t.Error("view should show duration")
	}
}

func TestToolCard_GroupWithMixedResults(t *testing.T) {
	mv := NewMessageViewport()
	mv.SetSize(80, 24)

	mv.AddToolCard(ToolCardData{ToolName: "tool_a", Stage: "executing"})
	mv.AddToolCard(ToolCardData{ToolName: "tool_b", Stage: "executing"})
	mv.UpdateToolCard("tool_a", true, "ok", "1s")
	mv.AddToolCard(ToolCardData{ToolName: "tool_c", Stage: "executing"})

	if mv.ToolCards[0].GroupID == "" {
		t.Error("tool_a should have GroupID")
	}
	if mv.ToolCards[2].GroupID == mv.ToolCards[0].GroupID {
		t.Error("tool_c should NOT be in the same group as tool_a after tool_a completed")
	}
}
