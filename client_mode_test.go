package devtui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSEClient_Authentication(t *testing.T) {
	authHeaderChan := make(chan string, 1)

	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderChan <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Send a dummy event to keep connection open briefly
		w.Write([]byte("event: log\ndata: {}\n\n"))
	}))
	defer sseServer.Close()

	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  sseServer.URL + "/logs",
		APIKey:     "test-api-key",
	}
	tui := NewTUI(config)
	tui.NewTabSection("SHORTCUTS", "Mock Shortcuts")
	tui.SetTestMode(true) // Ensure deterministic behavior if applicable

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start SSE client manually
	tui.sseWg.Add(1)
	go tui.startSSEClient(config.ClientURL, ctx)

	// Wait for auth header
	select {
	case auth := <-authHeaderChan:
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", auth)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for SSE connection with Auth header")
	}
}

func TestSSEClient_Reconnection(t *testing.T) {
	connectionCount := 0
	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionCount++
		if connectionCount == 1 {
			// First connection: close immediately
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	}))
	defer sseServer.Close()

	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  sseServer.URL + "/logs",
	}
	tui := NewTUI(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tui.sseWg.Add(1)
	go tui.startSSEClient(config.ClientURL, ctx)

	// Wait for at least two connection attempts
	timeout := time.After(3 * time.Second)
	for connectionCount < 2 {
		select {
		case <-timeout:
			t.Fatalf("Timed out waiting for SSE reconnection, count: %d", connectionCount)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestSSEClient_StateRefresh(t *testing.T) {
	// This test verifies that HandlerType 0 triggers fetchAndReconstructState
	// We'll mock the state refresh by checking if it attempts to connect to the daemon
	refreshTriggered := make(chan bool, 1)

	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/logs" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			// Send StateRefresh signal
			w.Write([]byte("event: log\ndata: {\"handler_type\": 0}\n\n"))
			return
		}
		if r.URL.Path == "/mcp" {
			refreshTriggered <- true
		}
	}))
	defer sseServer.Close()

	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  sseServer.URL + "/logs",
	}
	tui := NewTUI(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tui.sseWg.Add(1)
	go tui.startSSEClient(config.ClientURL, ctx)

	select {
	case <-refreshTriggered:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for state refresh trigger")
	}
}

func TestSSEClient_LogEventProcessing(t *testing.T) {
	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  "http://localhost:1234/logs",
	}
	tui := NewTUI(config)
	tab := tui.NewTabSection("TEST", "Test Tab")
	section := tab.(*tabSection)

	eventData := tabContentDTO{
		Id:          "test-id",
		Timestamp:   "2023-01-01 12:00:00",
		Content:     "test log message",
		Type:        1, // Msg.Info
		TabTitle:    "TEST",
		HandlerName: "TestHandler",
		HandlerType: handlerTypeLoggable,
	}
	data, _ := json.Marshal(eventData)

	tui.handleLogEvent(string(data))

	// Verify section contents
	section.mu.RLock()
	defer section.mu.RUnlock()
	if len(section.tabContents) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(section.tabContents))
	} else if section.tabContents[0].Content != "test log message" {
		t.Errorf("Expected content 'test log message', got '%s'", section.tabContents[0].Content)
	}
}

func TestSSEClient_NoAPIKey(t *testing.T) {
	authHeaderChan := make(chan string, 1)

	sseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeaderChan <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	}))
	defer sseServer.Close()

	config := &TuiConfig{
		ClientMode: true,
		ClientURL:  sseServer.URL + "/logs",
		APIKey:     "", // No API Key
	}
	tui := NewTUI(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tui.sseWg.Add(1)
	go tui.startSSEClient(config.ClientURL, ctx)

	select {
	case auth := <-authHeaderChan:
		if auth != "" {
			t.Errorf("Expected NO Authorization header, got '%s'", auth)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for SSE connection")
	}
}

func TestTuiConfig_APIKey_StoredInDevTUI(t *testing.T) {
	config := &TuiConfig{
		AppName: "TestApp",
		APIKey:  "secret-key",
	}
	tui := NewTUI(config)

	if tui.apiKey != "secret-key" {
		t.Errorf("Expected apiKey 'secret-key', got '%s'", tui.apiKey)
	}
}

func TestDevTUI_fetchAndReconstructState_Stripping(t *testing.T) {
	tui := &DevTUI{
		TuiConfig: &TuiConfig{
			ClientURL: "http://localhost:3030/logs",
		},
	}
	baseURL := tui.actionBaseURL()
	expected := "http://localhost:3030"
	if baseURL != expected {
		t.Errorf("Expected base URL %s, got %s", expected, baseURL)
	}
}

func TestDevTUI_clearRemoteHandlers(t *testing.T) {
	tui := &DevTUI{
		TuiConfig: &TuiConfig{},
	}
	section := &tabSection{Title: "Test"}
	tui.TabSections = []*tabSection{section}

	section.FieldHandlers = []*field{
		{isRemote: false},
		{isRemote: true},
		{isRemote: false},
		{isRemote: true},
	}

	tui.clearRemoteHandlers()

	if len(section.FieldHandlers) != 2 {
		t.Errorf("Expected 2 handlers after clearing, got %d", len(section.FieldHandlers))
	}
	for _, f := range section.FieldHandlers {
		if f.isRemote {
			t.Errorf("Found remote handler after clearing")
		}
	}
}

func TestDevTUI_reconstructRemoteHandlers(t *testing.T) {
	tui := &DevTUI{
		TuiConfig: &TuiConfig{
			ClientURL: "http://localhost:3030/logs",
		},
	}
	section := &tabSection{Title: "App"}
	tui.TabSections = []*tabSection{section}

	entries := []StateEntry{
		{TabTitle: "App", HandlerName: "Srv", HandlerType: 1}, // Local
		{TabTitle: "Other", HandlerName: "Log", HandlerType: 1},
	}

	tui.reconstructRemoteHandlers(entries)

	// entries[1] should be ignored (tab title mismatch)
	if len(section.FieldHandlers) != 1 {
		t.Errorf("Expected 1 handler added, got %d", len(section.FieldHandlers))
	}
}
