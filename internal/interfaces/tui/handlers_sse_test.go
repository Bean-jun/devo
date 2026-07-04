package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"devo/internal/interfaces/tui/components"
	"devo/internal/interfaces/tui/messages"
	"devo/internal/interfaces/tui/types"
)

func TestHandleSSE_CancelledStatePersists(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateProcessing
	app.chatView.Processing = true

	app.handleSSEEvent(messages.SSEEvent{
		Type: "session_state_change",
		Data: map[string]interface{}{
			"old_state": "processing",
			"new_state": "cancelled",
			"reason":    "cancelled",
		},
	})

	if app.state != StateCancelled {
		t.Errorf("state = %v, want StateCancelled", app.state)
	}

	if app.chatView.Processing {
		t.Error("chatView.Processing should be false after cancelled")
	}

	if app.statusBar.SessionState != "cancelled" {
		t.Errorf("statusBar.SessionState = %q, want %q", app.statusBar.SessionState, "cancelled")
	}
}

func TestHandleSSE_CancelledStateDoesNotRevertToReady(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateProcessing

	app.handleSSEEvent(messages.SSEEvent{
		Type: "session_state_change",
		Data: map[string]interface{}{
			"old_state": "processing",
			"new_state": "cancelled",
			"reason":    "cancelled",
		},
	})

	if app.state != StateCancelled {
		t.Fatalf("expected StateCancelled, got %v", app.state)
	}

	app.handleSSEEvent(messages.SSEEvent{
		Type: "message_complete",
		Data: map[string]interface{}{
			"full_text": "some response",
		},
	})

	if app.state != StateCancelled {
		t.Errorf("after message_complete, state = %v, want StateCancelled", app.state)
	}
}

func TestHandleSSE_CancelledStateResetsOnSend(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "cancelled",
	}
	app.activeSession = sess
	app.state = StateCancelled

	app.sendMessageCmd("new message")

	if app.state != StateProcessing {
		t.Errorf("state = %v, want StateProcessing (reset from cancelled)", app.state)
	}
}

func TestHandleSSE_CancelledStateAllowsSlash(t *testing.T) {
	app := newTestApp()
	app.state = StateCancelled

	app.chatView.Focus()
	app.chatView.InputArea.SetValue("")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}

	_ = app.handleKeyMsg(msg)

	if !app.showCommandPalette {
		t.Error("expected command palette to open in cancelled state, but it did not")
	}
}

func TestHandleSSE_CancelledStateBlocksEnterWhenNotReady(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateProcessing
	app.chatView.Focus()
	app.chatView.InputArea.SetValue("test message")

	msgsBefore := len(app.msgs)

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_ = app.handleKeyMsg(msg)

	if len(app.msgs) > msgsBefore {
		t.Error("should not send message in non-ready/non-cancelled state")
	}
}

func newTestApp() *App {
	app, _ := NewAppWithURL("http://localhost:8080", "0.0.1")
	app.ready = true
	app.width = 80
	app.height = 24
	app.layout()
	return app
}

func TestHandleSSE_Streaming(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	app.handleSSEEvent(messages.SSEEvent{
		Type: "streaming",
		Data: map[string]interface{}{
			"data": "Hello ",
		},
	})

	app.handleSSEEvent(messages.SSEEvent{
		Type: "streaming",
		Data: map[string]interface{}{
			"data": "World",
		},
	})

	if !app.chatView.MessageView.StreamingActive {
		t.Error("StreamingActive should be true after streaming events")
	}

	streamingContent := app.chatView.MessageView.StreamingBuffer.String()
	if streamingContent != "Hello World" {
		t.Errorf("streaming content = %q, want %q", streamingContent, "Hello World")
	}
}

func TestHandleSSE_StreamingFinalizedByMessageComplete(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	app.handleSSEEvent(messages.SSEEvent{
		Type: "streaming",
		Data: map[string]interface{}{
			"data": "Hello from streaming",
		},
	})

	app.handleSSEEvent(messages.SSEEvent{
		Type: "message_complete",
		Data: map[string]interface{}{
			"full_text": "Hello from streaming",
		},
	})

	if app.chatView.MessageView.StreamingActive {
		t.Error("StreamingActive should be false after message_complete")
	}

	found := false
	for _, msg := range app.msgs {
		if msg.Role == "assistant" && msg.Content == "Hello from streaming" {
			found = true
			break
		}
	}
	if !found {
		t.Error("streaming content should be finalized as assistant message")
	}
}

func TestHandleSSE_MessageCompleteFallsBackToFullText(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	app.handleSSEEvent(messages.SSEEvent{
		Type: "message_complete",
		Data: map[string]interface{}{
			"full_text": "Direct response without streaming",
		},
	})

	found := false
	for _, msg := range app.msgs {
		if msg.Role == "assistant" && msg.Content == "Direct response without streaming" {
			found = true
			break
		}
	}
	if !found {
		t.Error("message_complete should add full_text as assistant message when no streaming")
	}
}

func TestHandleSSE_ToolChunk(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	card := components.ToolCardData{
		ToolName: "exec_python",
		Params:   "print('hello')",
	}
	app.chatView.MessageView.AddToolCard(card)

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_chunk",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"data":      "Line 1: Hello\n",
		},
	})

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_chunk",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"data":      "Line 2: World\n",
		},
	})

	toolCards := app.chatView.MessageView.ToolCards
	if len(toolCards) != 1 {
		t.Fatalf("ToolCards length = %d, want 1", len(toolCards))
	}
	if len(toolCards[0].OutputChunks) != 2 {
		t.Fatalf("OutputChunks length = %d, want 2", len(toolCards[0].OutputChunks))
	}
	if toolCards[0].OutputChunks[0] != "Line 1: Hello\n" {
		t.Errorf("first chunk = %q", toolCards[0].OutputChunks[0])
	}
	if toolCards[0].OutputChunks[1] != "Line 2: World\n" {
		t.Errorf("second chunk = %q", toolCards[0].OutputChunks[1])
	}
	if toolCards[0].Stage != "executing" {
		t.Errorf("stage = %q, want 'executing'", toolCards[0].Stage)
	}
}

func TestHandleSSE_ToolChunk_EmptyDataIgnored(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_chunk",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"data":      "",
		},
	})

	if len(app.chatView.MessageView.ToolCards) > 0 {
		t.Error("empty tool_chunk should not create tool cards")
	}
}

func TestHandleSSE_ToolProgress(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	card := components.ToolCardData{
		ToolName: "exec_python",
		Params:   "print('hello')",
		Stage:    "pending",
	}
	app.chatView.MessageView.AddToolCard(card)

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_progress",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"stage":     "executing",
		},
	})

	toolCards := app.chatView.MessageView.ToolCards
	if len(toolCards) != 1 {
		t.Fatalf("ToolCards length = %d, want 1", len(toolCards))
	}
	if toolCards[0].Stage != "executing" {
		t.Errorf("stage = %q, want 'executing'", toolCards[0].Stage)
	}

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_progress",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"stage":     "done",
		},
	})

	if toolCards[0].Stage != "done" {
		t.Errorf("stage = %q, want 'done'", toolCards[0].Stage)
	}
}

func TestHandleSSE_ToolProgress_EmptyDataIgnored(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	card := components.ToolCardData{
		ToolName: "exec_python",
		Stage:    "pending",
	}
	app.chatView.MessageView.AddToolCard(card)

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_progress",
		Data: map[string]interface{}{
			"tool_name": "",
			"stage":     "executing",
		},
	})

	if app.chatView.MessageView.ToolCards[0].Stage != "pending" {
		t.Error("stage should not change when tool_name is empty")
	}
}

func TestHandleSSE_StreamingWithToolInterleaved(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess

	app.handleSSEEvent(messages.SSEEvent{
		Type: "streaming",
		Data: map[string]interface{}{
			"data": "Let me run a script.\n",
		},
	})

	card := components.ToolCardData{
		ToolName: "exec_python",
		Params:   "print('hello')",
	}
	app.chatView.MessageView.AddToolCard(card)

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_progress",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"stage":     "executing",
		},
	})

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_chunk",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"data":      "hello\n",
		},
	})

	app.handleSSEEvent(messages.SSEEvent{
		Type: "tool_progress",
		Data: map[string]interface{}{
			"tool_name": "exec_python",
			"stage":     "done",
		},
	})

	app.handleSSEEvent(messages.SSEEvent{
		Type: "streaming",
		Data: map[string]interface{}{
			"data": "Script completed successfully.",
		},
	})

	app.handleSSEEvent(messages.SSEEvent{
		Type: "message_complete",
		Data: map[string]interface{}{},
	})

	if app.chatView.MessageView.StreamingActive {
		t.Error("StreamingActive should be false after message_complete")
	}

	toolCards := app.chatView.MessageView.ToolCards
	if len(toolCards) != 1 {
		t.Fatalf("ToolCards length = %d, want 1", len(toolCards))
	}
	if toolCards[0].Stage != "done" {
		t.Errorf("tool card stage = %q, want 'done'", toolCards[0].Stage)
	}
	if len(toolCards[0].OutputChunks) != 1 {
		t.Errorf("OutputChunks length = %d, want 1", len(toolCards[0].OutputChunks))
	}
}
