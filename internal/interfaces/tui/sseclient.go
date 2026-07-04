package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"devo/internal/interfaces/tui/messages"
)

type SSEClient struct {
	url                  string
	eventCh              chan messages.SSEEvent
	errCh                chan error
	done                 chan struct{}
	client               *http.Client
	maxReconnectAttempts int
	reconnectDelay       time.Duration
	reconnectAttempt     int
	reconnecting         bool
}

func NewSSEClient() *SSEClient {
	return &SSEClient{
		eventCh:              make(chan messages.SSEEvent, 100),
		errCh:                make(chan error, 1),
		done:                 make(chan struct{}),
		client:               &http.Client{Timeout: 0},
		maxReconnectAttempts: 5,
		reconnectDelay:       1 * time.Second,
	}
}

func (s *SSEClient) SetReconnectConfig(maxAttempts int, initialDelay time.Duration) {
	s.maxReconnectAttempts = maxAttempts
	s.reconnectDelay = initialDelay
}

func (s *SSEClient) Connect(sseURL string) error {
	s.url = sseURL
	s.done = make(chan struct{})

	resp, err := s.doConnect()
	if err != nil {
		return err
	}

	go s.readEvents(resp)
	return nil
}

func (s *SSEClient) doConnect() (*http.Response, error) {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SSE connect: %w", err)
	}
	return resp, nil
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

func (s *SSEClient) IsReconnecting() bool {
	return s.reconnecting
}

func (s *SSEClient) readEvents(resp *http.Response) {
	defer resp.Body.Close()

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
		case <-s.done:
			return
		default:
		}

		s.reconnecting = true
		s.reconnectAttempt += 1
		if s.reconnectAttempt <= s.maxReconnectAttempts {
			s.sendReconnectEvent()
			s.tryReconnect()
		} else {
			select {
			case s.errCh <- err:
			case <-s.done:
			}
		}
	}
}

func (s *SSEClient) sendReconnectEvent() {
	event := messages.SSEEvent{
		Type: "reconnecting",
		Data: map[string]interface{}{
			"attempt":     s.reconnectAttempt,
			"max_attempt": s.maxReconnectAttempts,
		},
	}
	select {
	case s.eventCh <- event:
	case <-s.done:
	}
}

func (s *SSEClient) tryReconnect() {
	delay := s.reconnectDelay * (1 << (s.reconnectAttempt - 1))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
	case <-s.done:
		return
	}

	s.done = make(chan struct{})

	resp, err := s.doConnect()
	if err != nil {
		s.reconnecting = false
		if s.reconnectAttempt < s.maxReconnectAttempts {
			s.reconnectAttempt += 1
			s.sendReconnectEvent()
			s.tryReconnect()
		} else {
			select {
			case s.errCh <- err:
			case <-s.done:
			}
		}
		return
	}

	s.reconnecting = false
	s.reconnectAttempt = 0

	event := messages.SSEEvent{
		Type: "reconnected",
		Data: map[string]interface{}{
			"message": "reconnected successfully",
		},
	}
	select {
	case s.eventCh <- event:
	case <-s.done:
		resp.Body.Close()
		return
	}

	go s.readEvents(resp)
}

func (s *SSEClient) parseEvent(eventType string, eventID int64, data string) messages.SSEEvent {
	event := messages.SSEEvent{
		Type: eventType,
	}

	if data != "" {
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(data), &parsed); err == nil {
			if t, ok := parsed["type"].(string); ok {
				event.Type = t
			}
			if d, ok := parsed["data"].(map[string]interface{}); ok {
				event.Data = d
			} else {
				event.Data = parsed
			}
		} else {
			event.Data = map[string]interface{}{
				"raw": data,
			}
		}
	}

	_ = eventID
	return event
}
