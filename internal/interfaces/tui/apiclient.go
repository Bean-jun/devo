package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devo/internal/interfaces/tui/types"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *APIClient) CreateSession(workingDir, title string) (*types.SessionInfo, error) {
	req := types.CreateSessionRequest{
		WorkingDirectory: workingDir,
		Title:            title,
	}
	var sess types.SessionInfo
	if err := c.doJSON("POST", "/api/v1/sessions", req, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (c *APIClient) ListSessions(limit, offset int) ([]types.SessionInfo, error) {
	url := fmt.Sprintf("/api/v1/sessions?limit=%d&offset=%d", limit, offset)
	var resp types.ListSessionsResponse
	if err := c.doJSON("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Sessions, nil
}

func (c *APIClient) GetSession(id string) (*types.SessionInfo, error) {
	var sess types.SessionInfo
	if err := c.doJSON("GET", "/api/v1/sessions/"+id, nil, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (c *APIClient) SendMessage(sessionID, content string) error {
	req := types.SendMessageRequest{Content: content}
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/messages", req, nil)
}

func (c *APIClient) GetMessages(sessionID string, limit, offset int) ([]types.Message, error) {
	url := fmt.Sprintf("/api/v1/sessions/%s/messages?limit=%d&offset=%d", sessionID, limit, offset)
	var resp types.GetMessagesResponse
	if err := c.doJSON("GET", url, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func (c *APIClient) GetFiles(sessionID string) ([]types.FileInfo, error) {
	var files []types.FileInfo
	if err := c.doJSON("GET", "/api/v1/sessions/"+sessionID+"/files", nil, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *APIClient) Approve(sessionID, approvalID string) error {
	req := types.ApproveRequest{Decision: "approve"}
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/approve/"+approvalID, req, nil)
}

func (c *APIClient) Reject(sessionID, approvalID string) error {
	req := types.ApproveRequest{Decision: "reject"}
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/approve/"+approvalID, req, nil)
}

func (c *APIClient) Pause(sessionID string) error {
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/pause", nil, nil)
}

func (c *APIClient) Resume(sessionID string) error {
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/resume", nil, nil)
}

func (c *APIClient) Cancel(sessionID string) error {
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/cancel", nil, nil)
}

func (c *APIClient) Complete(sessionID string) error {
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/complete", nil, nil)
}

func (c *APIClient) Archive(sessionID string) error {
	return c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/archive", nil, nil)
}

func (c *APIClient) SetTrustLevel(sessionID, level string) error {
	req := types.SetTrustRequest{TrustLevel: level}
	return c.doJSON("PUT", "/api/v1/sessions/"+sessionID+"/trust", req, nil)
}

func (c *APIClient) SetApprovalPolicy(sessionID, opType, level string) error {
	req := types.SetApprovalPolicyRequest{
		OperationType: opType,
		PolicyLevel:   level,
	}
	return c.doJSON("PUT", "/api/v1/sessions/"+sessionID+"/approval-policy", req, nil)
}

func (c *APIClient) UpdateConfig(sessionID string, toolCallLimit int) error {
	req := types.UpdateConfigRequest{ToolCallLimit: toolCallLimit}
	return c.doJSON("PUT", "/api/v1/sessions/"+sessionID+"/config", req, nil)
}

func (c *APIClient) Rollback(sessionID string, targetMessageID string) (*types.RollbackResult, error) {
	req := types.RollbackRequest{TargetMessageID: targetMessageID}
	var result types.RollbackResult
	if err := c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/rollback", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *APIClient) SyncArchive(sessionID string) (*types.SyncArchiveResult, error) {
	var result types.SyncArchiveResult
	if err := c.doJSON("POST", "/api/v1/sessions/"+sessionID+"/sync-archive", nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *APIClient) SSEEndpoint(sessionID string) string {
	return fmt.Sprintf("%s/api/v1/sessions/%s/events", c.baseURL, sessionID)
}

func (c *APIClient) doJSON(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	Log("[HTTP] %s %s", method, path)
	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		Log("[HTTP] %s %s -> request error: %v", method, path, err)
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		Log("[HTTP] %s %s -> do error: %v", method, path, err)
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]string
		json.NewDecoder(resp.Body).Decode(&errBody)
		msg := errBody["error"]
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		Log("[HTTP] %s %s -> %d %s", method, path, resp.StatusCode, msg)
		return fmt.Errorf("API error [%d]: %s", resp.StatusCode, msg)
	}

	Log("[HTTP] %s %s -> %d OK", method, path, resp.StatusCode)
	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			Log("[HTTP] %s %s -> decode error: %v", method, path, err)
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
