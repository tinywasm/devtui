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

	case HandlerLogger:
		// Logger detection: check for MessageTracker to determine tracking capability
		_, hasTracking := handler.(MessageTracker)
		ts.registerLoggerHandler(h, color, hasTracking)

	default:
		// Invalid handler type - log error or panic
		if ts.tui != nil && ts.tui.Logger != nil {
			ts.tui.Logger("ERROR: Unknown handler type provided to AddHandler:", handler)
		}
	}
}

// AddLogger creates a logger function with the given name and tracking capability.
// enableTracking: true = can update existing lines, false = always creates new lines
//
// Parameters:
//   - name: Logger identifier for message display
//   - enableTracking: Enable message update tracking (vs always new lines)
//   - color: Hex color for logger messages (e.g., "#1e40af", empty string for default)
//   - tabSection: The tab section returned by NewTabSection (as any for decoupling)
//
// Returns:
//   - Variadic logging function: log("message", values...)
//
// Example:
//
//	tab := tui.NewTabSection("BUILD", "Compiler")
//	log := tui.AddLogger("BuildProcess", true, "#1e40af", tab)
//	log("Starting build...")
//	log("Compiling", 42, "files")
func (t *DevTUI) AddLogger(name string, enableTracking bool, color string, tabSection any) func(message ...any) {
	ts := t.validateTabSection(tabSection, "AddLogger")
	return ts.addLogger(name, enableTracking, color)
}

// addLogger - internal method (lowercase, private)
func (ts *tabSection) addLogger(name string, enableTracking bool, color string) func(message ...any) {
	if enableTracking {
		handler := &simpleWriterTrackerHandler{name: name}
		return ts.registerLoggerFunc(handler, color)
	} else {
		handler := &simpleWriterHandler{name: name}
		return ts.registerLoggerFunc(handler, color)
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
	var tracker MessageTracker
	if t, ok := handler.(MessageTracker); ok {
		tracker = t
	}

	anyH := NewEditHandler(handler, timeout, tracker, color)
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
	var tracker MessageTracker
	if t, ok := handler.(MessageTracker); ok {
		tracker = t
	}

	anyH := NewInteractiveHandler(handler, timeout, tracker, color)
	f := &field{
		handler:    anyH,
		parentTab:  ts,
		asyncState: &internalAsyncState{},
	}
	ts.addFields(f)
}

func (ts *tabSection) registerLoggerHandler(handler HandlerLogger, color string, hasTracking bool) {
	var anyH *anyHandler

	if hasTracking {
		// Handler implements MessageTracker
		if tracker, ok := handler.(interface {
			Name() string
			GetLastOperationID() string
			SetLastOperationID(string)
		}); ok {
			anyH = NewWriterTrackerHandler(tracker, color)
		} else {
			// This should not happen if hasTracking is true, but as a fallback:
			anyH = NewWriterHandler(handler, color)
		}
	} else {
		// Basic logger without tracking
		anyH = NewWriterHandler(handler, color)
	}

	// Register in writing handlers list
	ts.mu.Lock()
	ts.writingHandlers = append(ts.writingHandlers, anyH)
	ts.mu.Unlock()
}

// Internal simple handler implementations
type simpleWriterHandler struct {
	name string
}

func (w *simpleWriterHandler) Name() string {
	return w.name
}

type simpleWriterTrackerHandler struct {
	name            string
	lastOperationID string
}

func (w *simpleWriterTrackerHandler) Name() string {
	return w.name
}

func (w *simpleWriterTrackerHandler) GetLastOperationID() string {
	return w.lastOperationID
}

func (w *simpleWriterTrackerHandler) SetLastOperationID(id string) {
	w.lastOperationID = id
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
