package devtui

import (
	"github.com/tinywasm/fmt"
)

// validateTabSection validates that the provided any is a valid *tabSection
// Returns the typed tabSection or panics with clear error message
func (t *DevTUI) validateTabSection(tab any, methodName string) *tabSection {
	if tab == nil {
		panic(fmt.Sprintf(
			"DevTUI.%s: tabSection parameter is nil\n"+
				"Usage: tab := tui.NewTabSection(...); tui.%s(..., tab)",
			methodName, methodName))
	}

	ts, ok := tab.(*tabSection)
	if !ok {
		panic(fmt.Sprintf(
			"DevTUI.%s: invalid tabSection type %T\n"+
				"Expected: value returned by tui.NewTabSection()\n"+
				"Got: %T\n"+
				"Usage: tab := tui.NewTabSection(...); tui.%s(..., tab)",
			methodName, tab, tab, methodName))
	}

	if ts.tui != t {
		panic(fmt.Sprintf(
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
//	tui.AddHandler(myEditHandler, "#3b82f6", tab)
//	tui.AddHandler(myDisplayHandler, "", tab)
func (t *DevTUI) AddHandler(handler any, color string, tabSection any) {
	ts := t.validateTabSection(tabSection, "AddHandler")
	ts.addHandler(handler, color)
}

// addHandler - internal method (lowercase, private)
func (ts *tabSection) addHandler(handler any, color string) {
	// Detect Loggable interface and inject logger
	if loggable, ok := handler.(Loggable); ok {
		ts.registerLoggableHandler(loggable, color)
	}

	// Type detection and routing
	switch h := handler.(type) {

	case HandlerDisplay:
		ts.registerDisplayHandler(h, color)

	case HandlerInteractive:
		ts.registerInteractiveHandler(h, color)

	case HandlerExecution:
		ts.registerExecutionHandler(h, color)

	case HandlerEdit:
		ts.registerEditHandler(h, color)

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
		handler:   anyH,
		parentTab: ts,
	}
	ts.addFields(f)
}

func (ts *tabSection) registerEditHandler(handler HandlerEdit, color string) {
	anyH := NewEditHandler(handler, color)
	f := &field{
		handler:   anyH,
		parentTab: ts,
	}
	ts.addFields(f)

	// Check for shortcut support
	ts.registerShortcutsIfSupported(handler, len(ts.fieldHandlers)-1)
}

func (ts *tabSection) registerExecutionHandler(handler HandlerExecution, color string) {
	anyH := NewExecutionHandler(handler, color)
	f := &field{
		handler:   anyH,
		parentTab: ts,
	}
	ts.addFields(f)
}

func (ts *tabSection) registerInteractiveHandler(handler HandlerInteractive, color string) {
	anyH := NewInteractiveHandler(handler, color)
	f := &field{
		handler:   anyH,
		parentTab: ts,
	}
	ts.addFields(f)
}

// Register in writing handlers list
// registerLoggableHandler sets up logging for handlers implementing Loggable
func (ts *tabSection) registerLoggableHandler(handler Loggable, color string) {
	nameFunc := handler.Name

	// Detect handler type for specialized formatting
	hType := handlerTypeLoggable
	if _, ok := handler.(HandlerInteractive); ok {
		hType = handlerTypeInteractive
	} else if _, ok := handler.(HandlerDisplay); ok {
		hType = handlerTypeDisplay
	}

	// Detect streaming capability
	showAll := false
	if streamer, ok := handler.(StreamingLoggable); ok {
		showAll = streamer.AlwaysShowAllLogs()
	}

	// NEW: Override showAll if strictly in Debug mode from config
	// This forces all logs to be displayed without collapsing
	if ts.tui.Debug {
		showAll = true
	}

	// Create anyHandler for tracking
	anyH := &anyHandler{
		handlerType:  hType, // Use detected type
		nameFunc:     handler.Name,
		handlerColor: color,
		origHandler:  handler, // Store original handler for TabAware support
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
				msg = fmt.Sprintf("%v", message[0])
			}
		} else {
			msg = fmt.Sprintf("%v", message[0])
			for _, m := range message[1:] {
				msg += " " + fmt.Sprintf("%v", m)
			}
		}

		// Handle LogOpen/LogClose prefixes
		isOpening := false
		isClosing := false
		cleanMsg := msg

		if len(msg) >= 4 {
			if msg[:4] == LogOpen {
				isOpening = true
				cleanMsg = msg[4:]
			} else if msg[:4] == LogClose {
				isClosing = true
				cleanMsg = msg[4:]
			}
		}

		// Get message type and content
		messageStr, msgType := fmt.Translate(cleanMsg).StringType()

		// Get CURRENT name for dynamic tracking
		currentName := nameFunc()

		// Tracking logic:
		// If streaming (showAll) and not opening/closing -> no trackingID (always new line)
		// If not streaming -> always use currentName as trackingID
		// If opening/closing -> use currentName as trackingID (grouped)
		trackingID := ""
		if !showAll || isOpening || isClosing {
			trackingID = currentName
		}

		// Send to DevTUI
		ts.tui.sendMessageWithHandler(messageStr, msgType, ts, currentName, trackingID, color, hType)

		// Handle animation
		if isOpening {
			ts.startAnimation(currentName, messageStr, msgType, color)
		} else if isClosing {
			ts.stopAnimation(currentName)
		} else if trackingID == "" {
			// Regular streaming message: stop any pending animation
			ts.stopAnimation(currentName)
		}

		if msgType == fmt.Msg.Error || msgType == fmt.Msg.Debug {
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
