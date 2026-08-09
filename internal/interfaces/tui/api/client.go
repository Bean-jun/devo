package api

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

type Client struct {
	BaseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	url := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

func (c *Client) get(path string, result interface{}) error {
	return c.do("GET", path, nil, result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	return c.do("POST", path, body, result)
}

func (c *Client) put(path string, body interface{}, result interface{}) error {
	return c.do("PUT", path, body, result)
}

func (c *Client) del(path string) error {
	return c.do("DELETE", path, nil, nil)
}

// ─── Sessions ───

func (c *Client) CreateSession(workingDir, title string) (*types.SessionInfo, error) {
	req := types.CreateSessionRequest{
		WorkingDirectory: workingDir,
		Title:            title,
	}
	var session types.SessionInfo
	if err := c.post("/api/v1/sessions", req, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (c *Client) ListSessions() ([]types.SessionInfo, error) {
	var resp types.ListSessionsResponse
	if err := c.get("/api/v1/sessions", &resp); err != nil {
		return nil, err
	}
	return resp.Sessions, nil
}

func (c *Client) GetSession(sessionID string) (*types.SessionInfo, error) {
	var session types.SessionInfo
	if err := c.get("/api/v1/sessions/"+sessionID, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (c *Client) RenameSession(sessionID, title string) error {
	return c.put("/api/v1/sessions/"+sessionID, map[string]string{"title": title}, nil)
}

func (c *Client) DeleteSession(sessionID string) error {
	return c.del("/api/v1/sessions/" + sessionID)
}

func (c *Client) ArchiveSession(sessionID string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/archive", nil, nil)
}

func (c *Client) ExportSession(sessionID string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/export", nil, nil)
}

// ─── Messages ───

func (c *Client) SendMessage(sessionID string, req types.SendMessageRequest) (*types.Message, error) {
	var msg types.Message
	if err := c.post("/api/v1/sessions/"+sessionID+"/messages", req, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (c *Client) GetMessages(sessionID string) ([]types.Message, error) {
	var resp types.GetMessagesResponse
	if err := c.get("/api/v1/sessions/"+sessionID+"/messages", &resp); err != nil {
		return nil, err
	}
	return resp.Messages, nil
}

func (c *Client) Rollback(sessionID string, req types.RollbackRequest) (*types.GetMessagesResponse, error) {
	var resp types.GetMessagesResponse
	if err := c.post("/api/v1/sessions/"+sessionID+"/rollback", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ─── Files ───

func (c *Client) GetFiles(sessionID string, dirPath string) ([]types.FileInfo, error) {
	path := "/api/v1/files"
	if dirPath != "" {
		path += "?path=" + dirPath
	}
	var resp struct {
		Type    string           `json:"type"`
		Entries []types.FileInfo `json:"entries"`
	}
	if err := c.get(path, &resp); err != nil {
		return nil, err
	}
	return resp.Entries, nil
}

// ─── Skills ───

type SkillInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	Source      string `json:"source"`
}

type SkillsResponse struct {
	Skills []SkillInfo `json:"skills"`
}

func (c *Client) GetSkills() ([]SkillInfo, error) {
	var resp struct {
		Skills []SkillInfo `json:"skills"`
	}
	if err := c.get("/api/v1/skills", &resp); err != nil {
		return nil, err
	}
	return resp.Skills, nil
}

func (c *Client) SetSessionSkills(sessionID string, skillNames []string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/skills", map[string]interface{}{
		"skills": skillNames,
	}, nil)
}

func (c *Client) RemoveSessionSkill(sessionID string, skillName string) error {
	return c.del("/api/v1/sessions/" + sessionID + "/skills/" + skillName)
}

// ─── MCP ───

type MCPServerInfo struct {
	ServerID  string `json:"server_id"`
	Endpoint  string `json:"endpoint"`
	Transport string `json:"transport"`
	Status    string `json:"status"`
	ToolCount int    `json:"tool_count"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

type MCPServersResponse struct {
	Servers []MCPServerInfo `json:"servers"`
}

func (c *Client) GetMCPServers() ([]MCPServerInfo, error) {
	var resp MCPServersResponse
	if err := c.get("/api/v1/mcp/servers", &resp); err != nil {
		return nil, err
	}
	return resp.Servers, nil
}

func (c *Client) ToggleMcpServer(serverID string) error {
	return c.post("/api/v1/mcp/servers/"+serverID+"/toggle", nil, nil)
}

// ─── Memory ───

type MemoryItem struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Content string `json:"content"`
	Scope   string `json:"scope"`
}

type MemoryResponse struct {
	Memories []MemoryItem `json:"memories"`
}

func (c *Client) GetMemories(sessionID string) ([]MemoryItem, error) {
	var resp MemoryResponse
	if err := c.get("/api/v1/sessions/"+sessionID+"/memory", &resp); err != nil {
		return nil, err
	}
	return resp.Memories, nil
}

func (c *Client) UpsertMemory(sessionID string, key, content string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/memory", map[string]string{
		"key":     key,
		"content": content,
	}, nil)
}

func (c *Client) DeleteMemory(sessionID string, memoryID string) error {
	return c.del("/api/v1/sessions/" + sessionID + "/memory/" + memoryID)
}

// ─── Workspace ───

type WorkspaceInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

type WorkspacesResponse struct {
	Workspaces []WorkspaceInfo `json:"workspaces"`
}

func (c *Client) GetWorkspaces() ([]WorkspaceInfo, error) {
	var resp WorkspacesResponse
	if err := c.get("/api/v1/workspace", &resp); err != nil {
		return nil, err
	}
	return resp.Workspaces, nil
}

func (c *Client) GetCurrentWorkspace() (string, error) {
	var resp struct {
		WorkingDirectory string `json:"working_directory"`
	}
	if err := c.get("/api/v1/current-workspace", &resp); err != nil {
		return "", err
	}
	return resp.WorkingDirectory, nil
}

func (c *Client) SetWorkspace(path string) error {
	return c.post("/api/v1/current-workspace", map[string]string{
		"working_directory": path,
	}, nil)
}

func (c *Client) DeleteWorkspace(path string) error {
	return c.del("/api/v1/workspace?path=" + path)
}

// ─── Config ───

func (c *Client) GetConfigStatus() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.get("/api/v1/config/status", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) UpdateConfig(sessionID string, req types.UpdateConfigRequest) error {
	return c.put("/api/v1/sessions/"+sessionID+"/config", req, nil)
}

// ─── Session Control ───

func (c *Client) Pause(sessionID string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/pause", nil, nil)
}

func (c *Client) Resume(sessionID string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/resume", nil, nil)
}

func (c *Client) Cancel(sessionID string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/cancel", nil, nil)
}

func (c *Client) Complete(sessionID string) error {
	return c.post("/api/v1/sessions/"+sessionID+"/complete", nil, nil)
}

// ─── Approval ───

func (c *Client) Approve(sessionID string, req types.ApproveRequest) error {
	approvalID := req.ApprovalID
	if approvalID == "" {
		approvalID = req.Decision
	}
	return c.post("/api/v1/sessions/"+sessionID+"/approve/"+approvalID, req, nil)
}

func (c *Client) SetTrustLevel(sessionID string, level string) error {
	return c.put("/api/v1/sessions/"+sessionID+"/trust", map[string]string{
		"trust_level": level,
	}, nil)
}

// ─── Archive ───

func (c *Client) SyncArchive(sessionID string) (interface{}, error) {
	var result interface{}
	if err := c.post("/api/v1/sessions/"+sessionID+"/sync-archive", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ─── Version ───

func (c *Client) GetVersion() (string, error) {
	var resp struct {
		Version string `json:"version"`
	}
	if err := c.get("/api/v1/version", &resp); err != nil {
		return "", err
	}
	return resp.Version, nil
}
