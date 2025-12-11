package devtui

import (
	"context"
	"time"

	. "github.com/tinywasm/fmt"
)

// Internal async state management (not exported)
type internalAsyncState struct {
	isRunning   bool
	operationID string
	cancel      context.CancelFunc
	startTime   time.Time
}

// Field represents a field in the TUI with a handler-based approach
// field represents a field in the TUI with async capabilities
type field struct {
	// NEW: Handler-based approach with anyHandler (replaces fieldHandler)
	handler   *anyHandler // Handles all field behavior
	parentTab *tabSection // Direct reference to parent for message routing

	// NEW: Internal async state
	asyncState *internalAsyncState

	// UNCHANGED: Existing internal fields
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
		// Follow EXACT same MessageTracker logic as executeChangeSyncWithTracking
		var operationID string
		if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.id != nil {
			// Check if handler has existing operationID to reuse (for updates)
			if existingID := f.handler.GetLastOperationID(); existingID != "" {
				operationID = existingID
			} else {
				// Generate new ID for new operations
				operationID = f.parentTab.tui.id.GetNewID()
			}
		}

		// Create progress callback that follows MessageTracker logic

		// Use helper to safely collect progress messages
		progressChan, done := f.collectProgressMessages(func(msg string) {
			// Process message immediately
			f.sendMessage(msg)
		})

		// Execute handler
		f.handler.Change("", progressChan)
		close(progressChan)
		<-done

		// Set operation ID on handler for tracking (same as executeChangeSyncWithTracking)
		f.handler.SetLastOperationID(operationID)
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

// collectProgressMessages creates a goroutine that safely collects messages from a progress channel.
// Returns the progress channel (for handler to send to) and done channel (to wait for completion).
// The caller must close progressChan after handler completes, then wait on <-done.
// This helper unifies the pattern and ensures 'done' is always closed via defer, preventing panics.
func (f *field) collectProgressMessages(processMessage func(string)) (progressChan chan string, done chan struct{}) {
	progressChan = make(chan string, 10)
	done = make(chan struct{})

	go func() {
		defer close(done) // Always close done, even with early returns
		for msg := range progressChan {
			if processMessage != nil {
				processMessage(msg)
			}
		}
	}()

	return progressChan, done
}

// sendMessage sends a message through parent tab with automatic type detection
// REFACTORIZADO: Reemplaza sendProgressMessage, sendErrorMessage, sendSuccessMessage
func (f *field) sendMessage(msgs ...any) {
	if f.parentTab == nil || f.parentTab.tui == nil || len(msgs) == 0 {
		return
	}

	// Get operation ID from async state or use empty string
	var operationID string
	if f.asyncState != nil && f.asyncState.operationID != "" {
		operationID = f.asyncState.operationID
	}

	// Get handler name
	handlerName := ""
	handlerColor := ""
	if f.handler != nil {
		handlerName = f.handler.Name()
		handlerColor = f.handler.handlerColor // NEW: Get handler color
	}

	// NEW: If handler has Content() method, refresh display instead of creating messages
	if f.hasContentMethod() {
		f.parentTab.tui.updateViewport()
		return
	}

	// Convert and send message with automatic type detection
	message, msgType := Translate(msgs...).StringType()
	f.parentTab.tui.sendMessageWithHandler(message, msgType, f.parentTab, handlerName, operationID, handlerColor)
}

// executeAsyncChange executes the handler's Change method asynchronously
func (f *field) executeAsyncChange(valueToSave any) {
	if f.handler == nil || f.asyncState == nil {
		return
	}

	// In test mode, execute synchronously for predictable test behavior
	if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.isTestMode() {
		f.executeChangeSyncWithValue(valueToSave)
		return
	}

	// Create internal context with timeout from handler
	timeout := f.handler.Timeout()
	var ctx context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	f.asyncState.cancel = cancel
	f.asyncState.isRunning = true

	// Generate ONE operation ID for the entire async operation OR reuse existing one
	if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.id != nil {
		// Check if handler has existing operationID to reuse (for updates)
		if existingID := f.handler.GetLastOperationID(); existingID != "" {
			f.asyncState.operationID = existingID
		} else {
			// Generate new ID for new operations
			f.asyncState.operationID = f.parentTab.tui.id.GetNewID()
		}
	} else {
		// Log when id is nil for debugging
		if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.Logger != nil {
			f.parentTab.tui.Logger("Warning: Cannot generate operation ID, unixid not initialized")
		}
	}
	f.asyncState.startTime = time.Now()

	// Use the pre-captured value instead of getCurrentValue()
	currentValue := valueToSave

	// Execute user's Change method with context monitoring
	resultChan := make(chan struct {
		result string
		err    error
	}, 1)

	go func() {
		// Use helper to safely collect progress messages
		progressChan, done := f.collectProgressMessages(func(msg string) {
			if f.parentTab != nil {
				if f.hasContentMethod() {
					f.parentTab.tui.updateViewport()
					return
				}
				f.sendMessage(msg)
			}
		})

		// Ensure channel is closed when goroutine exits, even if context is cancelled
		// Use defer with panic recovery to prevent crashes
		defer func() {
			if r := recover(); r != nil {
				// Log the panic instead of crashing
				if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.Logger != nil {
					f.parentTab.tui.Logger("Internal error in handler goroutine:", r)
				}
			}
			// Safely close the channel
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Channel might already be closed, log but don't crash
						if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.Logger != nil {
							f.parentTab.tui.Logger("Channel close error (expected if timeout occurred):", r)
						}
					}
				}()
				close(progressChan)
			}()
			<-done
		}()

		f.handler.Change(currentValue.(string), progressChan)

		// Only send result if context wasn't cancelled
		select {
		case <-ctx.Done():
			// Context was cancelled, don't send result
			return
		default:
			result := f.handler.Value() // Obtener valor actualizado
			resultChan <- struct {
				result string
				err    error
			}{result, nil}
		}
	}()

	// Wait for completion or timeout
	select {
	case res := <-resultChan:
		// Operation completed normally
		f.asyncState.isRunning = false

		if res.err != nil {
			// Handler decides error message content
			f.sendMessage(res.err.Error())
		} else {
			switch f.handler.handlerType {
			case handlerTypeEdit:
				// NEW: If handler has Content() method, only refresh display
				if f.hasContentMethod() {
					f.parentTab.tui.updateViewport()
				} else {
					f.sendMessage(res.result)
				}
			case handlerTypeExecution:
				// Only send if handler explicitly implements Value()
				if _, ok := f.handler.origHandler.(interface{ Value() string }); ok {
					f.sendMessage(res.result)
				}
				// Other handler types: do not send success message
			}
		}

	case <-ctx.Done():
		// Operation timed out
		f.asyncState.isRunning = false

		if ctx.Err() == context.DeadlineExceeded {
			f.sendMessage(Fmt("Operation timed out after %v", timeout))
		} else {
			f.sendMessage("Operation was cancelled")
		}
	}

	cancel() // Clean up context
}

// executeChangeSyncWithValue executes the handler's Change method synchronously with pre-captured value
func (f *field) executeChangeSyncWithValue(valueToSave any) {
	if f.handler == nil {
		return
	}

	// In sync test mode, we don't generate operation IDs or send messages to avoid race conditions
	// Use the pre-captured value directly

	// Use helper to safely collect progress messages (discarding them in test mode)
	progressChan, done := f.collectProgressMessages(func(msg string) {
		// In sync test mode, we don't send messages to avoid race conditions
	})

	f.handler.Change(valueToSave.(string), progressChan)
	close(progressChan)
	<-done
	// In test mode, we don't send messages to UI to avoid race conditions
	// The test can verify the handler's internal state directly
}

// executeChangeSyncWithTracking executes the handler's Change method synchronously but maintains operation ID tracking
// This is specifically for testing operation ID reuse functionality
func (f *field) executeChangeSyncWithTracking(valueToSave any) {
	if f.handler == nil {
		return
	}

	// Generate or reuse operation ID like in async mode
	var operationID string
	if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.id != nil {
		// Check if handler has existing operationID to reuse (for updates)
		if existingID := f.handler.GetLastOperationID(); existingID != "" {
			operationID = existingID
		} else {
			// Generate new ID for new operations
			operationID = f.parentTab.tui.id.GetNewID()
		}
	}

	// Create progress callback that sends messages with operation tracking
	handlerName := f.handler.Name()
	handlerColor := f.handler.handlerColor // NEW: Get handler color

	// Use helper to safely collect progress messages
	progressChan, done := f.collectProgressMessages(func(msg string) {
		if f.parentTab != nil {
			// NEW: If handler has Content() method, refresh display instead of creating messages
			if f.hasContentMethod() {
				f.parentTab.tui.updateViewport()
				return
			}
			// For regular handlers, create timestamped messages with tracking
			_, msgType := Translate(msg).StringType()
			f.parentTab.tui.sendMessageWithHandler(msg, msgType, f.parentTab, handlerName, operationID, handlerColor)
		}
	})

	// Execute handler
	f.handler.Change(valueToSave.(string), progressChan)
	close(progressChan)
	<-done

	// Set operation ID on handler for tracking
	f.handler.SetLastOperationID(operationID)

	// Send success message (unless handler has Content() method)
	if f.parentTab != nil {
		// NEW: If handler has Content() method, only refresh display
		if f.hasContentMethod() {
			f.parentTab.tui.updateViewport()
		} else {
			// For regular handlers, send success message
			result := f.handler.Value()
			_, msgType := Translate(result).StringType()
			f.parentTab.tui.sendMessageWithHandler(result, msgType, f.parentTab, handlerName, operationID, handlerColor)
		}
	}
}

// handleEnter triggers async operation when user presses Enter
func (f *field) handleEnter() {
	if f.handler == nil {
		return
	}

	// NEW: Readonly fields don't respond to any keys
	if f.isDisplayOnly() {
		return
	}

	// Capture the current value BEFORE any state changes
	valueToSave := f.getCurrentValue()

	// In test mode, execute synchronously without goroutine
	if f.parentTab != nil && f.parentTab.tui != nil && f.parentTab.tui.isTestMode() {
		f.executeChangeSyncWithValue(valueToSave)
		return
	}

	// DevTUI handles async internally - user doesn't see this complexity
	go f.executeAsyncChange(valueToSave)
}
