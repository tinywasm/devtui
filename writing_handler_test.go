package devtui

import (
	"strings"
	"testing"
	"time"

	. "github.com/tinywasm/fmt"
)

// testLoggable is a helper for testing Loggable handlers
type testLoggable struct {
	name    string
	logFunc func(message ...any)
}

func (t *testLoggable) Name() string { return t.name }
func (t *testLoggable) SetLog(f func(message ...any)) {
	t.logFunc = f
}

// TestWriterHandlerRegistration tests the registration of writing handlers
func TestWriterHandlerRegistration(t *testing.T) {
	h := DefaultTUIForTest() // Empty TUI

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test writing handler registration")

	// Create a test loggable handler
	loggable := &testLoggable{name: "TestWriter"}

	// Register via AddHandler
	h.AddHandler(loggable, "", tab)

	if loggable.logFunc == nil {
		t.Fatal("AddHandler should inject logger into Loggable handler")
	}

	// Verify the handler was registered
	tabSection := tab.(*tabSection)
	if len(tabSection.writingHandlers) == 0 {
		t.Fatal("writingHandlers slice should be populated")
	}
}

// TestHandlerLoggerFunctionality tests the Loggable handler injection
func TestHandlerLoggerFunctionality(t *testing.T) {
	h := DefaultTUIForTest() // Empty TUI

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test Loggable functionality")

	// Register
	loggable := &testLoggable{name: "TestWriter"}
	h.AddHandler(loggable, "", tab)

	// Call it
	testMessage := "Test message from handler"
	loggable.logFunc(testMessage)

	// Verify
	tabSection := tab.(*tabSection)
	tabSection.mu.RLock()
	defer tabSection.mu.RUnlock()

	if len(tabSection.tabContents) == 0 {
		t.Fatal("No messages found in tab contents")
	}
}

// TestAutomaticTracking tests that multiple logs from same handler update previous one
func TestAutomaticTracking(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test automatic tracking")

	// Register
	loggable := &testLoggable{name: "TrackerWriter"}
	h.AddHandler(loggable, "", tab)

	// Write messages - should update
	loggable.logFunc("First")
	loggable.logFunc("Second") // Should UPDATE "First"

	// Verify messages
	tabSection := tab.(*tabSection)
	tabSection.mu.RLock()
	defer tabSection.mu.RUnlock()

	// In the clean view (tabContents), there should only be one message for this handler
	if len(tabSection.tabContents) != 1 {
		t.Errorf("Expected 1 message (due to update), got %d", len(tabSection.tabContents))
	}

	if !strings.Contains(tabSection.tabContents[0].Content, "Second") {
		t.Errorf("Expected message to be 'Second', got '%s'", tabSection.tabContents[0].Content)
	}
}

// TestMultipleHandlersInSameTab tests multiple handlers writing to the same tab
func TestMultipleHandlersInSameTab(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test multiple handlers")

	// Register both handlers
	l1 := &testLoggable{name: "W1"}
	l2 := &testLoggable{name: "W2"}
	h.AddHandler(l1, "", tab)
	h.AddHandler(l2, "", tab)

	// Write messages from both handlers
	l1.logFunc("From W1")
	l2.logFunc("From W2")

	time.Sleep(10 * time.Millisecond)

	// Verify both handlers are registered
	tabSection := tab.(*tabSection)
	if len(tabSection.writingHandlers) != 2 {
		t.Errorf("Expected 2 registered handlers, got %d", len(tabSection.writingHandlers))
	}

	// Verify messages from both handlers are present
	tabSection.mu.RLock()
	defer tabSection.mu.RUnlock()

	if len(tabSection.tabContents) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(tabSection.tabContents))
	}
}

// TestMessageTypeDetection tests that message types are still detected correctly
func TestMessageTypeDetection(t *testing.T) {
	h := DefaultTUIForTest()

	// Create a new tab for testing
	tab := h.NewTabSection("WritingTest", "Test message type detection")

	// Register
	loggable := &testLoggable{name: "TestWriter"}
	h.AddHandler(loggable, "", tab)

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
		loggable.logFunc(tc.message)
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
