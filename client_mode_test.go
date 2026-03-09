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
	authHeaderChan := make(chan string, 1)
	// Setup mock SSE server
	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderChan <- r.Header.Get("Authorization")
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
		ClientURL:  sseServer.URL + "/logs",
		APIKey:     "test-api-key",
		ExitChan:   make(chan bool),
	}
	tui := NewTUI(config)
	tui.SetTestMode(true) // Ensure deterministic behavior if applicable
	// Start SSE client manually (Init() does this normally, but tests don't call Init())
	go tui.startSSEClient(config.ClientURL)

	// Wait for auth header
	select {
	case auth := <-authHeaderChan:
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected auth header 'Bearer test-api-key', got '%s'", auth)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for auth header")
	}

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
		if r.URL.Path == "/mcp" {
			var body struct {
				Method string            `json:"method"`
				Params map[string]string `json:"params"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			if body.Method == "tinywasm/action" {
				actionReceived <- body.Params["key"]
			}
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

	// Test Ctrl+C: should send "stop" action and close ExitChan
	_, cmd := tui.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("Expected command from Ctrl+C, got nil")
	}

	// Verify "stop" action was sent
	select {
	case key := <-actionReceived:
		if key != "stop" {
			t.Errorf("Expected action key 'stop', got '%s'", key)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for action request")
	}

	// ExitChan should be closed by Ctrl+C
	select {
	case <-config.ExitChan:
		// Good - channel was closed
	default:
		t.Error("ExitChan should be closed by Ctrl+C in Client Mode")
	}
}

func TestTuiConfig_APIKey_StoredInDevTUI(t *testing.T) {
	apiKey := "secret-token-123"
	config := &TuiConfig{
		AppName: "TestApp",
		APIKey:  apiKey,
	}
	tui := NewTUI(config)

	if tui.apiKey != apiKey {
		t.Errorf("Expected apiKey %s, got %s", apiKey, tui.apiKey)
	}
}

func TestSSEConnect_NoAPIKey_NoAuthHeader(t *testing.T) {
	authHeaderChan := make(chan string, 1)
	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderChan <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		fmt.Fprintf(w, "data: {}\n\n")
		w.(http.Flusher).Flush()
	}))
	defer sseServer.Close()

	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  sseServer.URL + "/logs",
		APIKey:     "", // No API Key
		ExitChan:   make(chan bool),
	}
	tui := NewTUI(config)
	go tui.startSSEClient(config.ClientURL)
	defer close(config.ExitChan)

	select {
	case auth := <-authHeaderChan:
		if auth != "" {
			t.Errorf("Expected no Authorization header, got '%s'", auth)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for SSE request")
	}
}

func TestPostAction_SendsJSONRPCAction(t *testing.T) {
	type rpcRequest struct {
		Method string            `json:"method"`
		Params map[string]string `json:"params"`
	}
	requestChan := make(chan rpcRequest, 1)

	actionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body rpcRequest
		json.NewDecoder(r.Body).Decode(&body)
		requestChan <- body
		w.WriteHeader(200)
	}))
	defer actionServer.Close()

	tui := &DevTUI{
		TuiConfig: &TuiConfig{
			ClientURL: actionServer.URL + "/logs",
		},
	}
	client := tui.mcpClient()
	postAction(client, "ctrl+s", "save-value")

	select {
	case req := <-requestChan:
		if req.Method != "tinywasm/action" {
			t.Errorf("Expected method 'tinywasm/action', got '%s'", req.Method)
		}
		if req.Params["key"] != "ctrl+s" {
			t.Errorf("Expected key 'ctrl+s', got '%s'", req.Params["key"])
		}
		if req.Params["value"] != "save-value" {
			t.Errorf("Expected value 'save-value', got '%s'", req.Params["value"])
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for action request")
	}
}

func TestFetchState_CallsJSONRPCState(t *testing.T) {
	methodChan := make(chan string, 1)
	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		methodChan <- body.Method

		// Return empty state
		fmt.Fprintf(w, `{"jsonrpc":"2.0","result":[],"id":1}`)
	}))
	defer mcpServer.Close()

	tui := &DevTUI{
		TuiConfig: &TuiConfig{
			ClientURL: mcpServer.URL + "/logs",
		},
	}

	tui.fetchAndReconstructState()

	select {
	case method := <-methodChan:
		if method != "tinywasm/state" {
			t.Errorf("Expected method 'tinywasm/state', got '%s'", method)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for state request")
	}
}

func TestHandleLogEvent_StateRefreshSignal_FetchesState(t *testing.T) {
	methodChan := make(chan string, 1)
	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		methodChan <- body.Method
		fmt.Fprintf(w, `{"jsonrpc":"2.0","result":[],"id":1}`)
	}))
	defer mcpServer.Close()

	tui := &DevTUI{
		TuiConfig: &TuiConfig{
			ClientURL: mcpServer.URL + "/logs",
		},
		tabContentsChan: make(chan tabContent, 10),
	}

	// HandlerType: 0 should trigger refresh
	tui.handleLogEvent(`{"handler_type": 0}`)

	select {
	case method := <-methodChan:
		if method != "tinywasm/state" {
			t.Errorf("Expected method 'tinywasm/state', got '%s'", method)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for state request")
	}
}

func TestHandleLogEvent_NormalEntry_NoStateRefresh(t *testing.T) {
	methodChan := make(chan string, 1)
	mcpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		methodChan <- body.Method
		fmt.Fprintf(w, `{"jsonrpc":"2.0","result":[],"id":1}`)
	}))
	defer mcpServer.Close()

	tui := &DevTUI{
		TuiConfig: &TuiConfig{
			ClientURL: mcpServer.URL + "/logs",
		},
		tabContentsChan: make(chan tabContent, 10),
	}

	// HandlerType != 0 should NOT trigger refresh
	tui.handleLogEvent(`{"handler_type": 1, "tab_title": "SHORTCUTS"}`)

	select {
	case method := <-methodChan:
		t.Errorf("Did not expect any JSON-RPC calls, but got '%s'", method)
	case <-time.After(500 * time.Millisecond):
		// Good, no request made
	}
}
