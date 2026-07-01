package testutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type TestLogger interface {
	Logf(format string, args ...interface{})
	Log(args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

type MockServer struct {
	cmd    *exec.Cmd
	URL    string
	port   int
	tmpDir string
}

type ValidationResult struct {
	Passed             bool
	MissingToolCallIDs []string
	RequestCount       int
}

func findMockServerDir() string {
	candidates := []string{
		"mock_server",
		filepath.Join("..", "mock_server"),
	}
	for _, c := range candidates {
		info, err := os.Stat(filepath.Join(c, "main.go"))
		if err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func StartMockServer(t TestLogger, port int) (*MockServer, error) {
	mockDir := findMockServerDir()
	if mockDir == "" {
		return nil, fmt.Errorf("mock_server directory not found (looked for mock_server/main.go)")
	}

	ms := &MockServer{port: port}

	var err error
	ms.tmpDir, err = os.MkdirTemp("", "mock-server-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	binPath := filepath.Join(ms.tmpDir, "mock_server.exe")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Dir = mockDir
	if out, err := buildCmd.CombinedOutput(); err != nil {
		os.RemoveAll(ms.tmpDir)
		return nil, fmt.Errorf("build mock_server: %w\n%s", err, string(out))
	}

	ms.cmd = exec.Command(binPath)
	ms.cmd.Dir = ms.tmpDir
	ms.cmd.Env = append(os.Environ(),
		fmt.Sprintf("MOCK_PORT=%d", port),
		fmt.Sprintf("REQUEST_DIR=%s", filepath.Join(mockDir, "requests")),
	)
	ms.cmd.Stdout = os.Stdout
	ms.cmd.Stderr = os.Stderr

	if err := ms.cmd.Start(); err != nil {
		os.RemoveAll(ms.tmpDir)
		return nil, fmt.Errorf("start mock_server: %w", err)
	}

	ms.URL = fmt.Sprintf("http://localhost:%d", port)

	if !waitForServer(ms.URL+"/v1/tool_defs", 5*time.Second) {
		ms.Stop()
		return nil, fmt.Errorf("mock_server did not start in time at %s", ms.URL)
	}

	t.Logf("Mock server started on %s", ms.URL)
	return ms, nil
}

func (ms *MockServer) Stop() {
	if ms.cmd != nil && ms.cmd.Process != nil {
		ms.cmd.Process.Kill()
		ms.cmd.Wait()
	}
	if ms.tmpDir != "" {
		os.RemoveAll(ms.tmpDir)
	}
}

func (ms *MockServer) SetHoldDelay(delayMs int64) error {
	body, _ := json.Marshal(map[string]int64{"delay_ms": delayMs})
	resp, err := http.Post(ms.URL+"/v1/hold", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set hold delay failed: %d %s", resp.StatusCode, string(body))
	}
	return nil
}

func (ms *MockServer) ResetHold() error {
	resp, err := http.Post(ms.URL+"/v1/reset", "application/json", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (ms *MockServer) GetValidation(sessionID string) (*ValidationResult, error) {
	resp, err := http.Get(ms.URL + "/v1/validation/" + sessionID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		SessionID      string `json:"session_id"`
		LastValidation struct {
			Passed             bool     `json:"passed"`
			MissingToolCallIDs []string `json:"missing_tool_call_ids"`
			RequestCount       int      `json:"request_count"`
		} `json:"last_validation"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &ValidationResult{
		Passed:             result.LastValidation.Passed,
		MissingToolCallIDs: result.LastValidation.MissingToolCallIDs,
		RequestCount:       result.LastValidation.RequestCount,
	}, nil
}

func waitForServer(url string, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return false
		default:
			resp, err := http.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return true
				}
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
}

func PostJSON(url string, body interface{}) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return http.Post(url, "application/json", bytes.NewReader(data))
}

func PutJSON(url string, body interface{}) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func GetJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func ReadBody(resp *http.Response) string {
	body, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(body))
}
