package testutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type DevoServer struct {
	cmd     *exec.Cmd
	URL     string
	mockURL string
	port    int
	tmpDir  string
}

func findDevoRoot() string {
	candidates := []string{
		"..",
		filepath.Join("..", ".."),
		".",
	}
	for _, c := range candidates {
		info, err := os.Stat(filepath.Join(c, "cmd", "devo"))
		if err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func BuildDevo() (string, error) {
	devoRoot := findDevoRoot()
	if devoRoot == "" {
		return "", fmt.Errorf("devo project root not found (go.mod not found)")
	}

	tmpDir, err := os.MkdirTemp("", "devo-build-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	binPath := filepath.Join(tmpDir, "devo.exe")
	buildCmd := exec.Command("go", "build", "-o", binPath, "./cmd/devo/")
	buildCmd.Dir = devoRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("build devo: %w\n%s", err, string(out))
	}

	return binPath, nil
}

func StartDevo(t TestLogger, mockURL string, port int) (*DevoServer, error) {
	binPath, err := BuildDevo()
	if err != nil {
		return nil, err
	}

	ds := &DevoServer{port: port, mockURL: mockURL}

	ds.tmpDir, err = os.MkdirTemp("", "devo-test-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	ds.cmd = exec.Command(binPath, fmt.Sprintf("-port=%d", port))
	ds.cmd.Dir = ds.tmpDir
	ds.cmd.Env = append(os.Environ(),
		"DEVO_LLM_BASE_URL="+mockURL+"/v1",
		"DEVO_LLM_API_KEY=sk-mock-key",
		"DEVO_LLM_MODEL=mock-model",
		"DEVO_DB_PATH="+filepath.Join(ds.tmpDir, "devo.db"),
		"DEVO_LOG_PATH="+filepath.Join(ds.tmpDir, "devo.log"),
	)
	ds.cmd.Stdout = os.Stdout
	ds.cmd.Stderr = os.Stderr

	if err := ds.cmd.Start(); err != nil {
		os.RemoveAll(ds.tmpDir)
		return nil, fmt.Errorf("start devo: %w", err)
	}

	ds.URL = fmt.Sprintf("http://localhost:%d", port)

	if !waitForDevo(ds.URL+"/api/v1/config-status", 15*time.Second) {
		ds.Stop()
		return nil, fmt.Errorf("devo did not start in time at %s", ds.URL)
	}

	t.Logf("Devo started on %s", ds.URL)
	return ds, nil
}

func (ds *DevoServer) Stop() {
	if ds.cmd != nil && ds.cmd.Process != nil {
		ds.cmd.Process.Kill()
		ds.cmd.Wait()
	}
	if ds.tmpDir != "" {
		os.RemoveAll(ds.tmpDir)
	}
}

func (ds *DevoServer) Kill() {
	if ds.cmd != nil && ds.cmd.Process != nil {
		ds.cmd.Process.Kill()
		ds.cmd.Wait()
	}
}

func (ds *DevoServer) Restart(t TestLogger) error {
	oldPID := 0
	if ds.cmd != nil && ds.cmd.Process != nil {
		oldPID = ds.cmd.Process.Pid
		ds.cmd.Process.Kill()
		ds.cmd.Wait()
	}

	binPath, err := BuildDevo()
	if err != nil {
		return err
	}

	ds.cmd = exec.Command(binPath, fmt.Sprintf("-port=%d", ds.port))
	ds.cmd.Dir = ds.tmpDir
	ds.cmd.Env = append(os.Environ(),
		"DEVO_LLM_BASE_URL="+ds.mockURL+"/v1",
		"DEVO_LLM_API_KEY=sk-mock-key",
		"DEVO_LLM_MODEL=mock-model",
		"DEVO_DB_PATH="+filepath.Join(ds.tmpDir, "devo.db"),
		"DEVO_LOG_PATH="+filepath.Join(ds.tmpDir, "devo.log"),
	)
	ds.cmd.Stdout = os.Stdout
	ds.cmd.Stderr = os.Stderr

	if err := ds.cmd.Start(); err != nil {
		return fmt.Errorf("restart devo: %w", err)
	}

	if !waitForDevo(ds.URL+"/api/v1/config-status", 15*time.Second) {
		return fmt.Errorf("devo did not restart in time")
	}

	t.Logf("Devo restarted (old PID: %d, new PID: %d)", oldPID, ds.cmd.Process.Pid)
	return nil
}

func waitForDevo(url string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return false
		default:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				io.ReadAll(resp.Body)
				return true
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func MustOK(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
}

func ContextWithTimeout(seconds int) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(seconds)*time.Second)
}
