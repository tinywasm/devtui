package devtui

import (
	"strings"
	"testing"
)

type DynamicHandler struct {
	name string
	log  func(...any)
}

func (d *DynamicHandler) Name() string                  { return d.name }
func (d *DynamicHandler) SetLog(f func(message ...any)) { d.log = f }
func (d *DynamicHandler) AlwaysShowAllLogs() bool       { return true }
func (d *DynamicHandler) WaitingForUser() bool          { return true }
func (d *DynamicHandler) Value() string                 { return "" }
func (d *DynamicHandler) Label() string                 { return "Dynamic" }
func (d *DynamicHandler) Change(v string)               {}

func TestDynamicHandlerName(t *testing.T) {
	tui := NewTUI(&TuiConfig{AppName: "Test"})
	tab := tui.NewTabSection("Test", "Test")

	handler := &DynamicHandler{name: "STEP 1"}
	// AddHandler will register it as handlerTypeInteractive if it implements HandlerInteractive
	tui.AddHandler(handler, 0, "", tab)

	// Log with first name
	handler.log("Message 1")

	// Change name as if moving to next step
	handler.name = "STEP 2"
	handler.log("Message 2")

	// Verify tab contents
	ts := tab.(*tabSection)
	if len(ts.tabContents) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(ts.tabContents))
	}

	for i, msg := range ts.tabContents {
		expectedName := "STEP 1"
		if i == 1 {
			expectedName = "STEP 2"
		}

		// 1. Verify RawHandlerName is correctly captured at log time
		if msg.RawHandlerName != expectedName {
			t.Errorf("Message %d: expected RawHandlerName %q, got %q", i, expectedName, msg.RawHandlerName)
		}

		// 2. Verify handlerType is correctly preserved as Interactive
		if msg.handlerType != handlerTypeInteractive {
			t.Errorf("Message %d: expected handlerTypeInteractive, got %v", i, msg.handlerType)
		}

		// 3. Verify formatting: Interactive handlers should show HH:MM:SS + Content (no handler name)
		formatted := tui.formatMessage(msg, false)
		if strings.Contains(formatted, expectedName) {
			t.Errorf("Message %d: formatted output incorrectly contains handler name: %q", i, formatted)
		}

		t.Logf("Formatted %d (%s): %q", i, msg.RawHandlerName, formatted)
	}
}
