package devtui

import (
	"testing"
	"time"
)

// Test handlers implementing the new interfaces

type testDisplayHandler struct{}

func (h *testDisplayHandler) Name() string    { return "Test Display Handler" }
func (h *testDisplayHandler) Content() string { return "This is display content" }

type testEditHandler struct {
	value string
}

func (h *testEditHandler) Name() string  { return "TestEdit" }
func (h *testEditHandler) Label() string { return "Test Edit" }
func (h *testEditHandler) Value() string { return h.value }
func (h *testEditHandler) Change(newValue string, progress chan<- string) {
	h.value = newValue
	if progress != nil {
		progress <- "Changed"
	}
}

type testRunHandler struct{}

func (h *testRunHandler) Name() string  { return "TestRun" }
func (h *testRunHandler) Label() string { return "Test Run" }
func (h *testRunHandler) Execute(progress chan<- string) {
	if progress != nil {
		progress <- "Operation completed"
	}
}

type testLoggableHandler struct {
	name    string
	logFunc func(message ...any)
}

func (h *testLoggableHandler) Name() string { return h.name }
func (h *testLoggableHandler) SetLog(f func(message ...any)) {
	h.logFunc = f
}

func TestNewAPIHandlers(t *testing.T) {
	// Create TUI
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "Test New API",
		ExitChan: exitChan,
	})

	// Create tab section
	tab := tui.NewTabSection("Test", "Testing new API")

	// Test HandlerDisplay registration
	tui.AddHandler(&testDisplayHandler{}, 0, "", tab)

	// Test HandlerEdit registration with and without timeout
	tui.AddHandler(&testEditHandler{value: "initial"}, 0, "", tab)           // Sync
	tui.AddHandler(&testEditHandler{value: "async"}, 5*time.Second, "", tab) // Async

	// Test HandlerExecution registration with and without timeout
	tui.AddHandler(&testRunHandler{}, 0, "", tab)              // Sync
	tui.AddHandler(&testRunHandler{}, 10*time.Second, "", tab) // Async

	// Test Loggable registration
	l1 := &testLoggableHandler{name: "Log1"}
	l2 := &testLoggableHandler{name: "Log2"}
	tui.AddHandler(l1, 0, "", tab)
	tui.AddHandler(l2, 0, "", tab)

	// Verify field count (5 fields registered)
	tabSection := tab.(*tabSection)
	if len(tabSection.fieldHandlers) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(tabSection.fieldHandlers))
	}

	// Test field types
	fields := tabSection.fieldHandlers

	// First field should be HandlerDisplay (read-only)
	if !fields[0].isDisplayOnly() {
		t.Error("First field should be display-only")
	}

	// Second and third fields should be HandlerEdit (editable)
	if !fields[1].editable() {
		t.Error("Second field should be editable")
	}
	if !fields[2].editable() {
		t.Error("Third field should be editable")
	}

	// Fourth and fifth fields should be HandlerExecution (not editable, but not display-only)
	if fields[3].editable() {
		t.Error("Fourth field should not be editable")
	}
	if fields[3].isDisplayOnly() {
		t.Error("Fourth field should not be display-only")
	}

	// Verify Loggable injection
	if l1.logFunc == nil {
		t.Error("Log1 should have logger injected")
	}
	if l2.logFunc == nil {
		t.Error("Log2 should have logger injected")
	}

	// Test calling log functions
	l1.logFunc("test message")
	l2.logFunc("tracked message")

	// Verify writing handlers were registered
	if len(tabSection.writingHandlers) != 2 {
		t.Errorf("Expected 2 writing handlers, got %d", len(tabSection.writingHandlers))
	}

	close(exitChan)
}
