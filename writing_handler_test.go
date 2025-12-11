package devtui

import (
	"strings"
	"testing"
	"time"

	. "github.com/tinywasm/fmt"
)

// TestWriterHandlerRegistration tests the registration of writing handlers
func TestWriterHandlerRegistration(t *testing.T) {
	h := DefaultTUIForTest() // Empty TUI

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test writing handler registration")

	// Create a test writing handler using centralized handler

	// Register the handler and get its writer
	writer := h.AddLogger("TestWriter", false, "", tab)

	if writer == nil {
		t.Fatal("RegisterHandlerLogger should return a non-nil writer")
	}

	// Verify the handler was registered
	tabSection := tab.(*tabSection)
	if tabSection.writingHandlers == nil {
		t.Fatal("writingHandlers slice should be initialized")
	}

	if registeredHandler := tabSection.getWritingHandler("TestWriter"); registeredHandler == nil {
		t.Fatal("Handler should be registered in writingHandlers slice")
	}
}

// TestHandlerLoggerFunctionality tests the HandlerLogger wrapper
func TestHandlerLoggerFunctionality(t *testing.T) {
	h := DefaultTUIForTest() // Empty TUI

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test HandlerLogger functionality")

	// Register the handler and get its logger function (basic logger without tracking)
	logger := h.AddLogger("TestWriter", false, "", tab)

	// Call the logger function with a test message
	testMessage := "Test message from handler"
	logger(testMessage)

	// Verify handler was registered (basic writer doesn't have tracking)
	tabSection := tab.(*tabSection)
	if registeredHandler := tabSection.getWritingHandler("TestWriter"); registeredHandler == nil {
		t.Fatal("Handler should be registered in writingHandlers slice")
	}
}

// TestHandlerLoggerWithTracking tests the tracking functionality
func TestHandlerLoggerWithTracking(t *testing.T) {
	h := DefaultTUIForTest() // Empty TUI

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test HandlerLogger with tracking")

	// Register a writer with tracking enabled
	writer := h.AddLogger("TrackerWriter", true, "", tab)

	// Call the logger function with a test message
	testMessage := "Test tracking message"
	writer(testMessage)

	// Verify handler was registered with tracking capability
	tabSection := tab.(*tabSection)
	registeredHandler := tabSection.getWritingHandler("TrackerWriter")
	if registeredHandler == nil {
		t.Fatal("Handler should be registered in writingHandlers slice")
	}

	// Verify the handler has tracking capability by checking if it has operation ID methods
	if registeredHandler.GetLastOperationID() == "" {
		// This is expected initially - operation ID is set when messages are sent
		t.Log("Operation ID is initially empty, which is correct")
	}

	// Simulate setting an operation ID (this would happen during message processing)
	registeredHandler.SetLastOperationID("test-op-123")

	// Verify the operation ID was set
	if registeredHandler.GetLastOperationID() != "test-op-123" {
		t.Errorf("Expected operation ID 'test-op-123', got '%s'", registeredHandler.GetLastOperationID())
	}
}

// TestHandlerNameInMessages tests that handler names appear in formatted messages
func TestHandlerNameInMessages(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test handler name in messages")

	// Create a test writing handler

	// Register the handler and get its writer
	writer := h.AddLogger("Writer", false, "", tab)

	// Write a test message
	testMessage := "Test message with handler name"
	writer(testMessage)

	// Give some time for message processing
	time.Sleep(10 * time.Millisecond)

	// Check if the message contains the handler name
	// Note: We need to check the formatted message in the tab contents
	tabSection := tab.(*tabSection)
	tabSection.mu.RLock()
	defer tabSection.mu.RUnlock()

	if len(tabSection.tabContents) == 0 {
		t.Fatal("No messages found in tab contents")
	}

	lastContent := tabSection.tabContents[len(tabSection.tabContents)-1]
	expectedName := padHandlerName("Writer", HandlerNameWidth)
	if lastContent.handlerName != expectedName {
		t.Errorf("Message should have handler name '%s', got '%s'", expectedName, lastContent.handlerName)
	}

	if !strings.Contains(lastContent.Content, testMessage) {
		t.Errorf("Message content should contain test message: %s", lastContent.Content)
	}
}

// TestExplicitWriterRegistration tests that writers must be explicitly registered using NewLogger
func TestExplicitWriterRegistration(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test explicit writer registration")

	// Create a test field handler using centralized handler
	fieldHandler := NewTestEditableHandler("TestField", "test")

	// Add field using new API (does NOT auto-register for writing anymore)
	h.AddHandler(fieldHandler, 0, "", tab)

	// Verify the field handler was NOT auto-registered for writing
	tabSection := tab.(*tabSection)
	handlerName := fieldHandler.Name()
	if registeredHandler := tabSection.getWritingHandler(handlerName); registeredHandler != nil {
		t.Fatalf("Handler should NOT be auto-registered in writingHandlers slice with name '%s'", handlerName)
	}

	// Now explicitly register a writer with the same name
	writer := h.AddLogger(handlerName, false, "", tab)
	if writer == nil {
		t.Fatal("NewLogger should return a non-nil writer")
	}

	// Verify the writer was explicitly registered
	if registeredHandler := tabSection.getWritingHandler(handlerName); registeredHandler == nil {
		t.Fatalf("Writer should be explicitly registered in writingHandlers slice with name '%s'", handlerName)
	}
}

// TestOperationIDControl tests that handlers can control message updates vs new messages
func TestOperationIDControl(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test operation ID control")

	// Register a writer with tracking enabled for operation ID control
	writer := h.AddLogger("Writer", true, "", tab)

	// First write - should create new message
	writer("First message")
	time.Sleep(10 * time.Millisecond)

	// Second write - should potentially update existing message (with tracking enabled)
	writer("Updated message")
	time.Sleep(10 * time.Millisecond)

	// Verify the writer was registered with tracking capability
	tabSection := tab.(*tabSection)
	registeredHandler := tabSection.getWritingHandler("Writer")
	if registeredHandler == nil {
		t.Fatal("Handler should be registered in writingHandlers slice")
	}

	// Verify messages were created
	tabSection.mu.RLock()
	defer tabSection.mu.RUnlock()

	if len(tabSection.tabContents) < 1 {
		t.Fatalf("Expected at least 1 message, got %d", len(tabSection.tabContents))
	}

	// Check that the handler name is preserved in messages
	for _, content := range tabSection.tabContents {
		expectedName := padHandlerName("Writer", HandlerNameWidth)
		if content.handlerName != expectedName {
			t.Errorf("All messages should have handler name '%s', got '%s'", expectedName, content.handlerName)
		}
	}
}

// TestMultipleHandlersInSameTab tests multiple handlers writing to the same tab
func TestMultipleHandlersInSameTab(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test multiple handlers")

	// Create multiple test writing handlers

	// Register both handlers
	writer1 := h.AddLogger("W1", false, "", tab)
	writer2 := h.AddLogger("W2", false, "", tab)

	// Write messages from both handlers
	writer1("Message from Writer1")
	writer2("Message from Writer2")

	time.Sleep(10 * time.Millisecond)

	// Verify both handlers are registered
	tabSection := tab.(*tabSection)
	if len(tabSection.writingHandlers) != 2 {
		t.Errorf("Expected 2 registered handlers, got %d", len(tabSection.writingHandlers))
	}

	// Verify messages from both handlers are present
	tabSection.mu.RLock()
	defer tabSection.mu.RUnlock()

	var writer1Messages, writer2Messages int
	for _, content := range tabSection.tabContents {
		switch content.handlerName {
		case padHandlerName("W1", HandlerNameWidth):
			writer1Messages++
		case padHandlerName("W2", HandlerNameWidth):
			writer2Messages++
		}
	}

	if writer1Messages == 0 {
		t.Error("Should have messages from W1")
	}
	if writer2Messages == 0 {
		t.Error("Should have messages from W2")
	}
}

// TestMessageTypeDetection tests that message types are still detected correctly with handler names
func TestMessageTypeDetection(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test message type detection")

	// Create a test writing handler
	writer := h.AddLogger("TestWriter", false, "", tab)

	// Test different message types
	testCases := []struct {
		message      string
		expectedType MessageType
	}{
		{"Error occurred", Msg.Error},
		{"Success! Operation completed", Msg.Success},
		{"Warning: This is a warning", Msg.Warning},
		{"Info: This is information", Msg.Info},
	}

	tabSection := tab.(*tabSection)
	for _, tc := range testCases {
		writer(tc.message)
		time.Sleep(5 * time.Millisecond)

		// Check the last message
		tabSection.mu.RLock()
		if len(tabSection.tabContents) > 0 {
			lastMessage := tabSection.tabContents[len(tabSection.tabContents)-1]
			if lastMessage.Type != tc.expectedType {
				t.Errorf("Message '%s' should have type %v, got %v", tc.message, tc.expectedType, lastMessage.Type)
			}
		}
		tabSection.mu.RUnlock()
	}
}