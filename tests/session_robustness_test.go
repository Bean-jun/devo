package tests

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"devo-tests/testutil"
)

var (
	mock   *testutil.MockServer
	devo   *testutil.DevoServer
	client *testutil.DevoClient
)

type testMLogger struct{}

func (l *testMLogger) Logf(format string, args ...interface{}) {
	fmt.Printf("[LOG] "+format+"\n", args...)
}
func (l *testMLogger) Log(args ...interface{})   { fmt.Println(args...) }
func (l *testMLogger) Fatal(args ...interface{}) { fmt.Println("FATAL:", args); os.Exit(1) }
func (l *testMLogger) Fatalf(format string, args ...interface{}) {
	fmt.Printf("FATAL: "+format+"\n", args...)
	os.Exit(1)
}
func (l *testMLogger) Errorf(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

func TestMain(m *testing.M) {
	logger := &testMLogger{}
	var err error

	mock, err = testutil.StartMockServer(logger, 9090)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start mock server: %v\n", err)
		os.Exit(1)
	}

	devo, err = testutil.StartDevo(logger, mock.URL, 9091)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start devo: %v\n", err)
		mock.Stop()
		os.Exit(1)
	}

	client = testutil.NewDevoClient(devo.URL)

	code := m.Run()

	devo.Stop()
	mock.Stop()
	os.Exit(code)
}

func workingDir() string {
	dir, _ := os.Getwd()
	return dir
}

func assertStatus(t *testing.T, resp *http.Response, expectedStatus int, context string) {
	t.Helper()
	if resp.StatusCode != expectedStatus {
		body := testutil.ReadBody(resp)
		t.Errorf("%s: expected status %d, got %d (body: %s)", context, expectedStatus, resp.StatusCode, body)
	}
}

func assertState(t *testing.T, sessionID, expectedState string, context string) {
	t.Helper()
	state, err := client.GetState(sessionID)
	if err != nil {
		t.Errorf("%s: get state: %v", context, err)
		return
	}
	if state != expectedState {
		t.Errorf("%s: expected state %q, got %q", context, expectedState, state)
	}
}

func mustCreateSession(t *testing.T) string {
	t.Helper()
	sessionID, err := client.CreateSession(workingDir())
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Logf("Created session: %s", sessionID)
	return sessionID
}

func waitForIdle(t *testing.T, sessionID string) {
	t.Helper()
	if err := client.WaitForState(sessionID, "idle", 30*time.Second); err != nil {
		t.Fatalf("wait for idle: %v", err)
	}
}

func waitForThinking(t *testing.T, sessionID string) {
	t.Helper()
	if err := client.WaitForState(sessionID, "thinking", 10*time.Second); err != nil {
		t.Fatalf("wait for thinking: %v", err)
	}
}

func waitForToolExecuting(t *testing.T, sessionID string) {
	t.Helper()
	if err := client.WaitForState(sessionID, "tool_executing", 10*time.Second); err != nil {
		t.Fatalf("wait for tool_executing: %v", err)
	}
}

func waitForStateNotIdle(t *testing.T, sessionID string) {
	t.Helper()
	if err := client.WaitForStateNot(sessionID, "idle", 10*time.Second); err != nil {
		t.Fatalf("wait for state != idle: %v", err)
	}
}

func nextRoundWorks(t *testing.T, sessionID string) {
	t.Helper()
	resp, err := client.SendMessage(sessionID, "Hello, continue the conversation")
	if err != nil {
		t.Fatalf("send follow-up message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "follow-up message should be accepted")
	waitForIdle(t, sessionID)
}

func assertMockValidationPassed(t *testing.T, sessionID string, context string) {
	t.Helper()
	vr, err := mock.GetValidation(sessionID)
	if err != nil {
		t.Logf("%s: get validation: %v", context, err)
		return
	}
	if !vr.Passed {
		t.Errorf("%s: mock_server validation FAILED (orphan tool_calls detected): missing tool_call_ids: %v",
			context, vr.MissingToolCallIDs)
	} else {
		t.Logf("%s: mock_server validation PASSED", context)
	}
}

// =============================================================================
// 场景 1：Pause（暂停）
// Pause 仅在 ToolExecuting 阶段生效，Thinking 阶段忽略
// =============================================================================

func TestP2_PauseDuringToolCalls(t *testing.T) {
	sessionID := mustCreateSession(t)

	trustResp, err := client.SetTrustLevel(sessionID, "elevated")
	if err != nil {
		t.Fatalf("set trust level: %v", err)
	}
	assertStatus(t, trustResp, http.StatusOK, "set trust level to elevated")

	mock.SetHoldDelay(3000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Run a slow command ${execute_command}$")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message with tool calls")

	waitForToolExecuting(t, sessionID)
	time.Sleep(300 * time.Millisecond)

	pauseResp, err := client.Pause(sessionID)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	assertStatus(t, pauseResp, http.StatusOK, "pause during tool calls")

	assertState(t, sessionID, "paused", "after Pause during tool calls")

	resumeResp, err := client.Resume(sessionID)
	if err != nil {
		t.Fatalf("resume (Bug #1: Pause does not set session.State=Paused): %v", err)
	}
	assertStatus(t, resumeResp, http.StatusOK, "resume after pause")

	assertState(t, sessionID, "tool_executing", "after Resume")

	waitForIdle(t, sessionID)
	nextRoundWorks(t, sessionID)
	assertMockValidationPassed(t, sessionID, "P2")
}

func TestP3_PauseDuringApprovalWaiting(t *testing.T) {
	sessionID := mustCreateSession(t)

	mock.SetHoldDelay(1000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Write a new file ${write_file}$")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message that triggers approval")

	waitForStateNotIdle(t, sessionID)
	time.Sleep(500 * time.Millisecond)

	pauseResp, err := client.Pause(sessionID)
	if err != nil {
		t.Logf("Pause during approval returned error (expected 409, state is not ToolExecuting): %v", err)
	} else {
		assertStatus(t, pauseResp, http.StatusConflict, "pause during approval should return 409")
	}

	cancelResp, err := client.Cancel(sessionID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	assertStatus(t, cancelResp, http.StatusOK, "cancel after pause attempt")

	waitForIdle(t, sessionID)
	nextRoundWorks(t, sessionID)
	assertMockValidationPassed(t, sessionID, "P3")
}

// =============================================================================
// 场景 2：Cancel（取消）
// Thinking 阶段：中止流，无需清理
// ToolExecuting/AwaitingApproval/Paused 阶段：补 tool_result + user 消息
// =============================================================================

func TestC1_CancelDuringStreaming(t *testing.T) {
	sessionID := mustCreateSession(t)

	mock.SetHoldDelay(3000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Tell me a long story about programming")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message")

	waitForThinking(t, sessionID)
	time.Sleep(200 * time.Millisecond)

	cancelResp, err := client.Cancel(sessionID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	assertStatus(t, cancelResp, http.StatusOK, "cancel during streaming")

	assertState(t, sessionID, "idle", "after Cancel during streaming")

	nextRoundWorks(t, sessionID)
}

func TestC2_CancelDuringToolCalls(t *testing.T) {
	sessionID := mustCreateSession(t)

	trustResp, err := client.SetTrustLevel(sessionID, "elevated")
	if err != nil {
		t.Fatalf("set trust level: %v", err)
	}
	assertStatus(t, trustResp, http.StatusOK, "set trust level to elevated")

	mock.SetHoldDelay(3000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Execute a command and read a file ${execute_command,read_file}$")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message with tool calls")

	waitForToolExecuting(t, sessionID)
	time.Sleep(200 * time.Millisecond)

	cancelResp, err := client.Cancel(sessionID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	assertStatus(t, cancelResp, http.StatusOK, "cancel during tool calls")

	assertState(t, sessionID, "idle", "after Cancel during tool calls")

	nextRoundWorks(t, sessionID)
	assertMockValidationPassed(t, sessionID, "C2")
}

func TestC3_CancelDuringApprovalWaiting(t *testing.T) {
	sessionID := mustCreateSession(t)

	mock.SetHoldDelay(1000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Write a new file ${write_file}$")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message that triggers approval")

	waitForStateNotIdle(t, sessionID)
	time.Sleep(500 * time.Millisecond)

	cancelResp, err := client.Cancel(sessionID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	assertStatus(t, cancelResp, http.StatusOK, "cancel during approval")

	assertState(t, sessionID, "idle", "after Cancel during approval")

	nextRoundWorks(t, sessionID)
	assertMockValidationPassed(t, sessionID, "C3")
}

// =============================================================================
// 场景 3：Resume（恢复）
// Resume 仅从 Paused 状态恢复，启动工具调度
// =============================================================================

func TestR2_ResumeAfterPauseDuringToolCalls(t *testing.T) {
	sessionID := mustCreateSession(t)

	trustResp, err := client.SetTrustLevel(sessionID, "elevated")
	if err != nil {
		t.Fatalf("set trust level: %v", err)
	}
	assertStatus(t, trustResp, http.StatusOK, "set trust level to elevated")

	mock.SetHoldDelay(3000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Execute a slow command and read a file ${execute_command,read_file}$")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message with slow tool calls")

	waitForToolExecuting(t, sessionID)
	time.Sleep(200 * time.Millisecond)

	pauseResp, err := client.Pause(sessionID)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	assertStatus(t, pauseResp, http.StatusOK, "pause")

	assertState(t, sessionID, "paused", "after Pause (R2)")

	resumeResp, err := client.Resume(sessionID)
	if err != nil {
		t.Fatalf("resume (Bug #1: Pause does not set session.State=Paused): %v", err)
	}
	assertStatus(t, resumeResp, http.StatusOK, "resume after pause")

	assertState(t, sessionID, "tool_executing", "after Resume")

	waitForIdle(t, sessionID)
	nextRoundWorks(t, sessionID)
	assertMockValidationPassed(t, sessionID, "R2")
}

// =============================================================================
// 场景 4：Crash（崩溃重启）
// 重启后根据 DB 中的状态清理消息历史
// =============================================================================

func TestCr1_CrashDuringStreaming(t *testing.T) {
	sessionID := mustCreateSession(t)

	mock.SetHoldDelay(5000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Tell me a very long story")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message")

	waitForThinking(t, sessionID)
	time.Sleep(500 * time.Millisecond)

	t.Log("Killing Devo to simulate crash...")
	devo.Kill()

	time.Sleep(1 * time.Second)

	t.Log("Restarting Devo...")
	if err := devo.Restart(t); err != nil {
		t.Fatalf("restart devo: %v", err)
	}

	client = testutil.NewDevoClient(devo.URL)

	time.Sleep(1 * time.Second)

	assertState(t, sessionID, "idle", "after crash restart during streaming")

	resp, err = client.SendMessage(sessionID, "Hello after crash")
	if err != nil {
		t.Fatalf("send follow-up after crash: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "follow-up after crash during streaming")

	waitForIdle(t, sessionID)
}

func TestCr2_CrashDuringToolCalls(t *testing.T) {
	sessionID := mustCreateSession(t)

	trustResp, err := client.SetTrustLevel(sessionID, "elevated")
	if err != nil {
		t.Fatalf("set trust level: %v", err)
	}
	assertStatus(t, trustResp, http.StatusOK, "set trust level to elevated")

	mock.SetHoldDelay(5000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Execute a command and read a file ${execute_command,read_file}$")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message with tool calls")

	waitForToolExecuting(t, sessionID)
	time.Sleep(500 * time.Millisecond)

	t.Log("Killing Devo to simulate crash...")
	devo.Kill()

	time.Sleep(1 * time.Second)

	t.Log("Restarting Devo...")
	if err := devo.Restart(t); err != nil {
		t.Fatalf("restart devo: %v", err)
	}

	client = testutil.NewDevoClient(devo.URL)

	time.Sleep(1 * time.Second)

	assertState(t, sessionID, "idle", "after crash restart during tool calls")

	resp, err = client.SendMessage(sessionID, "Hello after crash")
	if err != nil {
		t.Fatalf("send follow-up after crash: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "follow-up after crash during tool calls")

	waitForIdle(t, sessionID)
	assertMockValidationPassed(t, sessionID, "Cr2")
}

func TestCr3_CrashDuringApprovalWaiting(t *testing.T) {
	sessionID := mustCreateSession(t)

	mock.SetHoldDelay(1000)
	defer mock.ResetHold()

	resp, err := client.SendMessage(sessionID, "Write a new file ${write_file}$")
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "send message that triggers approval")

	waitForStateNotIdle(t, sessionID)
	time.Sleep(500 * time.Millisecond)

	t.Log("Killing Devo to simulate crash...")
	devo.Kill()

	time.Sleep(1 * time.Second)

	t.Log("Restarting Devo...")
	if err := devo.Restart(t); err != nil {
		t.Fatalf("restart devo: %v", err)
	}

	client = testutil.NewDevoClient(devo.URL)

	time.Sleep(1 * time.Second)

	assertState(t, sessionID, "idle", "after crash restart during approval")

	resp, err = client.SendMessage(sessionID, "Hello after crash")
	if err != nil {
		t.Fatalf("send follow-up after crash: %v", err)
	}
	assertStatus(t, resp, http.StatusAccepted, "follow-up after crash during approval")

	waitForIdle(t, sessionID)
	assertMockValidationPassed(t, sessionID, "Cr3")
}
