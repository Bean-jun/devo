package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"devo/internal/interfaces/tui/types"
)

type SSEClient struct {
	baseURL string
	eventCh chan types.SSEEvent
	errCh   chan error
	done    chan struct{}
}

func NewSSEClient(baseURL string) *SSEClient {
	return &SSEClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		eventCh: make(chan types.SSEEvent, 100),
		errCh:   make(chan error, 10),
		done:    make(chan struct{}),
	}
}

func (s *SSEClient) Connect(sessionID string) error {
	url := s.baseURL + "/api/v1/sessions/" + sessionID + "/events"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect failed: %w", err)
	}

	s.done = make(chan struct{})
	go s.readLoop(resp)
	return nil
}

func (s *SSEClient) readLoop(resp *http.Response) {
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)

	for {
		select {
		case <-s.done:
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			select {
			case s.errCh <- err:
			case <-s.done:
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)

			var wrapper map[string]interface{}
			if err := json.Unmarshal([]byte(data), &wrapper); err != nil {
				continue
			}

			eventType := "message"
			if t, ok := wrapper["type"].(string); ok {
				eventType = t
			}

			var payloadData map[string]interface{}
			if nested, ok := wrapper["data"].(map[string]interface{}); ok {
				payloadData = nested
			} else {
				payloadData = wrapper
			}

			evt := types.SSEEvent{
				Type: eventType,
				Data: payloadData,
			}

			select {
			case s.eventCh <- evt:
			case <-s.done:
				return
			}
		}
	}
}

func (s *SSEClient) Disconnect() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *SSEClient) Events() <-chan types.SSEEvent {
	return s.eventCh
}

func (s *SSEClient) Errors() <-chan error {
	return s.errCh
}
