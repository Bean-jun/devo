package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type DevoClient struct {
	BaseURL string
	client  *http.Client
}

type SessionInfo struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

type CreateSessionResponse struct {
	ID    string `json:"id"`
	State string `json:"state"`
}

type MessageResponse struct {
	Status string `json:"status"`
}

type ControlResponse struct {
	State string `json:"state"`
}

func NewDevoClient(baseURL string) *DevoClient {
	return &DevoClient{
		BaseURL: baseURL,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *DevoClient) CreateSession(workingDir string) (string, error) {
	body := map[string]string{"working_directory": workingDir}
	resp, err := PostJSON(c.BaseURL+"/api/v1/sessions", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result CreateSessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func (c *DevoClient) SendMessage(sessionID, message string) (*http.Response, error) {
	body := map[string]string{"content": message}
	resp, err := PostJSON(c.BaseURL+"/api/v1/sessions/"+sessionID+"/messages", body)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *DevoClient) SendMessageAndWait(sessionID, message string) (*http.Response, string, error) {
	resp, err := c.SendMessage(sessionID, message)
	if err != nil {
		return nil, "", err
	}
	bodyStr := ReadBody(resp)
	resp.Body.Close()
	return resp, bodyStr, nil
}

func (c *DevoClient) Pause(sessionID string) (*http.Response, error) {
	return http.Post(c.BaseURL+"/api/v1/sessions/"+sessionID+"/pause", "application/json", nil)
}

func (c *DevoClient) Resume(sessionID string) (*http.Response, error) {
	return http.Post(c.BaseURL+"/api/v1/sessions/"+sessionID+"/resume", "application/json", nil)
}

func (c *DevoClient) Cancel(sessionID string) (*http.Response, error) {
	return http.Post(c.BaseURL+"/api/v1/sessions/"+sessionID+"/cancel", "application/json", nil)
}

func (c *DevoClient) SetTrustLevel(sessionID, trustLevel string) (*http.Response, error) {
	body := map[string]string{"trust_level": trustLevel}
	return PutJSON(c.BaseURL+"/api/v1/sessions/"+sessionID+"/trust", body)
}

func (c *DevoClient) GetState(sessionID string) (string, error) {
	var result struct {
		State string `json:"state"`
	}
	if err := GetJSON(c.BaseURL+"/api/v1/sessions/"+sessionID, &result); err != nil {
		return "", err
	}
	return result.State, nil
}

func (c *DevoClient) WaitForState(sessionID, targetState string, timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout waiting for state %s", targetState)
		default:
			state, err := c.GetState(sessionID)
			if err != nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if state == targetState {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (c *DevoClient) WaitForStateNot(sessionID, notState string, timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout waiting for state != %s", notState)
		default:
			state, err := c.GetState(sessionID)
			if err != nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if state != notState {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (c *DevoClient) GetSession(sessionID string) (*SessionInfo, error) {
	var result SessionInfo
	if err := GetJSON(c.BaseURL+"/api/v1/sessions/"+sessionID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func DecodeResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

func ReadBodyString(resp *http.Response) string {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}
