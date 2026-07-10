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
	"strconv"
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

// 管道读取完成后，若在强制关闭 fd 之后仍未能在此时间内退出（极端情况），
// 放弃等待，直接进入结果处理，避免 Execute 无限期阻塞。
const pipeDrainGracePeriod = 3 * time.Second

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
		// 验证能否真正执行
		cmd := exec.Command(path, "-c", "print('ok')")
		if err := cmd.Run(); err == nil {
			return path // 返回绝对路径，而不是 name
		}
	}
	return ""
}

func (t *ExecPythonTool) Name() string {
	return "exec_python"
}

func (t *ExecPythonTool) Description() string {
	return `Execute Python code and return the output. This is the ONLY runtime tool — use it for ALL tasks.

In sync mode (default): Python runs to completion, Go waits for exit code and output.
  Use subprocess.run() to call shell commands, ALWAYS with explicit text/encoding/errors:
    subprocess.run(
        ["go", "build", "./..."],
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",   # never let a bad byte crash the whole call
        cwd=None,           # set explicitly if you need a directory other than the CWD you're already in
    )
  Do not rely on ambient environment variables for decoding: this Python process already runs
  in UTF-8 mode, but that does NOT guarantee the external program you're calling emits UTF-8
  bytes (e.g. a Windows exe printing in the system's local codepage). Passing encoding="utf-8",
  errors="replace" explicitly makes decoding failures visible/safe instead of raising
  UnicodeDecodeError or silently producing mojibake.
  If you need to pass env=..., base it on os.environ.copy() (not a fresh dict) so you don't
  accidentally strip inherited settings from nested subprocess calls.

In background mode: Python starts a long-running process, prints __DEVO_BG_PID__=<pid>, then exits.
  Use subprocess.Popen with start_new_session=True: p = subprocess.Popen(["npm", "run", "dev"], start_new_session=True)
  Python MUST print the PID marker and exit. Do NOT use subprocess.run for background processes.

IMPORTANT: any subprocess you spawn from within Python (sync mode) should either finish before
your script exits, or be started with start_new_session=True and its stdout/stderr redirected
(e.g. to subprocess.DEVNULL or a file) — NOT left inheriting this process's stdout/stderr.
Otherwise the tool call may hang until the timeout is reached, because the parent process's
output pipes will not reach EOF while a child still holds them open.

Output is always treated as UTF-8. Security: Python code is pre-checked for dangerous patterns
(best-effort only, not a substitute for sandboxing).`
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
				"description": "执行模式：sync（默认，等待任务完成）| background（启动后台进程后立即返回）",
				"enum":        []string{"sync", "background"},
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "执行超时时间（秒），sync 默认 30，background 默认 10",
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
	if mode == "background" {
		timeoutSeconds = 10
	}
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
	if mode == "background" {
		timeoutSeconds = 10
	}
	if ts, ok := params["timeout_seconds"].(float64); ok && ts > 0 {
		timeoutSeconds = int(ts)
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	if t.pythonBin == "" {
		return fmt.Errorf("python not found: none of %v are available in PATH", pythonSearchOrder)
	}

	cmd := exec.CommandContext(execCtx, t.pythonBin, "-u", "-c", code)
	cmd.Dir = workingDir
	cmd.SysProcAttr = getPythonSysProcAttr()

	// 强制 Python 以 UTF-8 编解码 stdout/stderr，而不是事后猜测编码。
	// 这样即便在 LANG=C / 非 UTF-8 locale 的环境下也不会产生乱码或
	// UnicodeEncodeError。decodePythonOutput 中的猜测逻辑仅作为极端情况兜底。
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

	var stdoutBuf, stderrBuf bytes.Buffer
	var writeMu sync.Mutex // 保护对 StreamWriter 的并发写入（stdout/stderr 两个 goroutine 共用）

	var scanErrMu sync.Mutex
	var scanErrs []error

	readPipe := func(pipe io.ReadCloser, buf *bytes.Buffer, prefix string) {
		scanner := bufio.NewScanner(pipe)
		// 默认 64KB 单行上限太容易截断长输出（如单行大 JSON），扩大到 1MB 起步，最高 16MB。
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
	// 这里用 execCtx.Done() 做兜底：超时/取消发生时主动杀掉整个进程组，
	// 并且只再等待一小段宽限期，宽限期结束后强制关闭管道来解除 Scan() 阻塞。
	select {
	case <-readDone:
		// 正常情况：所有输出已读完（隐含子进程已退出或已关闭 fd）。
	case <-execCtx.Done():
		process.KillProcessGroup(pythonPID)
		select {
		case <-readDone:
		case <-time.After(pipeDrainGracePeriod):
			// 极端情况：仍有孙子进程持有 fd。强制关闭管道解除 Scan() 阻塞，
			// 代价是可能丢失少量尚未读取的输出。
			stdoutPipe.Close()
			stderrPipe.Close()
			<-readDone
		}
	}

	err = cmd.Wait()

	timedOut := false
	cancelled := false
	exitCode := 0

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			timedOut = true
			exitCode = -1
			process.KillProcessGroup(pythonPID)
		} else if ctx.Err() != nil {
			cancelled = true
			exitCode = -1
			process.KillProcessGroup(pythonPID)
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

	if mode == "background" {
		return t.handleBackgroundResult(ctx, code, w, pythonPID, stdout, stderr, timedOut, cancelled)
	}

	return t.handleSyncResult(w, pythonPID, exitCode, stdout, stderr, timedOut, cancelled, timeoutSeconds)
}

func (t *ExecPythonTool) handleSyncResult(w StreamWriter, pythonPID, exitCode int, stdout, stderr string, timedOut, cancelled bool, timeoutSeconds int) error {
	pidTag := fmt.Sprintf("\n__DEVO_CHILD_PID__=%d", pythonPID)

	if timedOut {
		output := fmt.Sprintf("Python execution timed out after %d seconds.\nExit code: %d\nStdout:\n%s", timeoutSeconds, exitCode, stdout)
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	if cancelled {
		output := fmt.Sprintf("Python execution cancelled.\nExit code: %d\nStdout:\n%s", exitCode, stdout)
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	output := fmt.Sprintf("Exit code: %d\nStdout:\n%s", exitCode, stdout)
	if stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", stderr)
	}
	output += pidTag

	w.WriteDone(exitCode == 0, output)
	return nil
}

func (t *ExecPythonTool) handleBackgroundResult(ctx context.Context, code string, w StreamWriter, pythonPID int, stdout, stderr string, timedOut, cancelled bool) error {
	pidTag := fmt.Sprintf("\n__DEVO_CHILD_PID__=%d", pythonPID)

	if timedOut {
		output := "Python background startup timed out.\n"
		bgPID := parseBGPID(stdout)
		if bgPID > 0 {
			output += fmt.Sprintf("但已检测到后台进程 PID: %d\n", bgPID)
			output += fmt.Sprintf("__DEVO_BG_PID__=%d", bgPID)
			// 注册到管理器，会话结束时自动清理
			if t.bgManager != nil {
				sessionID := SessionIDFromContext(ctx)
				cmd := code
				if len(cmd) > 200 {
					cmd = cmd[:200] + "..."
				}
				if err := t.bgManager.Register(bgPID, cmd, sessionID, ""); err != nil {
					stderr += fmt.Sprintf("\n[warning] failed to register background process: %v", err)
				}
			}
		} else {
			output += "未检测到 __DEVO_BG_PID__ 标记，后台进程可能未成功启动。\n"
			output += "请确保 Python 代码中打印了 __DEVO_BG_PID__=<pid> 后再退出。"
		}
		output += pidTag
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		// 超时但检测到 PID 标记时算成功，因为进程实际上已经启动了
		w.WriteDone(bgPID > 0, output)
		return nil
	}

	if cancelled {
		output := "Python background startup cancelled.\n"
		bgPID := parseBGPID(stdout)
		if bgPID > 0 {
			output += fmt.Sprintf("但后台进程已启动 (PID: %d)\n", bgPID)
			output += fmt.Sprintf("__DEVO_BG_PID__=%d", bgPID)
			if t.bgManager != nil {
				sessionID := SessionIDFromContext(ctx)
				cmd := code
				if len(cmd) > 200 {
					cmd = cmd[:200] + "..."
				}
				if err := t.bgManager.Register(bgPID, cmd, sessionID, ""); err != nil {
					stderr += fmt.Sprintf("\n[warning] failed to register background process: %v", err)
				}
			}
		}
		output += pidTag
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		w.WriteDone(bgPID > 0, output)
		return nil
	}

	bgPID := parseBGPID(stdout)
	if bgPID == 0 {
		output := "后台进程启动失败：Python 进程已退出，但未检测到 __DEVO_BG_PID__ 标记。\n"
		output += "请确保 Python 代码中打印了 __DEVO_BG_PID__=<pid> 后再退出。\n"
		if stdout != "" {
			output += fmt.Sprintf("\nStdout:\n%s", stdout)
		}
		if stderr != "" {
			output += fmt.Sprintf("\nStderr:\n%s", stderr)
		}
		output += pidTag
		w.WriteDone(false, output)
		return nil
	}

	// 成功启动，注册到后台进程管理器
	if t.bgManager != nil {
		sessionID := SessionIDFromContext(ctx)
		cmd := code
		if len(cmd) > 200 {
			cmd = cmd[:200] + "..."
		}
		if err := t.bgManager.Register(bgPID, cmd, sessionID, ""); err != nil {
			stderr += fmt.Sprintf("\n[warning] failed to register background process: %v", err)
		}
	}

	output := fmt.Sprintf("后台进程已启动 (PID: %d)\n", bgPID)
	if stdout != "" {
		output += fmt.Sprintf("启动输出:\n%s", stdout)
	}
	if stderr != "" {
		output += fmt.Sprintf("\nStderr:\n%s", stderr)
	}
	output += fmt.Sprintf("\n__DEVO_BG_PID__=%d", bgPID)
	output += pidTag

	w.WriteDone(true, output)
	return nil
}

func parseBGPID(content string) int {
	marker := "__DEVO_BG_PID__="
	idx := findLastIndex(content, marker)
	if idx < 0 {
		return 0
	}

	start := idx + len(marker)
	end := start
	for end < len(content) && content[end] >= '0' && content[end] <= '9' {
		end++
	}

	if end == start {
		return 0
	}

	pid, err := strconv.Atoi(content[start:end])
	if err != nil {
		return 0
	}
	return pid
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

	return strings.ToValidUTF8(string(data), "\uFFFD")
}

func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
