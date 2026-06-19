package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
	"devo/internal/taskexec/llmclient"
)

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	Headers map[string]string
}

type Client struct {
	config     Config
	httpClient *http.Client
	tools      []llmclient.ToolDefinition
}

func New(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.Model == "" {
		config.Model = "gpt-4o"
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) SetTools(tools []llmclient.ToolDefinition) {
	c.tools = tools
}

func (c *Client) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	openaiMsgs := make([]openaiMessage, 0, len(messages)+1)

	if systemPrompt != "" {
		openaiMsgs = append(openaiMsgs, openaiMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	for _, msg := range messages {
		om := openaiMessage{
			Role:       string(msg.Role),
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
		}

		if len(msg.ToolCalls) > 0 {
			toolCalls := make([]openaiToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				argsJSON, err := json.Marshal(tc.Params)
				if err != nil {
					argsJSON = []byte("{}")
				}
				toolCalls[i] = openaiToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openaiFunctionCall{
						Name:      tc.ToolName,
						Arguments: string(argsJSON),
					},
				}
			}
			om.ToolCalls = toolCalls
		}

		openaiMsgs = append(openaiMsgs, om)
	}

	reqBody := openaiChatRequest{
		Model:    c.config.Model,
		Messages: openaiMsgs,
	}

	if len(c.tools) > 0 {
		toolDefs := make([]openaiToolDef, len(c.tools))
		for i, t := range c.tools {
			toolDefs[i] = openaiToolDef{
				Type: "function",
				Function: openaiFunctionDef{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Params,
				},
			}
		}
		reqBody.Tools = toolDefs
		reqBody.ToolChoice = "auto"
	}

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.config.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp openaiChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := chatResp.Choices[0]
	result := &llmclient.CompleteResult{}

	if choice.Message.Content != "" {
		result.Text = choice.Message.Content
	}

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]session.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			var params map[string]interface{}
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
					params = map[string]interface{}{"_raw": tc.Function.Arguments}
				}
			}
			if params == nil {
				params = make(map[string]interface{})
			}
			result.ToolCalls[i] = session.ToolCall{
				ID:       tc.ID,
				ToolName: tc.Function.Name,
				Params:   params,
			}
		}
	}

	if chatResp.Usage != nil {
		result.TokenUsage = &tokenmeter.TokenUsage{
			InputTokens:  chatResp.Usage.PromptTokens,
			OutputTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:  chatResp.Usage.TotalTokens,
			Source:       tokenmeter.SourceExact,
		}
	}

	return result, nil
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openaiFunctionCall `json:"function"`
}

type openaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiChatRequest struct {
	Model      string          `json:"model"`
	Messages   []openaiMessage `json:"messages"`
	Tools      []openaiToolDef `json:"tools,omitempty"`
	ToolChoice string          `json:"tool_choice,omitempty"`
}

type openaiToolDef struct {
	Type     string            `json:"type"`
	Function openaiFunctionDef `json:"function"`
}

type openaiFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type openaiChatResponse struct {
	Choices []openaiChoice `json:"choices"`
	Usage   *openaiUsage   `json:"usage,omitempty"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openaiChoice struct {
	Message openaiRespMessage `json:"message"`
}

type openaiRespMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}
