package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu         sync.Mutex
	reqCounter int
	requestDir string
)

func main() {
	requestDir = filepath.Join(".", "requests")
	os.MkdirAll(requestDir, 0755)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", handleChatCompletions)

	addr := ":8080"
	log.Printf("Mock LLM server starting on http://localhost%s", addr)
	log.Printf("Requests saved to: %s", requestDir)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
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

	var req struct {
		Stream bool `json:"stream"`
	}
	json.Unmarshal(body, &req)

	if req.Stream {
		handleStream(w, body)
	} else {
		handleNonStream(w)
	}
}

func handleStream(w http.ResponseWriter, body []byte) {
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

		data, _ := json.Marshal(chunkResp)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

func handleNonStream(w http.ResponseWriter) {
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

	json.NewEncoder(w).Encode(resp)
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
