package messages

import "time"

type SSEEvent struct {
	Type string
	Data map[string]interface{}
}

type APIResponse struct {
	Kind string
	Data interface{}
	Err  error
}

type ApprovalDecision struct {
	Approved bool
}

type TickMsg time.Time
