package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"devo/internal/interfaces/tui/types"
)

func TestEscape_WhenToolExecuting_Pauses(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateToolExecuting

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("expected pause command, got nil")
	}
}

func TestEscape_WhenPaused_Cancels(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "paused",
	}
	app.activeSession = sess
	app.state = StatePaused
	app.statusBar.SessionState = "paused"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("expected cancel command when paused, got nil")
	}
}

func TestEscape_WhenPausedViaStatusBar_Cancels(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "paused",
	}
	app.activeSession = sess
	app.statusBar.SessionState = "paused"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("expected cancel command when statusBar shows paused, got nil")
	}
}

func TestEscape_WhenThinking_Cancels(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateThinking

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("expected cancel command when thinking, got nil")
	}
}

func TestEscape_WhenProcessing_Cancels(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateProcessing

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("expected cancel command when processing, got nil")
	}
}

func TestEscape_WhenAwaitingApproval_Rejects(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateAwaitingApproval
	app.approvalModal.Visible = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("expected approval rejection command, got nil")
	}
}

func TestEscape_WhenIdle_TogglesFocus(t *testing.T) {
	app := newTestApp()
	app.state = StateReady
	app.chatView.Focus()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("expected nil command when idle (just toggle focus)")
	}

	if app.chatView.InputArea.Focused() {
		t.Error("expected input area to be blurred after Esc in idle state")
	}

	cmd = app.handleKeyMsg(msg)
	if cmd != nil {
		t.Error("expected nil command when idle and blurred (just toggle focus)")
	}
	if !app.chatView.InputArea.Focused() {
		t.Error("expected input area to be focused after second Esc")
	}
}

func TestEscape_WhenCancelled_TogglesFocus(t *testing.T) {
	app := newTestApp()
	app.state = StateCancelled
	app.chatView.Focus()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("expected nil command when cancelled (just toggle focus)")
	}
}

func TestEscape_WhenModalOpen_ClosesModal(t *testing.T) {
	app := newTestApp()
	app.state = StateReady
	app.showCommandPalette = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("expected nil command when closing modal")
	}
	if app.showCommandPalette {
		t.Error("expected command palette to be closed")
	}
}

func TestEscape_WhenRollbackPickerOpen_ClosesPicker(t *testing.T) {
	app := newTestApp()
	app.state = StateReady
	app.showRollbackPicker = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("expected nil command when closing rollback picker")
	}
	if app.showRollbackPicker {
		t.Error("expected rollback picker to be closed")
	}
}

func TestEscape_WhenSessionPickerOpen_ClosesPicker(t *testing.T) {
	app := newTestApp()
	app.state = StateReady
	app.showSessionPicker = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("expected nil command when closing session picker")
	}
	if app.showSessionPicker {
		t.Error("expected session picker to be closed")
	}
}

func TestEscape_ModalTakesPriorityOverState(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateProcessing
	app.showCommandPalette = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("modal should take priority, expected nil command")
	}
	if app.showCommandPalette {
		t.Error("expected command palette to be closed even when processing")
	}
}

func TestEscape_SessionStatePausedStringMatch(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "paused",
	}
	app.activeSession = sess
	app.state = StateReady
	app.statusBar.SessionState = "paused"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("expected cancel command when statusBar shows paused string")
	}
}

func TestCtrlP_Removed(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateProcessing

	msg := tea.KeyMsg{Type: tea.KeyCtrlP}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("Ctrl+P should be removed and return nil")
	}
}

func TestCtrlR_Removed(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "paused",
	}
	app.activeSession = sess
	app.state = StatePaused
	app.statusBar.SessionState = "paused"

	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	cmd := app.handleKeyMsg(msg)

	if cmd != nil {
		t.Error("Ctrl+R should be removed and return nil")
	}
}

func TestCtrlC_StillWorks(t *testing.T) {
	app := newTestApp()
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "processing",
	}
	app.activeSession = sess
	app.state = StateProcessing

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("Ctrl+C should still trigger cancel command")
	}
}

func TestCtrlQ_StillWorks(t *testing.T) {
	app := newTestApp()
	app.state = StateReady

	msg := tea.KeyMsg{Type: tea.KeyCtrlQ}
	_ = app.handleKeyMsg(msg)

	if app.state != StateQuitting {
		t.Error("Ctrl+Q should set state to StateQuitting")
	}
}

func TestAltY_TogglesYOLO(t *testing.T) {
	app := newTestApp()
	app.state = StateReady

	if app.yoloMode {
		t.Error("YOLO should be off by default")
	}
	if app.statusBar.YOLOMode {
		t.Error("statusBar YOLO should be off by default")
	}

	msg := tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'y'}, Alt: true})
	_ = app.handleKeyMsg(msg)

	if !app.yoloMode {
		t.Error("YOLO should be on after Alt+Y")
	}
	if !app.statusBar.YOLOMode {
		t.Error("statusBar YOLO should be on after Alt+Y")
	}

	_ = app.handleKeyMsg(msg)

	if app.yoloMode {
		t.Error("YOLO should be off after second Alt+Y")
	}
}

func TestYOLOMode_AutoApproves(t *testing.T) {
	app := newTestApp()
	app.yoloMode = true
	app.state = StateAwaitingApproval
	app.approvalModal.Visible = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("YOLO mode should auto-approve on any key")
	}
}

func TestYOLOMode_DisplaysInStatusBar(t *testing.T) {
	app := newTestApp()
	app.yoloMode = true
	app.statusBar.YOLOMode = true
	app.statusBar.AppName = "Devo"
	app.statusBar.SessionState = "idle"
	app.statusBar.Width = 80

	view := app.statusBar.View()
	if view == "" {
		t.Error("status bar view should not be empty")
	}
}

func TestYOLOMode_Off_RequiresManualApproval(t *testing.T) {
	app := newTestApp()
	app.yoloMode = false
	app.state = StateAwaitingApproval
	app.approvalModal.Visible = true

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := app.handleKeyMsg(msg)

	if cmd == nil {
		t.Error("non-YOLO mode should return approval decision")
	}
}

func TestSlashYOLO_Toggles(t *testing.T) {
	app := newTestApp()
	app.state = StateReady

	if app.yoloMode {
		t.Error("YOLO should be off initially")
	}

	cmd := app.executeSlashCommand("/yolo")
	_ = cmd

	if !app.yoloMode {
		t.Error("/yolo should toggle YOLO on")
	}

	cmd = app.executeSlashCommand("/yolo")
	_ = cmd

	if app.yoloMode {
		t.Error("/yolo should toggle YOLO off")
	}
}

func TestSlashHelp_ShowsPanel(t *testing.T) {
	app := newTestApp()
	app.state = StateReady

	if app.showHelpPanel {
		t.Error("help panel should not be shown initially")
	}
	if app.helpPanel.Visible {
		t.Error("help panel should be hidden initially")
	}

	cmd := app.executeSlashCommand("/help")
	_ = cmd

	if !app.showHelpPanel {
		t.Error("/help should set showHelpPanel to true")
	}
	if !app.helpPanel.Visible {
		t.Error("/help should make help panel visible")
	}
}

func TestEsc_ClosesHelpPanel(t *testing.T) {
	app := newTestApp()
	app.state = StateReady
	app.showHelpPanel = true
	app.helpPanel.Show()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_ = app.handleKeyMsg(msg)

	if app.showHelpPanel {
		t.Error("Esc should close help panel")
	}
	if app.helpPanel.Visible {
		t.Error("Esc should hide help panel")
	}
}

func TestSlashTrust_NoArg(t *testing.T) {
	app := newTestApp()
	app.state = StateReady
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "idle",
	}
	app.activeSession = sess

	cmd := app.executeSlashCommand("/trust")
	_ = cmd
}

func TestSlashTrust_ValidLevel(t *testing.T) {
	app := newTestApp()
	app.state = StateReady
	sess := &types.SessionInfo{
		ID:    "test-session",
		Title: "Test",
		State: "idle",
	}
	app.activeSession = sess

	cmd := app.executeSlashCommand("/trust elevated")
	_ = cmd
}

func TestCommandPaletteHasYOLOAndTrust(t *testing.T) {
	app := newTestApp()
	app.ready = true
	app.state = StateReady

	app.commandPalette.Show()
	app.commandPalette.Query = "/"

	app.commandPalette.Filter()

	foundYOLO := false
	foundTrust := false
	for _, item := range app.commandPalette.Items {
		if item.Action == "yolo" {
			foundYOLO = true
		}
		if item.Action == "trust" {
			foundTrust = true
		}
	}

	if !foundYOLO {
		t.Error("command palette should contain /yolo")
	}
	if !foundTrust {
		t.Error("command palette should contain /trust")
	}
}

func TestIsConfigError_ConfigFileNotFound(t *testing.T) {
	app := newTestApp()
	err := fmt.Errorf("config file not found: ~/.devo/config.yaml")
	if !app.isConfigError(err) {
		t.Error("should detect config file not found error")
	}
}

func TestIsConfigError_GenericError(t *testing.T) {
	app := newTestApp()
	err := fmt.Errorf("connection refused")
	if app.isConfigError(err) {
		t.Error("should not detect connection refused as config error")
	}
}

func TestIsConfigError_NilError(t *testing.T) {
	app := newTestApp()
	if app.isConfigError(nil) {
		t.Error("nil error should not be config error")
	}
}
