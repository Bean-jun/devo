package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"devo/internal/core/session"
	"devo/internal/core/tokenmeter"
	"devo/internal/taskexec/llmclient"
	"devo/internal/taskexec/tools"
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
	registry   *tools.Registry
}

func New(config Config, registry *tools.Registry) *Client {
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
		registry: registry,
	}
}

func (c *Client) Complete(ctx context.Context, messages []session.Message, systemPrompt string) (*llmclient.CompleteResult, error) {
	reqBody := c.buildChatRequest(messages, systemPrompt, false)

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respBody, err := c.doChatRequest(ctx, bodyJSON)
	if err != nil {
		return nil, err
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
		result.ToolCalls = convertToolCalls(choice.Message.ToolCalls)
	}

	if chatResp.Usage != nil {
		result.TokenUsage = convertUsage(chatResp.Usage)
	}

	return result, nil
}

func (c *Client) CompleteStream(ctx context.Context, messages []session.Message, systemPrompt string, callback llmclient.StreamCallback) error {
	reqBody := c.buildChatRequest(messages, systemPrompt, true)

	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.config.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("openai api error (status %d): %s", resp.StatusCode, string(body))
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return err
	}

	return c.parseSSEStream(ctx, resp.Body, callback)
}

func (c *Client) buildChatRequest(messages []session.Message, systemPrompt string, stream bool) *openaiChatRequest {
	reqBody := &openaiChatRequest{
		Model:    c.config.Model,
		Messages: convertMessages(messages, systemPrompt),
	}

	if stream {
		reqBody.Stream = true
		reqBody.StreamOptions = &openaiStreamOptions{
			IncludeUsage: true,
		}
	}

	if c.registry != nil {
		toolList := c.registry.ListTools()
		if len(toolList) > 0 {
			reqBody.Tools = buildToolDefs(toolList)
			reqBody.ToolChoice = "auto"
		}
	}

	return reqBody
}

func convertMessages(messages []session.Message, systemPrompt string) []openaiMessage {
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

	return openaiMsgs
}

func buildToolDefs(toolList []tools.Tool) []openaiToolDef {
	result := make([]openaiToolDef, len(toolList))
	for i, t := range toolList {
		result[i] = openaiToolDef{
			Type: "function",
			Function: openaiFunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.ParamsSchema(),
			},
		}
	}
	return result
}

func (c *Client) doChatRequest(ctx context.Context, bodyJSON []byte) ([]byte, error) {
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

	return respBody, nil
}

func convertToolCalls(toolCalls []openaiToolCall) []session.ToolCall {
	result := make([]session.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		var params map[string]interface{}
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				params = map[string]interface{}{"_raw": tc.Function.Arguments}
			}
		}
		if params == nil {
			params = make(map[string]interface{})
		}
		result[i] = session.ToolCall{
			ID:       tc.ID,
			ToolName: tc.Function.Name,
			Params:   params,
		}
	}
	return result
}

func convertUsage(usage *openaiUsage) *tokenmeter.TokenUsage {
	tu := &tokenmeter.TokenUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  usage.TotalTokens,
		Source:       tokenmeter.SourceExact,
	}
	if usage.PromptTokensDetails != nil {
		tu.CachedTokens = usage.PromptTokensDetails.CachedTokens
	}
	return tu
}

func (c *Client) parseSSEStream(ctx context.Context, body io.Reader, callback llmclient.StreamCallback) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fullTextBuilder strings.Builder
	var accumulatedToolCalls []accumulatedToolCall
	var usage *tokenmeter.TokenUsage
	var finishReason string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			callback(llmclient.StreamEvent{Type: "error", Err: ctx.Err()})
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			break
		}

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			usage = convertUsage(chunk.Usage)
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		choiceFinishReason := chunk.Choices[0].FinishReason
		if choiceFinishReason != "" {
			finishReason = choiceFinishReason
		}

		if delta.Content != "" {
			fullTextBuilder.WriteString(delta.Content)
			callback(llmclient.StreamEvent{
				Type:     "token",
				Token:    delta.Content,
				FullText: fullTextBuilder.String(),
			})
		}

		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				for len(accumulatedToolCalls) <= tc.Index {
					accumulatedToolCalls = append(accumulatedToolCalls, accumulatedToolCall{})
				}
				acc := &accumulatedToolCalls[tc.Index]

				if tc.ID != "" {
					acc.ID = tc.ID
				}
				if tc.Function.Name != "" {
					acc.Name = tc.Function.Name
				}
				acc.Arguments += tc.Function.Arguments
			}
		}
	}

	if finishReason != "" {
		fullText := fullTextBuilder.String()
		toolCalls := make([]session.ToolCall, 0, len(accumulatedToolCalls))
		for _, acc := range accumulatedToolCalls {
			if acc.ID == "" {
				continue
			}
			var params map[string]interface{}
			if acc.Arguments != "" {
				if err := json.Unmarshal([]byte(acc.Arguments), &params); err != nil {
					params = map[string]interface{}{"_raw": acc.Arguments}
				}
			}
			if params == nil {
				params = make(map[string]interface{})
			}
			toolCalls = append(toolCalls, session.ToolCall{
				ID:       acc.ID,
				ToolName: acc.Name,
				Params:   params,
			})
		}

		callback(llmclient.StreamEvent{
			Type:         "done",
			FullText:     fullText,
			ToolCalls:    toolCalls,
			FinishReason: finishReason,
			TokenUsage:   usage,
		})
	}

	if err := scanner.Err(); err != nil {
		callback(llmclient.StreamEvent{Type: "error", Err: err})
		return fmt.Errorf("read sse stream: %w", err)
	}

	return nil
}

type accumulatedToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type openaiStreamChunk struct {
	Choices []openaiStreamChoice `json:"choices"`
	Usage   *openaiUsage         `json:"usage,omitempty"`
}

type openaiStreamChoice struct {
	Delta        openaiStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

type openaiStreamDelta struct {
	Content   string           `json:"content,omitempty"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
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
	Index    int                `json:"index,omitempty"`
}

type openaiFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiChatRequest struct {
	Model         string               `json:"model"`
	Messages      []openaiMessage      `json:"messages"`
	Tools         []openaiToolDef      `json:"tools,omitempty"`
	ToolChoice    string               `json:"tool_choice,omitempty"`
	Stream        bool                 `json:"stream,omitempty"`
	StreamOptions *openaiStreamOptions `json:"stream_options,omitempty"`
}

type openaiStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
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

type openaiPromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type openaiUsage struct {
	PromptTokens        int                        `json:"prompt_tokens"`
	CompletionTokens    int                        `json:"completion_tokens"`
	TotalTokens         int                        `json:"total_tokens"`
	PromptTokensDetails *openaiPromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

type openaiChoice struct {
	Message openaiRespMessage `json:"message"`
}

type openaiRespMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}
