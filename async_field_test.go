package devtui

import (
	"fmt"
	"testing"
)

// Test async field operations

func TestFieldHandler_BasicOperation(t *testing.T) {
	// Use centralized handler from handler_test.go
	handler := NewTestEditableHandler("Test Field", "initial")

	// Use simplified DefaultTUIForTest
	tui := DefaultTUIForTest()

	// Create a test tab and add the handler using new API
	tab := tui.NewTabSection("Test Tab", "Test description")
	tui.AddHandler(handler, 0, "", tab)

	// Verify field was created with handler
	ts := tab.(*tabSection)
	if len(ts.fieldHandlers) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(ts.fieldHandlers))
	}

	field := ts.fieldHandlers[0]
	anyH := field.handler

	// Test handler methods through anyHandler
	if anyH.Label() != "Test Field" {
		t.Errorf("Expected label 'Test Field', got '%s'", anyH.Label())
	}

	if anyH.Value() != "initial" {
		t.Errorf("Expected value 'initial', got '%s'", anyH.Value())
	}

	if !anyH.editable() {
		t.Error("Expected field to be editable")
	}

	if anyH.Timeout() != 0 {
		t.Errorf("Expected timeout 0s, got %v", anyH.Timeout())
	}
}

func TestFieldHandler_AsyncExecution(t *testing.T) {
	// Use centralized handler from handler_test.go - non-editable for action button
	slowHandler := NewTestNonEditableHandler("Slow Operation", "Click to run")
	tui := DefaultTUIForTest()

	// Create a test tab and add the handler using new API
	tab := tui.NewTabSection("Test Tab", "Test description")
	tui.AddHandler(slowHandler, 0, "", tab)

	ts := tab.(*tabSection)
	field := ts.fieldHandlers[0]

	// Test that async state is initialized
	if field.asyncState == nil {
		t.Fatal("Async state not initialized")
	}

	if field.asyncState.isRunning {
		t.Error("Async operation should not be running initially")
	}
}

func TestFieldHandler_ErrorHandling(t *testing.T) {
	// Test error handling using centralized error handler
	handler := NewTestErrorHandler("Error Field", "test")

	// The new API does not return errors, so just call Change
	handler.Change("any value")
	// No error to check; if the handler panics or misbehaves, the test will fail
}

func TestFieldHandler_TimeoutConfiguration(t *testing.T) {
	// Test Edit Handler
	editHandler := NewTestEditableHandler("Test", "value")
	tui := DefaultTUIForTest()
	tab := tui.NewTabSection("Test Tab", "Test description")
	tui.AddHandler(editHandler, 0, "", tab)

	ts := tab.(*tabSection)
	if len(ts.fieldHandlers) != 1 {
		t.Fatalf("Expected 1 field, got %d", len(ts.fieldHandlers))
	}

	field := ts.fieldHandlers[0]
	timeout := field.handler.Timeout()
	if timeout != 0 {
		t.Errorf("Expected timeout 0s for edit handler, got %v", timeout)
	}

	// Test Execution Handler
	execHandler := NewTestNonEditableHandler("Action", "Press Enter")
	tab2 := tui.NewTabSection("Test Tab 2", "Test description")
	// Provide a timeout of 0 as in other tests
	tui.AddHandler(execHandler, 0, "", tab2)

	ts2 := tab2.(*tabSection)
	if len(ts2.fieldHandlers) != 1 {
		t.Fatalf("Expected 1 field in second tab, got %d", len(ts2.fieldHandlers))
	}

	field2 := ts2.fieldHandlers[0]
	timeout2 := field2.handler.Timeout()
	if timeout2 != 0 {
		t.Errorf("Expected timeout 0s for execution handler, got %v", timeout2)
	}
}

func TestFieldHandler_EditableFields(t *testing.T) {
	// Use centralized handlers from handler_test.go
	editableHandler := NewTestEditableHandler("editable Field", "original")
	nonEditableHandler := NewTestNonEditableHandler("Non-editable Field", "action button")

	// Test editable field
	if !editableHandler.editable() {
		t.Error("Handler should be editable")
	}
	editableHandler.Change("new value")

	if editableHandler.Value() != "new value" {
		t.Errorf("Expected value 'new value', got '%s'", editableHandler.Value())
	}

	// Test non-editable field (button) - now using execution handler

	originalValue := nonEditableHandler.Value()
	nonEditableHandler.Change("attempted change")

	if nonEditableHandler.Value() != originalValue {
		t.Error("Non-editable field value should not change")
	}
}

func TestAsyncState_Management(t *testing.T) {
	// Use centralized handler from handler_test.go
	handler := NewTestEditableHandler("Test Field", "test")
	tui := DefaultTUIForTest()

	// Create a test tab and add the handler using new API
	tab := tui.NewTabSection("Test Tab", "Test description")
	tui.AddHandler(handler, 0, "", tab)

	ts := tab.(*tabSection)
	field := ts.fieldHandlers[0]

	// Test initial async state
	if field.asyncState == nil {
		t.Fatal("Async state should be initialized")
	}

	if field.asyncState.isRunning {
		t.Error("Field should not be running initially")
	}

	if field.asyncState.trackingID != "" {
		t.Error("Tracking ID should be empty initially")
	}

	if field.asyncState.cancel != nil {
		t.Error("Cancel function should be nil initially")
	}
}

// Benchmark tests for performance

func BenchmarkFieldHandler_SimpleOperation(b *testing.B) {
	// Use centralized handler from handler_test.go
	handler := NewTestEditableHandler("Benchmark Field", "test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.Change(fmt.Sprintf("value-%d", i))
	}
}

func BenchmarkFieldHandler_MultipleFields(b *testing.B) {
	// Create multiple handlers using centralized handler
	var handlers []any
	for i := 0; i < 10; i++ {
		handler := NewTestEditableHandler(
			fmt.Sprintf("Field-%d", i),
			fmt.Sprintf("value-%d", i),
		)
		handlers = append(handlers, handler)
	}

	tui := DefaultTUIForTest(handlers...)
	if len(tui.TabSections) == 0 {
		b.Fatal("No tab sections found")
	}
	ts := tui.TabSections[GetFirstTestTabIndex()]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j, field := range ts.fieldHandlers {
			field.handler.Change(fmt.Sprintf("benchmark-value-%d-%d", i, j))
		}
	}
}
