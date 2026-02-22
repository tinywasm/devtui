package devtui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	tinyfmt "github.com/tinywasm/fmt"
)

func TestClientModeSSE(t *testing.T) {
	// Setup mock SSE server
	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)

		// Send a log message
		dto := tabContentDTO{
			Id:          "123",
			Timestamp:   "10:00:00",
			Content:     "Hello from Server",
			Type:        tinyfmt.Msg.Info,
			TabTitle:    "SHORTCUTS", // Default tab
			HandlerName: "TestHandler",
			HandlerType: handlerTypeLoggable,
		}
		data, _ := json.Marshal(dto)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()

		// Keep connection open for a bit
		time.Sleep(100 * time.Millisecond)
	}))
	defer sseServer.Close()

	// Initialize TUI in Client Mode
	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  sseServer.URL,
		ExitChan:   make(chan bool),
	}
	tui := NewTUI(config)
	tui.SetTestMode(true) // Ensure deterministic behavior if applicable

	// Wait for message to arrive in channel
	select {
	case msg := <-tui.tabContentsChan:
		if msg.Content != "Hello from Server" {
			t.Errorf("Expected content 'Hello from Server', got '%s'", msg.Content)
		}
		if msg.tabSection.title != "SHORTCUTS" {
			t.Errorf("Expected tab 'SHORTCUTS', got '%s'", msg.tabSection.title)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for SSE message")
	}

	// Close ExitChan to stop SSE client
	close(config.ExitChan)
}

func TestClientModeKeyboard(t *testing.T) {
	actionReceived := make(chan string, 1)

	// Setup mock action server
	actionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/action" {
			key := r.URL.Query().Get("key")
			actionReceived <- key
		}
		w.WriteHeader(200)
	}))
	defer actionServer.Close()

	// Initialize TUI in Client Mode
	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  actionServer.URL,
		ExitChan:   make(chan bool),
	}
	tui := NewTUI(config)

	// Simulate 'r' key press
	// We call Update directly
	tui.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	// Verify request sent
	select {
	case key := <-actionReceived:
		if key != "r" {
			t.Errorf("Expected action key 'r', got '%s'", key)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for action request")
	}

	// Test Ctrl+C
	// We expect tea.Quit command sequence
	_, cmd := tui.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Expected command from Ctrl+C, got nil")
	}

	// In Client Mode, ExitChan should NOT be closed by Ctrl+C
	select {
	case <-config.ExitChan:
		t.Error("ExitChan should not be closed by Ctrl+C in Client Mode")
	default:
		// Good
	}

	// Close explicitly for cleanup
	close(config.ExitChan)
}
