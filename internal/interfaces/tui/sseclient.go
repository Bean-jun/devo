package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"devo/internal/interfaces/tui/messages"
)

type SSEClient struct {
	url     string
	eventCh chan messages.SSEEvent
	errCh   chan error
	done    chan struct{}
	client  *http.Client
}

func NewSSEClient() *SSEClient {
	return &SSEClient{
		eventCh: make(chan messages.SSEEvent, 100),
		errCh:   make(chan error, 1),
		done:    make(chan struct{}),
		client:  &http.Client{Timeout: 0},
	}
}

func (s *SSEClient) Connect(sseURL string) error {
	s.url = sseURL
	s.done = make(chan struct{})

	req, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}

	go s.readEvents(resp)
	return nil
}

func (s *SSEClient) Disconnect() {
	select {
	case <-s.done:
		return
	default:
		close(s.done)
	}
}

func (s *SSEClient) Events() <-chan messages.SSEEvent {
	return s.eventCh
}

func (s *SSEClient) Errors() <-chan error {
	return s.errCh
}

func (s *SSEClient) readEvents(resp *http.Response) {
	defer resp.Body.Close()
	defer close(s.eventCh)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var eventID int64
	var dataLines []string

	for scanner.Scan() {
		select {
		case <-s.done:
			return
		default:
		}

		line := scanner.Text()

		if line == "" {
			if len(dataLines) > 0 {
				event := s.parseEvent(eventType, eventID, strings.Join(dataLines, "\n"))
				select {
				case s.eventCh <- event:
				case <-s.done:
					return
				}
			}
			eventType = ""
			eventID = 0
			dataLines = nil
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "id:") {
			idStr := strings.TrimSpace(strings.TrimPrefix(line, "id:"))
			eventID, _ = strconv.ParseInt(idStr, 10, 64)
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case s.errCh <- err:
		case <-s.done:
		}
	}
}

func (s *SSEClient) parseEvent(eventType string, eventID int64, data string) messages.SSEEvent {
	event := messages.SSEEvent{
		Type: eventType,
	}

	if data != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(data), &parsed); err == nil {
			event.Data = parsed
		} else {
			event.Data = map[string]interface{}{
				"raw": data,
			}
		}
	}

	_ = eventID
	return event
}
