package devtui

import (
	"time"

	"github.com/tinywasm/fmt"
)

// validateTabSection validates that the provided any is a valid *tabSection
// Returns the typed tabSection or panics with clear error message
func (t *DevTUI) validateTabSection(tab any, methodName string) *tabSection {
	if tab == nil {
		panic(fmt.Fmt(
			"DevTUI.%s: tabSection parameter is nil\n"+
				"Usage: tab := tui.NewTabSection(...); tui.%s(..., tab)",
			methodName, methodName))
	}

	ts, ok := tab.(*tabSection)
	if !ok {
		panic(fmt.Fmt(
			"DevTUI.%s: invalid tabSection type %T\n"+
				"Expected: value returned by tui.NewTabSection()\n"+
				"Got: %T\n"+
				"Usage: tab := tui.NewTabSection(...); tui.%s(..., tab)",
			methodName, tab, tab, methodName))
	}

	if ts.tui != t {
		panic(fmt.Fmt(
			"DevTUI.%s: tabSection belongs to different DevTUI instance\n"+
				"Each tabSection can only be used with the DevTUI instance that created it",
			methodName))
	}

	return ts
}

// AddHandler is the ONLY method to register handlers of any type.
// It accepts any handler interface and internally detects the type.
// Does NOT return anything - enforces complete decoupling.
//
// Supported handler interfaces (from interfaces.go):
//   - HandlerDisplay: Static/dynamic content display
//   - HandlerEdit: Interactive text input fields
//   - HandlerExecution: Action buttons
//   - HandlerInteractive: Combined display + interaction
//   - HandlerLogger: Basic line-by-line logging (via MessageTracker detection)
//
// Optional interfaces (detected automatically):
//   - MessageTracker: Enables message update tracking
//   - ShortcutProvider: Registers global keyboard shortcuts
//
// Parameters:
//   - handler: ANY handler implementing one of the supported interfaces
//   - timeout: Operation timeout (used for Edit/Execution/Interactive handlers, ignored for Display)
//   - color: Hex color for handler messages (e.g., "#1e40af", empty string for default)
//   - tabSection: The tab section returned by NewTabSection (as any for decoupling)
//
// Example:
//
//	tab := tui.NewTabSection("BUILD", "Compiler")
//	tui.AddHandler(myEditHandler, 2*time.Second, "#3b82f6", tab)
//	tui.AddHandler(myDisplayHandler, 0, "", tab)
func (t *DevTUI) AddHandler(handler any, timeout time.Duration, color string, tabSection any) {
	ts := t.validateTabSection(tabSection, "AddHandler")
	ts.addHandler(handler, timeout, color)
}

// addHandler - internal method (lowercase, private)
func (ts *tabSection) addHandler(handler any, timeout time.Duration, color string) {
	// Detect Loggable interface and inject logger
	if loggable, ok := handler.(Loggable); ok {
		ts.registerLoggableHandler(loggable, color)
	}

	// Type detection and routing
	switch h := handler.(type) {

	case HandlerDisplay:
		ts.registerDisplayHandler(h, color)

	case HandlerInteractive:
		ts.registerInteractiveHandler(h, timeout, color)

	case HandlerExecution:
		ts.registerExecutionHandler(h, timeout, color)

	case HandlerEdit:
		ts.registerEditHandler(h, timeout, color)

	default:
		// If not a known interface but is Loggable, it's valid (logging-only handler)
		if _, ok := handler.(Loggable); !ok {
			// Invalid handler type - log error
			if ts.tui != nil && ts.tui.Logger != nil {
				ts.tui.Logger("ERROR: Unknown handler type provided to AddHandler:", handler)
			}
		}
	}
}

// Internal registration methods (private)

func (ts *tabSection) registerDisplayHandler(handler HandlerDisplay, color string) {
	anyH := NewDisplayHandler(handler, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)
}

func (ts *tabSection) registerEditHandler(handler HandlerEdit, timeout time.Duration, color string) {
	anyH := NewEditHandler(handler, timeout, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)

	// Check for shortcut support
	ts.registerShortcutsIfSupported(handler, len(ts.fieldHandlers)-1)
}

func (ts *tabSection) registerExecutionHandler(handler HandlerExecution, timeout time.Duration, color string) {
	anyH := NewExecutionHandler(handler, timeout, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)
}

func (ts *tabSection) registerInteractiveHandler(handler HandlerInteractive, timeout time.Duration, color string) {
	anyH := NewInteractiveHandler(handler, timeout, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)
}

// Register in writing handlers list
// registerLoggableHandler sets up logging for handlers implementing Loggable
func (ts *tabSection) registerLoggableHandler(handler Loggable, color string) {
	handlerName := handler.Name()

	// Create anyHandler for tracking
	anyH := &anyHandler{
		handlerType:  handlerTypeLoggable,
		nameFunc:     handler.Name,
		handlerColor: color,
	}

	// Register in writing handlers
	ts.mu.Lock()
	ts.writingHandlers = append(ts.writingHandlers, anyH)
	ts.mu.Unlock()

	// Create logger function that DevTUI intercepts
	logger := func(message ...any) {
		if len(message) == 0 {
			return
		}

		// Format message
		var msg string
		if len(message) == 1 {
			if str, ok := message[0].(string); ok {
				msg = str
			} else {
				msg = fmt.Fmt("%v", message[0])
			}
		} else {
			msg = fmt.Fmt("%v", message[0])
			for _, m := range message[1:] {
				msg += " " + fmt.Fmt("%v", m)
			}
		}

		// Get message type and content
		messageStr, msgType := fmt.Translate(msg).StringType()

		// Send to DevTUI with handler tracking
		ts.tui.sendMessageWithHandler(messageStr, msgType, ts, handlerName, "", color)

		if msgType == fmt.Msg.Error {
			ts.tui.Logger(msg)
		}
	}

	// Inject logger into handler
	handler.SetLog(logger)
}

// registerShortcutsIfSupported checks if handler implements shortcut interface and registers shortcuts
func (ts *tabSection) registerShortcutsIfSupported(handler HandlerEdit, fieldIndex int) {
	// Check if handler implements shortcut interface
	if shortcutProvider, hasShortcuts := handler.(ShortcutProvider); hasShortcuts {
		shortcuts := shortcutProvider.Shortcuts()
		// shortcuts is an ordered slice of single-entry maps to preserve registration order
		for _, m := range shortcuts {
			for key, description := range m {
				entry := &ShortcutEntry{
					Key:         key,
					Description: description,
					TabIndex:    ts.index,
					FieldIndex:  fieldIndex,
					HandlerName: handler.Name(),
					Value:       key, // Use the key as the value by default
				}
				ts.tui.shortcutRegistry.Register(key, entry)
			}
		}
	}
}
