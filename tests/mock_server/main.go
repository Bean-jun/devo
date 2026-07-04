package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	mu         sync.Mutex
	reqCounter int
	requestDir string
	sessions   sync.Map
	holdDelay  int64
)

type sessionState struct {
	mu          sync.Mutex
	reqCount    int
	lastValid   bool
	lastMissing []string
}

type toolDef struct {
	Description string
	Parameters  map[string]interface{}
}

var toolDefs = map[string]toolDef{
	"read_file": {
		Description: "Read the contents of a file at the given path",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径（相对于工作目录或绝对路径）",
				},
			},
			"required": []string{"path"},
		},
	},
	"write_file": {
		Description: "Create a new file or overwrite an existing file with the given content",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "文件路径（相对于工作目录）",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "要写入的文件内容",
				},
			},
			"required": []string{"path", "content"},
		},
	},
	"edit_file": {
		Description: "Edit a file using replace or patch mode",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目标文件路径（相对于工作目录）",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "编辑模式：replace（查找替换）或 patch（unified diff 补丁）",
					"enum":        []string{"replace", "patch"},
				},
				"old_str": map[string]interface{}{
					"type":        "string",
					"description": "（mode=replace 时必填）要替换的原始文本",
				},
				"new_str": map[string]interface{}{
					"type":        "string",
					"description": "（mode=replace 时选填）替换后的新文本",
				},
				"patch": map[string]interface{}{
					"type":        "string",
					"description": "（mode=patch 时必填）unified diff 格式的补丁内容",
				},
			},
			"required": []string{"path", "mode"},
		},
	},
	"list_files": {
		Description: "List files and directories at the given path",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "目录路径（相对于工作目录或绝对路径），默认为工作目录根目录",
				},
				"max_depth": map[string]interface{}{
					"type":        "integer",
					"description": "最大遍历深度，默认 1",
				},
				"max_files": map[string]interface{}{
					"type":        "integer",
					"description": "最大返回文件数，默认 500",
				},
			},
		},
	},
	"search_codebase": {
		Description: "Search for a pattern in files within the working directory",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "搜索的正则表达式模式",
				},
				"path": map[string]interface{}{
					"type":        "string",
					"description": "搜索路径（相对于工作目录），默认为工作目录",
				},
			},
			"required": []string{"pattern"},
		},
	},
	"execute_command": {
		Description: "Execute a shell command with security filtering and timeout control",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "要执行的 shell 命令",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "命令执行超时时间（秒），默认 30",
				},
				"mode": map[string]interface{}{
					"type":        "string",
					"description": "执行模式：sync(等待完成)、async(启动后台任务)、auto(自动检测，默认)",
					"enum":        []string{"sync", "async", "auto"},
				},
			},
			"required": []string{"command"},
		},
	},
	"exec_python": {
		Description: "Execute a Python code snippet and return the output",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"code": map[string]interface{}{
					"type":        "string",
					"description": "Python 代码片段，通过 python -c 执行",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "执行超时时间（秒），默认 30",
				},
			},
			"required": []string{"code"},
		},
	},
	"use_skill": {
		Description: "Load a skill to get detailed instructions for a specific task domain",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"skill_name": map[string]interface{}{
					"type":        "string",
					"description": "The name of the skill to load",
				},
			},
			"required": []string{"skill_name"},
		},
	},
}

var toolParams map[string]map[string]interface{}

func initToolParams() {
	toolParams = map[string]map[string]interface{}{
		"read_file":       {"path": "README.md"},
		"write_file":      {"path": "mock_output.txt", "content": "Hello from mock server"},
		"edit_file":       {"path": "mock_output.txt", "mode": "replace", "old_str": "Hello from mock server", "new_str": "Hello from mock server (updated)"},
		"list_files":      {"path": ".", "max_depth": float64(1)},
		"search_codebase": {"pattern": "func main"},
		"execute_command": {"command": "powershell -Command \"for ($i=1; $i -le 5; $i++) { Write-Output ('Line '+$i+': Hello from mock server'); Start-Sleep -Seconds 1 }\"", "timeout_seconds": float64(30)},
		"exec_python":     {"code": "import time\nfor i in range(1, 6):\n    print(f'Line {i}: Hello from mock server')\n    time.sleep(1)"},
		"use_skill":       {"skill_name": "Go Expert"},
	}
}

var toolCallRe = regexp.MustCompile(`\$\{([^}]+)\}\$`)

type chatMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCallMsg   `json:"tool_calls,omitempty"`
}

type toolCallMsg struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function functionCallMsg `json:"function"`
}

type functionCallMsg struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type holdRequest struct {
	DelayMs int64 `json:"delay_ms"`
}

func main() {
	initToolParams()

	if dir := os.Getenv("REQUEST_DIR"); dir != "" {
		requestDir = dir
	} else {
		requestDir = filepath.Join(".", "requests")
	}
	os.MkdirAll(requestDir, 0755)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", handleChatCompletions)
	mux.HandleFunc("/v1/tool_defs", handleToolDefs)
	mux.HandleFunc("/v1/hold", handleHold)
	mux.HandleFunc("/v1/reset", handleReset)
	mux.HandleFunc("/v1/validation/{session_id}", handleValidation)
	mux.HandleFunc("/sessions", handleListSessions)

	addr := ":8080"
	if port := os.Getenv("MOCK_PORT"); port != "" {
		addr = ":" + port
	}
	log.Printf("Mock LLM server starting on http://localhost%s", addr)
	log.Printf("Requests saved to: %s", requestDir)
	log.Printf("Supported tools: %s", toolNames())
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func toolNames() string {
	names := make([]string, 0, len(toolDefs))
	for n := range toolDefs {
		names = append(names, n)
	}
	return strings.Join(names, ", ")
}

func handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	mu.Lock()
	reqCounter++
	filename := filepath.Join(requestDir, fmt.Sprintf("req_%03d.json", reqCounter))
	mu.Unlock()

	prettyJSON, _ := formatJSON(body)
	if err := os.WriteFile(filename, prettyJSON, 0644); err != nil {
		log.Printf("Failed to save request: %v", err)
	} else {
		log.Printf("Saved: %s (%d bytes)", filename, len(body))
	}

	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse request: %v", err), http.StatusBadRequest)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		sessionID = "default"
	}
	trackSession(sessionID)

	validationErr := validateToolCalls(req.Messages, sessionID)
	if validationErr != "" {
		log.Printf("[REQ-%03d] [session=%s] VALIDATION FAILED: %s", reqCounter, sessionID, validationErr)
		sendError(w, validationErr)
		return
	}

	toolNames := extractToolNames(req.Messages)
	if len(toolNames) > 0 {
		log.Printf("[REQ-%03d] [session=%s] Tool calls requested: %v", reqCounter, sessionID, toolNames)
		if req.Stream {
			handleStreamToolCalls(w, body, toolNames, sessionID, reqCounter)
		} else {
			handleNonStreamToolCalls(w, toolNames, sessionID, reqCounter)
		}
		return
	}

	if hasToolResults(req.Messages) {
		log.Printf("[REQ-%03d] [session=%s] Tool results received, returning final response", reqCounter, sessionID)
		if req.Stream {
			handleStreamFinal(w, body, sessionID, reqCounter)
		} else {
			handleNonStreamFinal(w, sessionID, reqCounter)
		}
		return
	}

	if req.Stream {
		handleStreamDefault(w, body, sessionID, reqCounter)
	} else {
		handleNonStreamDefault(w, sessionID, reqCounter)
	}
}

func trackSession(sessionID string) {
	val, _ := sessions.LoadOrStore(sessionID, &sessionState{})
	state := val.(*sessionState)
	state.mu.Lock()
	state.reqCount++
	count := state.reqCount
	state.mu.Unlock()
	log.Printf("[session=%s] request #%d", sessionID, count)
}

func extractToolNames(msgs []chatMessage) []string {
	userIdx := -1
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			userIdx = i
			break
		}
	}
	if userIdx < 0 {
		return nil
	}

	var content string
	// Try direct unmarshal to string (OpenAI format: content is already a string)
	if err := json.Unmarshal(msgs[userIdx].Content, &content); err != nil {
		log.Printf("[DEBUG] extractToolNames: failed to unmarshal content to string: %v", err)
		return nil
	}
	log.Printf("[DEBUG] extractToolNames: user content = %q", content)
	matches := toolCallRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		log.Printf("[DEBUG] extractToolNames: no matches found")
		return nil
	}

	// Collect all tool names from all matches
	var allNames []string
	for _, match := range matches {
		if len(match) >= 2 {
			allNames = append(allNames, strings.Split(match[1], ",")...)
		}
	}
	if len(allNames) == 0 {
		return nil
	}

	for i := userIdx + 1; i < len(msgs); i++ {
		if msgs[i].Role == "assistant" && len(msgs[i].ToolCalls) > 0 {
			return nil
		}
	}

	result := make([]string, 0, len(allNames))
	for _, n := range allNames {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := toolDefs[n]; !ok {
			log.Printf("WARNING: unknown tool '%s', skipping", n)
			continue
		}
		result = append(result, n)
	}
	return result
}

func hasToolResults(msgs []chatMessage) bool {
	hasToolCalls := false
	hasToolResults := false
	toolCallIDs := make(map[string]bool)

	for _, m := range msgs {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			hasToolCalls = true
			for _, tc := range m.ToolCalls {
				toolCallIDs[tc.ID] = true
			}
		}
		if m.Role == "tool" {
			hasToolResults = true
			delete(toolCallIDs, m.ToolCallID)
		}
	}

	if !hasToolCalls || !hasToolResults {
		return false
	}

	return len(toolCallIDs) == 0
}

func validateToolCalls(msgs []chatMessage, sessionID string) string {
	var lastAssistantWithCalls *chatMessage
	toolCallIDs := make(map[string]bool)
	respondedIDs := make(map[string]bool)

	for i := range msgs {
		m := &msgs[i]
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			lastAssistantWithCalls = m
			toolCallIDs = make(map[string]bool)
			respondedIDs = make(map[string]bool)
			for _, tc := range m.ToolCalls {
				toolCallIDs[tc.ID] = true
			}
			continue
		}
		if m.Role == "tool" && lastAssistantWithCalls != nil {
			if _, exists := toolCallIDs[m.ToolCallID]; exists {
				respondedIDs[m.ToolCallID] = true
			}
		}
		if m.Role == "user" || (m.Role == "assistant" && len(m.ToolCalls) == 0) {
			lastAssistantWithCalls = nil
			toolCallIDs = nil
			respondedIDs = nil
		}
	}

	val, _ := sessions.LoadOrStore(sessionID, &sessionState{})
	state := val.(*sessionState)
	state.mu.Lock()
	defer state.mu.Unlock()

	if lastAssistantWithCalls == nil || len(toolCallIDs) == 0 {
		state.lastValid = true
		state.lastMissing = nil
		return ""
	}

	var missing []string
	for id := range toolCallIDs {
		if !respondedIDs[id] {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		state.lastValid = false
		state.lastMissing = missing
		return fmt.Sprintf(
			"An assistant message with 'tool_calls' must be followed by tool messages responding to each 'tool_call_id'. "+
				"The following tool_call_ids did not have response messages: %s",
			strings.Join(missing, ", "),
		)
	}

	state.lastValid = true
	state.lastMissing = nil
	return ""
}

func sendError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"message": msg,
			"type":    "invalid_request_error",
			"code":    "tool_validation_failed",
		},
	})
}

func buildToolCalls(toolNames []string) []map[string]interface{} {
	calls := make([]map[string]interface{}, len(toolNames))
	for i, name := range toolNames {
		params := toolParams[name]
		argsJSON, _ := json.Marshal(params)
		calls[i] = map[string]interface{}{
			"id":   fmt.Sprintf("call_mock_%s_%d", name, i+1),
			"type": "function",
			"function": map[string]interface{}{
				"name":      name,
				"arguments": string(argsJSON),
			},
		}
	}
	return calls
}

func handleStreamToolCalls(w http.ResponseWriter, body []byte, toolNames []string, sessionID string, reqNum int) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	delay := getHoldDelay()
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	chunkID := fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano())

	var allChunks []map[string]interface{}

	roleChunk := map[string]interface{}{
		"id":      chunkID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "mock-model",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]interface{}{
					"role":    "assistant",
					"content": nil,
				},
				"finish_reason": nil,
			},
		},
	}
	allChunks = append(allChunks, roleChunk)
	sendSSE(w, flusher, roleChunk)

	for i, name := range toolNames {
		params := toolParams[name]
		argsJSON, _ := json.Marshal(params)
		argsStr := string(argsJSON)

		toolCallChunk := map[string]interface{}{
			"id":      chunkID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "mock-model",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"tool_calls": []map[string]interface{}{
							{
								"index": i,
								"id":    fmt.Sprintf("call_mock_%s_%d", name, i+1),
								"type":  "function",
								"function": map[string]interface{}{
									"name":      name,
									"arguments": argsStr,
								},
							},
						},
					},
					"finish_reason": nil,
				},
			},
		}
		allChunks = append(allChunks, toolCallChunk)
		sendSSE(w, flusher, toolCallChunk)
		time.Sleep(50 * time.Millisecond)
	}

	finishChunk := map[string]interface{}{
		"id":      chunkID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   "mock-model",
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         map[string]interface{}{},
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     estimatePromptTokens(body),
			"completion_tokens": len(toolNames) * 10,
			"total_tokens":      estimatePromptTokens(body) + len(toolNames)*10,
		},
	}
	allChunks = append(allChunks, finishChunk)
	sendSSE(w, flusher, finishChunk)

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	saveResponse(reqNum, allChunks)
}

func handleNonStreamToolCalls(w http.ResponseWriter, toolNames []string, sessionID string, reqNum int) {
	w.Header().Set("Content-Type", "application/json")

	calls := buildToolCalls(toolNames)

	resp := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "mock-model",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":       "assistant",
					"content":    nil,
					"tool_calls": calls,
				},
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     100,
			"completion_tokens": len(toolNames) * 10,
			"total_tokens":      100 + len(toolNames)*10,
		},
	}

	saveResponse(reqNum, resp)
	json.NewEncoder(w).Encode(resp)
}

func handleStreamFinal(w http.ResponseWriter, body []byte, sessionID string, reqNum int) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	delay := getHoldDelay()
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	chunkID := fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano())

	finalText := "I've completed the requested operations based on the tool results. The task has been executed successfully."
	words := strings.Fields(finalText)
	totalTokens := len(words)

	var allChunks []map[string]interface{}

	for i, word := range words {
		content := word
		if i < len(words)-1 {
			content += " "
		}

		chunk := map[string]interface{}{
			"id":      chunkID,
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "mock-model",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"delta": map[string]interface{}{
						"content": content,
					},
					"finish_reason": nil,
				},
			},
		}

		if i == len(words)-1 {
			chunk["choices"].([]map[string]interface{})[0]["finish_reason"] = "stop"
			chunk["usage"] = map[string]interface{}{
				"prompt_tokens":     estimatePromptTokens(body),
				"completion_tokens": totalTokens,
				"total_tokens":      estimatePromptTokens(body) + totalTokens,
			}
		}

		allChunks = append(allChunks, chunk)
		sendSSE(w, flusher, chunk)
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	saveResponse(reqNum, allChunks)
}

func handleNonStreamFinal(w http.ResponseWriter, sessionID string, reqNum int) {
	w.Header().Set("Content-Type", "application/json")

	resp := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "mock-model",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "I've completed the requested operations based on the tool results. The task has been executed successfully.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     100,
			"completion_tokens": 20,
			"total_tokens":      120,
		},
	}

	saveResponse(reqNum, resp)
	json.NewEncoder(w).Encode(resp)
}

func handleStreamDefault(w http.ResponseWriter, body []byte, sessionID string, reqNum int) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	mockTokens := []string{
		"Hello", "!", " ", "I'm", " ", "a", " ", "mock", " ", "LLM", " ",
		"server", ".", " ", "This", " ", "is", " ", "a", " ", "simulated",
		" ", "response", " ", "for", " ", "testing", " ", "Devo", ".",
	}

	totalTokens := len(mockTokens)
	chunkSize := 3
	chunkCount := (totalTokens + chunkSize - 1) / chunkSize

	var allChunks []map[string]interface{}

	for i := 0; i < chunkCount; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > totalTokens {
			end = totalTokens
		}

		chunk := ""
		for _, t := range mockTokens[start:end] {
			chunk += t
		}

		delta := map[string]interface{}{
			"role":    "assistant",
			"content": chunk,
		}

		chunkResp := map[string]interface{}{
			"id":      fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano()),
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "mock-model",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"delta":         delta,
					"finish_reason": nil,
				},
			},
		}

		if i == chunkCount-1 {
			chunkResp["choices"].([]map[string]interface{})[0]["finish_reason"] = "stop"
			chunkResp["usage"] = map[string]interface{}{
				"prompt_tokens":     estimatePromptTokens(body),
				"completion_tokens": totalTokens,
				"total_tokens":      estimatePromptTokens(body) + totalTokens,
			}
		}

		allChunks = append(allChunks, chunkResp)
		data, _ := json.Marshal(chunkResp)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	saveResponse(reqNum, allChunks)
}

func handleNonStreamDefault(w http.ResponseWriter, sessionID string, reqNum int) {
	w.Header().Set("Content-Type", "application/json")

	resp := map[string]interface{}{
		"id":      fmt.Sprintf("chatcmpl-mock-%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   "mock-model",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "Hello! I'm a mock LLM server. This is a simulated response for testing Devo.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     100,
			"completion_tokens": 20,
			"total_tokens":      120,
		},
	}

	saveResponse(reqNum, resp)
	json.NewEncoder(w).Encode(resp)
}

func handleToolDefs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolDefs)
}

func handleListSessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	result := make(map[string]int)
	sessions.Range(func(key, value interface{}) bool {
		state := value.(*sessionState)
		state.mu.Lock()
		result[key.(string)] = state.reqCount
		state.mu.Unlock()
		return true
	})
	json.NewEncoder(w).Encode(result)
}

func sendSSE(w http.ResponseWriter, flusher http.Flusher, data map[string]interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

func estimatePromptTokens(body []byte) int {
	return len(body) / 4
}

func formatJSON(raw []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return raw, nil
	}
	return json.MarshalIndent(obj, "", "  ")
}

func saveResponse(reqNum int, resp interface{}) {
	filename := filepath.Join(requestDir, fmt.Sprintf("req_%03d_resp.json", reqNum))
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal response for req %d: %v", reqNum, err)
		return
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Failed to save response for req %d: %v", reqNum, err)
	} else {
		log.Printf("Saved response: %s (%d bytes)", filename, len(data))
	}
}

func handleHold(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req holdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	mu.Lock()
	holdDelay = req.DelayMs
	mu.Unlock()
	log.Printf("Hold delay set to %dms", req.DelayMs)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"delay_ms": req.DelayMs})
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	mu.Lock()
	holdDelay = 0
	mu.Unlock()
	log.Printf("Hold delay reset to 0")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{"delay_ms": 0})
}

func handleValidation(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	val, ok := sessions.Load(sessionID)
	if !ok {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session_id": sessionID,
			"last_validation": map[string]interface{}{
				"passed":                true,
				"missing_tool_call_ids": []string{},
				"request_count":         0,
			},
		})
		return
	}
	state := val.(*sessionState)
	state.mu.Lock()
	defer state.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"session_id": sessionID,
		"last_validation": map[string]interface{}{
			"passed":                state.lastValid,
			"missing_tool_call_ids": state.lastMissing,
			"request_count":         state.reqCount,
		},
	})
}

func getHoldDelay() int64 {
	mu.Lock()
	defer mu.Unlock()
	return holdDelay
}
