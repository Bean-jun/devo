package tools

import (
	"bufio"
	"bytes"
	"context"
	"devo/internal/pkg/process"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

type ExecPythonTool struct {
	pythonBin string
	bgManager *BackgroundProcessManager
}

var pythonSearchOrder = []string{"python3", "python", "python3.12", "python3.11", "python3.10"}

// pythonBlacklistPatterns 是基于源码文本的最后一道防线，不是主要安全边界。
// 真正的隔离应在进程/系统层面完成（容器化、namespace、资源限制、网络限制等）。
// 这些正则很容易被字符串拼接、shell=True、eval/exec、base64 等方式绕过，
// 不要假设通过了黑名单检查的代码就是"安全"的。
var pythonBlacklistPatterns = []*regexp.Regexp{
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']rm["']\s*,\s*["']-rf["']\s*,\s*["']/["']`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']rm["']\s*,\s*["']-rf["']\s*,\s*["']\$HOME`),
	regexp.MustCompile(`os\.system\s*\(\s*["'].*rm\s+-rf\s+/`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']mkfs`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']dd\s+if=`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["'](?:curl|wget)["'].*\|.*sh`),
	regexp.MustCompile(`os\.system\s*\(\s*["'].*:\s*\(\s*\)\s*\{`),
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*\[["']chmod["'].*777\s+/`),
	regexp.MustCompile(`(?i)subprocess\.(?:run|call|Popen)\s*\(\s*\[["'](?:format|diskpart)["']`),
	regexp.MustCompile(`(?i)subprocess\.(?:run|call|Popen)\s*\(\s*\[["'](?:del|rd|Remove-Item)["']`),
	// shell=True 字符串形式的命令注入，原黑名单只覆盖了 list 形式的调用
	regexp.MustCompile(`subprocess\.(?:run|call|Popen)\s*\(\s*["'].*rm\s+-rf\s+/`),
	regexp.MustCompile(`os\.system\s*\(\s*["'].*mkfs`),
}

// pipeDrainGracePeriod 是 sync 模式超时后强制关闭管道前的等待时间。
const pipeDrainGracePeriod = 3 * time.Second

// backgroundPopenPattern matches any use of subprocess.Popen (or `from subprocess
// import Popen` then bare `Popen(`). In background mode this is almost always a
// spawn-and-exit variant - the agent starts a child, reads a few lines of output
// to confirm readiness, then lets Python exit, leaving the actual server running
// as an orphaned grandchild that BackgroundProcessManager cannot stop. The correct
// primitive for background mode is `subprocess.run([...])` which blocks until the
// child exits and lets the runtime capture the PID automatically.
var backgroundPopenPattern = regexp.MustCompile(`\b(?:subprocess\.)?Popen\s*\(`)

func NewExecPythonTool(bgManager *BackgroundProcessManager) *ExecPythonTool {
	return &ExecPythonTool{
		pythonBin: detectPython(),
		bgManager: bgManager,
	}
}

func detectPython() string {
	for _, name := range pythonSearchOrder {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		// Verify the binary actually works. On Windows, Microsoft installs a
		// python3.exe stub under WindowsApps that exits with code 49 and
		// prints nothing - it just redirects users to the Store. Treat any
		// non-zero exit (or timeout) as "not really python" and keep scanning.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		cmd := exec.CommandContext(ctx, path, "--version")
		err = cmd.Run()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			continue
		}
		if err != nil {
			continue
		}
		return path
	}
	return ""
}

func (t *ExecPythonTool) Name() string {
	return "exec_python"
}

func (t *ExecPythonTool) Description() string {
	return `Execute Python code. This is the ONLY runtime tool - use it for shell commands,
builds, tests, data processing, and starting services.

Two modes:

sync (default): Python runs to completion. Use for finite tasks.
  Call shell commands via subprocess.run with explicit decoding:
    subprocess.run(["go", "build", "./..."],
        capture_output=True, text=True, encoding="utf-8", errors="replace")
  This Python runs in UTF-8 mode, but the external program you call may emit bytes in the
  local codepage (e.g. a Windows exe) - encoding="utf-8" + errors="replace" makes decoding
  failures visible instead of raising UnicodeDecodeError or producing mojibake.
  If you pass env=, base it on os.environ.copy() so you don't strip inherited settings.
  Any subprocess you spawn must either finish before this script exits, or be started with
  start_new_session=True and its stdout/stderr redirected to DEVNULL or a file - NOT left
  inheriting this process's pipes, or the tool call hangs until timeout.

background: Python itself IS the long-running process. The tool returns immediately with the
  PID; stdout/stderr stream to the frontend in real time (visible in the BG panel). The
  process is a direct child of devo - killed automatically on devo exit.
  Write code that blocks directly:
    subprocess.run(["npm", "run", "dev"])        # Python blocks on the dev server
    uvicorn.run(app, host="0.0.0.0", port=8000) # Python runs the server itself
  CORRECT pattern (the only accepted one): call subprocess.run([...]) and let it block.
  The runtime captures the PID automatically and streams the child's stdout/stderr to the
  frontend for you. If you need the output persisted to a file as well, pass the file
  handle explicitly - this is allowed and is the supported way to log a background server:
    with open("server.log", "w") as log:
        subprocess.run(["npm", "run", "dev"], stdout=log, stderr=subprocess.STDOUT)
  Do NOT use subprocess.Popen in background mode - it is rejected by PreCheck and would
  leave the server as an orphaned grandchild that stop_background_process cannot reach.
  Python MUST block; if it exits, the background process is unregistered and any
  grandchild is orphaned and unstoppable via stop_background_process.
  To stop later: use stop_background_process with the returned PID, or list_background_processes
  to discover active PIDs. To verify readiness: poll an HTTP endpoint in a SEPARATE sync
  exec_python call - do not try to capture startup output from within the background call.

Cross-platform note: on Windows, commands that resolve to .cmd/.bat scripts (npm, npx,
  yarn, gradlew, activate) cannot be run directly via subprocess list args - wrap them
  with cmd /c, e.g. subprocess.run(["cmd", "/c", "npm", "run", "dev"]). Real .exe
  binaries (python, node, git, go) can be called directly.

Never use os.system(). Always use subprocess with list arguments.

Security: Python code is pre-checked for dangerous patterns (best-effort, not a sandbox).`
}

func (t *ExecPythonTool) RiskLevel() RiskLevel {
	return RiskLevelHigh
}

func (t *ExecPythonTool) ParamsSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"code": map[string]interface{}{
				"type":        "string",
				"description": "Python 代码片段，通过 python -c 执行",
			},
			"mode": map[string]interface{}{
				"type":        "string",
				"description": "执行模式：sync（默认，等待任务完成）| background（启动长进程，立即返回 PID，输出实时流给前端）",
				"enum":        []string{"sync", "background"},
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "执行超时时间（秒），仅 sync 模式生效，默认 30",
			},
		},
		"required": []string{"code"},
	}
}

func (t *ExecPythonTool) OperationType(workingDir string, params map[string]interface{}) string {
	return "exec_python"
}

func (t *ExecPythonTool) GetCommandContext(workingDir string, params map[string]interface{}) map[string]any {
	mode := "sync"
	if m, ok := params["mode"].(string); ok && m == "background" {
		mode = "background"
	}

	timeoutSeconds := 30
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	return map[string]any{
		"working_directory": workingDir,
		"invocation":        "python -c <code>",
		"mode":              mode,
		"timeout_seconds":   timeoutSeconds,
	}
}

func (t *ExecPythonTool) PreCheck(params map[string]interface{}) error {
	code, ok := params["code"].(string)
	if !ok || code == "" {
		return fmt.Errorf("missing required parameter: code")
	}

	if isPythonCodeBlacklisted(code) {
		return fmt.Errorf("code rejected by security blacklist")
	}

	// In background mode, Python itself must block on the long-running process.
	// Any use of subprocess.Popen tends to be a spawn-and-exit variant
	// (Popen + readline + exit, Popen + sleep + exit, Popen + print PID + exit)
	// that leaves the actual server as an orphaned grandchild the
	// BackgroundProcessManager cannot stop. The correct primitive is
	// subprocess.run, which blocks until the child exits.
	mode, _ := params["mode"].(string)
	if mode == "background" && backgroundPopenPattern.MatchString(code) {
		return fmt.Errorf("background mode rejected: code uses subprocess.Popen. " +
			"In background mode Python itself must block - use `subprocess.run([...])` " +
			"instead. The runtime captures the PID automatically and streams output to " +
			"the frontend; using Popen + reading output + exiting leaves the server " +
			"orphaned and unstoppable via stop_background_process")
	}

	return nil
}

func (t *ExecPythonTool) Execute(ctx context.Context, workingDir string, params map[string]interface{}, w StreamWriter) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	code, ok := params["code"].(string)
	if !ok || code == "" {
		return fmt.Errorf("missing required parameter: code")
	}

	code = strings.TrimSpace(code)

	mode := "sync"
	if m, ok := params["mode"].(string); ok && m == "background" {
		mode = "background"
	}

	timeoutSeconds := 30
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	if t.pythonBin == "" {
		return fmt.Errorf("python not found: none of %v are available in PATH", pythonSearchOrder)
	}

	// sync 模式需要一个可超时的 context 来终止长跑的 python；background 模式
	// 不能用 execCtx 绑定 cmd，否则 Execute 返回时的 defer cancel() 会立刻把
	// python 进程杀掉。background 模式的进程生命周期由 bgManager.Stop / devo
	// 退出负责，不跟随 execCtx。
	var execCtx context.Context
	var cancel context.CancelFunc
	if mode == "sync" {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
	}

	var cmd *exec.Cmd
	if mode == "sync" {
		cmd = exec.CommandContext(execCtx, t.pythonBin, "-u", "-c", code)
	} else {
		cmd = exec.Command(t.pythonBin, "-u", "-c", code)
	}
	cmd.Dir = workingDir
	cmd.SysProcAttr = getPythonSysProcAttr()

	// 强制 Python 以 UTF-8 编解码 stdout/stderr。
	cmd.Env = append(os.Environ(),
		"PYTHONIOENCODING=utf-8",
		"PYTHONUTF8=1",
		"PYTHONUNBUFFERED=1",
	)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start python: %w", err)
	}

	pythonPID := cmd.Process.Pid

	if mode == "background" {
		return t.executeBackground(ctx, code, w, cmd, pythonPID, stdoutPipe, stderrPipe)
	}

	return t.executeSync(execCtx, w, cmd, pythonPID, stdoutPipe, stderrPipe, timeoutSeconds)
}

// executeSync runs python to completion and streams output via w. Behavior
// preserved from the original implementation; only the dead __DEVO_CHILD_PID__
// marker is removed.
func (t *ExecPythonTool) executeSync(execCtx context.Context, w StreamWriter, cmd *exec.Cmd, pythonPID int, stdoutPipe, stderrPipe io.ReadCloser, timeoutSeconds int) error {
	var stdoutBuf, stderrBuf bytes.Buffer
	var writeMu sync.Mutex

	var scanErrMu sync.Mutex
	var scanErrs []error

	readPipe := func(pipe io.ReadCloser, buf *bytes.Buffer, prefix string) {
		scanner := bufio.NewScanner(pipe)
		scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line + "\n")
			writeMu.Lock()
			w.WriteChunk(prefix + line + "\n")
			writeMu.Unlock()
		}
		if err := scanner.Err(); err != nil {
			scanErrMu.Lock()
			scanErrs = append(scanErrs, fmt.Errorf("%sscan error: %w", prefix, err))
			scanErrMu.Unlock()
		}
	}

	readDone := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			readPipe(stdoutPipe, &stdoutBuf, "")
		}()
		go func() {
			defer wg.Done()
			readPipe(stderrPipe, &stderrBuf, "[stderr] ")
		}()
		wg.Wait()
		close(readDone)
	}()

	// 关键修复：不要无条件依赖管道 EOF 来解除阻塞。
	// 如果用户的 Python 代码启动了未 detach 的子进程（继承了 stdout/stderr fd），
	// 即使主进程被 kill，管道写端仍可能保持打开，导致 Scan() 永远读不到 EOF，
	// 从而让整个 Execute 调用无限期挂起，timeoutSeconds 形同虚设。
	select {
	case <-readDone:
		// 正常情况：所有输出已读完（隐含子进程已退出或已关闭 fd）。
	case <-execCtx.Done():
		process.KillProcessGroup(pythonPID)
		select {
		case <-readDone:
		case <-time.After(pipeDrainGracePeriod):
			stdoutPipe.Close()
			stderrPipe.Close()
			<-readDone
		}
	}

	err := cmd.Wait()

	timedOut := false
	cancelled := false
	exitCode := 0

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			timedOut = true
			exitCode = -1
		} else if execCtx.Err() == context.Canceled {
			cancelled = true
			exitCode = -1
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return fmt.Errorf("python execution failed: %w", err)
		}
	}

	stdout := decodePythonOutput(stdoutBuf.Bytes())
	stderr := decodePythonOutput(stderrBuf.Bytes())

	if len(scanErrs) > 0 {
		var sb strings.Builder
		for _, e := range scanErrs {
			sb.WriteString(e.Error())
			sb.WriteString("; ")
		}
		stderr += fmt.Sprintf("\n[warning] output may be truncated: %s", sb.String())
	}

	if timedOut {
		output := fmt.Sprintf("Python execution timed out after %d seconds.\nExit code: %d\nStdout:\n%s", timeoutSeconds, exitCode, stdout)
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		w.WriteDone(false, output)
		return nil
	}

	if cancelled {
		output := fmt.Sprintf("Python execution cancelled.\nExit code: %d\nStdout:\n%s", exitCode, stdout)
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		w.WriteDone(false, output)
		return nil
	}

	output := fmt.Sprintf("Exit code: %d\nStdout:\n%s", exitCode, stdout)
	if stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", stderr)
	}

	w.WriteDone(exitCode == 0, output)
	return nil
}

// executeBackground registers the python process with the BackgroundProcessManager
// and returns immediately. The manager owns the pipes from here on - it starts a
// goroutine that streams stdout/stderr to the frontend via the manager's
// OutputForwarder (set by the agent loop), and unregisters the process when
// both pipes reach EOF (python exited) or Stop is called.
//
// cmd.Wait is invoked in a separate goroutine because we cannot return from
// Execute without releasing the StreamWriter, but we still need Wait to reap
// the process and free OS resources. The goroutine is owned by the manager's
// process record and exits after Wait returns.
func (t *ExecPythonTool) executeBackground(ctx context.Context, code string, w StreamWriter, cmd *exec.Cmd, pythonPID int, stdoutPipe, stderrPipe io.ReadCloser) error {
	if t.bgManager == nil {
		// No manager - close pipes and kill python to avoid leaking a process.
		stdoutPipe.Close()
		stderrPipe.Close()
		process.KillProcessGroup(pythonPID)
		return fmt.Errorf("background process manager not configured")
	}

	sessionID := SessionIDFromContext(ctx)
	cmdPreview := code
	if len(cmdPreview) > 200 {
		cmdPreview = cmdPreview[:200] + "..."
	}

	// Reap the python process when it exits. We can't call cmd.Wait in Execute
	// (would block until python exits, defeating the point of background mode),
	// so the manager kicks off a goroutine that calls Wait after registering.
	t.bgManager.Register(pythonPID, cmdPreview, sessionID, stdoutPipe, stderrPipe)
	go func() {
		_ = cmd.Wait()
		// Python exited. The pipe reader goroutine inside the manager will see
		// EOF and unregister. Nothing else to do here.
	}()

	w.WriteChunk(fmt.Sprintf("background process started, PID=%d\n", pythonPID))
	w.WriteDone(true, fmt.Sprintf("background process started, PID=%d", pythonPID))
	return nil
}

func isPythonCodeBlacklisted(code string) bool {
	for _, pattern := range pythonBlacklistPatterns {
		if pattern.MatchString(code) {
			return true
		}
	}
	return false
}

// decodePythonOutput 现在主要作为兜底：由于 Execute 中已经通过
// PYTHONIOENCODING/PYTHONUTF8 强制子进程使用 UTF-8，正常情况下这里
// 应该总是命中 utf8.Valid 分支。仍保留 GBK 兜底以应对某些历史遗留
// 场景（例如子进程内部再调用了忽略环境变量、自行按本地编码打印的
// 第三方可执行文件）。
func decodePythonOutput(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	if utf8.Valid(data) {
		return string(data)
	}

	if runtime.GOOS == "windows" {
		reader := transform.NewReader(bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder())
		decoded, err := io.ReadAll(reader)
		if err == nil {
			result := string(decoded)
			if utf8.ValidString(result) {
				return result
			}
		}
	}

	return strings.ToValidUTF8(string(data), "�")
}
