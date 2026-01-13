package devtui

import (
	. "github.com/tinywasm/fmt"
)

// Field represents a field in the TUI with a handler-based approach
type field struct {
	// Handler-based approach with anyHandler (replaces fieldHandler)
	handler   *anyHandler // Handles all field behavior
	parentTab *tabSection // Direct reference to parent for message routing

	// Internal fields for editing
	tempEditValue string // use for edit
	index         int
	cursor        int // cursor position in text value
}

// setTempEditValueForTest permite modificar tempEditValue en tests
func (f *field) setTempEditValueForTest(val string) {
	f.tempEditValue = val
}

// setCursorForTest permite modificar el cursor en tests
func (f *field) setCursorForTest(cursor int) {
	f.cursor = cursor
}

// setFieldHandlers sets the field handlers slice (mainly for testing)
// Only for internal/test use
func (ts *tabSection) setFieldHandlers(handlers []*field) {
	ts.fieldHandlers = handlers
}

// addFields adds one or more field handlers to the section (private)
func (ts *tabSection) addFields(fields ...*field) {
	ts.fieldHandlers = append(ts.fieldHandlers, fields...)
}

func (f *field) Value() string {
	if f.handler != nil {
		return f.handler.Value()
	}
	return ""
}

// GetHandlerForTest returns the handler for testing purposes
func (f *field) getHandlerForTest() *anyHandler {
	return f.handler
}

func (f *field) editable() bool {
	if f.handler != nil {
		return f.handler.editable()
	}
	return false
}

// READONLY FIELD CONVENTION:
// - FieldHandler with Label() == "" (exactly empty string) indicates readonly/info display
// - Uses fieldReadOnlyStyle (highlight background + clear text)
// - No keyboard interaction allowed (no cursor, no Enter response)
// - Message content displayed without timestamp for cleaner visual
// - Navigation between fields works, but no interaction within readonly content
func (f *field) isDisplayOnly() bool {
	if f.handler == nil {
		return false
	}
	return f.handler.handlerType == handlerTypeDisplay
}

// NUEVO: Detección para execution con footer expandido
func (f *field) isExecutionHandler() bool {
	if f.handler == nil {
		return false
	}
	return f.handler.handlerType == handlerTypeExecution
}

// NUEVO: Detección para handlers que usan footer expandido (Display + Execution)
func (f *field) usesExpandedFooter() bool {
	return f.isDisplayOnly() || f.isExecutionHandler()
}

// NUEVO: Método para mostrar contenido en la sección principal - only Display handlers show content immediately
func (f *field) getDisplayContent() string {
	if f.handler != nil && f.handler.contentFunc != nil && f.isDisplayOnly() {
		return f.handler.contentFunc()
	}
	return ""
}

// NEW: Helper method to detect Content() capability - only Display handlers have Content()
func (f *field) hasContentMethod() bool {
	return f.handler != nil && f.handler.contentFunc != nil && f.isDisplayOnly()
}

func (f *field) isInteractiveHandler() bool {
	if f.handler == nil {
		return false
	}
	return f.handler.handlerType == handlerTypeInteractive
}

func (f *field) shouldAutoActivateEditMode() bool {
	if f.isInteractiveHandler() && f.handler != nil {
		return f.handler.WaitingForUser()
	}
	return false
}

// NEW: Trigger content display for interactive handlers via Change()
func (f *field) triggerContentDisplay() {
	if f.isInteractiveHandler() && f.handler != nil && !f.handler.WaitingForUser() {
		// Execute handler - messages flow through h.log()
		f.handler.Change("")
	}
}

// NUEVO: Método para footer expandido - Name() usa espacio de label + value
func (f *field) getExpandedFooterLabel() string {
	if f.usesExpandedFooter() && f.handler != nil {
		if f.isDisplayOnly() && f.handler.nameFunc != nil {
			// Display handlers show Name() in footer
			return f.handler.nameFunc()
		} else if f.isExecutionHandler() && f.handler.valueFunc != nil {
			// Execution handlers show Value() in footer for better UX
			return f.handler.valueFunc()
		}
	}
	return ""
}

func (f *field) setCursorAtEnd() {
	// Calculate cursor position based on rune count, not byte count
	if f.handler != nil {
		f.cursor = len([]rune(f.handler.Value()))
	}
}

// getCurrentValue returns the appropriate value for Change() method
func (f *field) getCurrentValue() any {
	if f.handler == nil {
		return ""
	}

	if f.handler.editable() {
		// For editable fields, return the edited text (tempEditValue or current value)
		// This matches current field behavior with tempEditValue
		// Check if we're in editing mode by looking at parent tab's edit state
		if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.editModeActivated {
			// In edit mode, always use tempEditValue (even if empty string)
			return f.tempEditValue
		}
		return f.handler.Value()
	} else {
		// For non-editable fields (action buttons), return the original value
		return f.handler.Value()
	}
}

// sendMessage sends a message through parent tab with automatic type detection
func (f *field) sendMessage(msgs ...any) {
	if f.parentTab == nil || f.parentTab.tui == nil || len(msgs) == 0 {
		return
	}

	// Get handler name and color
	handlerName := ""
	handlerColor := ""
	if f.handler != nil {
		handlerName = f.handler.Name()
		handlerColor = f.handler.handlerColor
	}

	// NEW: If handler has Content() method, refresh display instead of creating messages
	if f.hasContentMethod() {
		f.parentTab.tui.updateViewport()
		return
	}

	// trackingID is now the handlerName for automatic tracking
	trackingID := handlerName

	// Convert and send message with automatic type detection
	message, msgType := Translate(msgs...).StringType()
	hType := handlerTypeLoggable
	if f.handler != nil {
		hType = f.handler.handlerType
	}
	f.parentTab.tui.sendMessageWithHandler(message, msgType, f.parentTab, handlerName, trackingID, handlerColor, hType)
}

// handleEnter executes the handler's Change method synchronously.
// Logs are already async via tabContentsChan, so no goroutine needed here.
func (f *field) handleEnter() {
	if f.handler == nil {
		return
	}

	// Readonly fields don't respond to any keys
	if f.isDisplayOnly() {
		return
	}

	// Capture the current value BEFORE any state changes
	valueToSave := f.getCurrentValue()

	// Panic recovery for safety
	defer func() {
		if r := recover(); r != nil {
			if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.Logger != nil {
				f.parentTab.tui.Logger("Handler panic:", r)
			}
		}
	}()

	// Execute synchronously - logs flow through tabContentsChan (already async)
	f.handler.Change(valueToSave.(string))
}
