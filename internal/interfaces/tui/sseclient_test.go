package tui

import (
	"testing"
	"time"
)

func TestSSEClient_NewHasDefaults(t *testing.T) {
	client := NewSSEClient()
	if client.maxReconnectAttempts != 5 {
		t.Errorf("default maxReconnectAttempts = %d, want 5", client.maxReconnectAttempts)
	}
	if client.reconnectDelay != 1*time.Second {
		t.Errorf("default reconnectDelay = %v, want 1s", client.reconnectDelay)
	}
}

func TestSSEClient_SetReconnectConfig(t *testing.T) {
	client := NewSSEClient()
	client.SetReconnectConfig(3, 2*time.Second)

	if client.maxReconnectAttempts != 3 {
		t.Errorf("maxReconnectAttempts = %d, want 3", client.maxReconnectAttempts)
	}
	if client.reconnectDelay != 2*time.Second {
		t.Errorf("reconnectDelay = %v, want 2s", client.reconnectDelay)
	}
}

func TestSSEClient_IsReconnecting(t *testing.T) {
	client := NewSSEClient()
	if client.IsReconnecting() {
		t.Error("IsReconnecting should be false initially")
	}
}

func TestSSEClient_Disconnect(t *testing.T) {
	client := NewSSEClient()
	client.Disconnect()
}

func TestSSEClient_Events(t *testing.T) {
	client := NewSSEClient()
	ch := client.Events()
	if ch == nil {
		t.Error("Events channel should not be nil")
	}
}

func TestSSEClient_Errors(t *testing.T) {
	client := NewSSEClient()
	ch := client.Errors()
	if ch == nil {
		t.Error("Errors channel should not be nil")
	}
}
