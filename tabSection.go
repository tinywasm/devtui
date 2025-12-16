package devtui

import (
	"sync"
	"time"

	. "github.com/tinywasm/fmt"
)

// Interface for handling tab field sectionFields

// tabContent imprime contenido en la tui con id único
type tabContent struct {
	Id         string // unix number id eg: "1234567890" - INMUTABLE
	Timestamp  string // unix nano timestamp - MUTABLE (se actualiza en cada cambio)
	Content    string
	Type       MessageType
	tabSection *tabSection

	// NEW: Async fields (always present, nil when not async)
	operationID *string // nil for sync messages, value for async operations
	isProgress  bool    // true if this is a progress update
	isComplete  bool    // true if async operation completed

	// NEW: Handler identification
	handlerName    string // Formatted/padded Handler name for display
	RawHandlerName string // Unformatted raw handler name used for matching/updating
	handlerColor   string // NEW: Handler-specific color for message formatting
}

// tabSection represents a tab section in the TUI with configurable fields and content
type tabSection struct {
	index              int      // index of the tab
	title              string   // eg: "BUILD", "TEST"
	fieldHandlers      []*field // Field actions configured for the section
	sectionDescription string   // eg: "Press 't' to compile", "Press 'r' to run tests"
	// internal use
	tabContents          []tabContent // message contents
	indexActiveEditField int          // Índice del campo de configuración seleccionado
	tui                  *DevTUI
	mu                   sync.RWMutex // Para proteger tabContents y writingHandlers de race conditions

	// Writing handler registry for external handlers using new interfaces
	writingHandlers []*anyHandler // CAMBIO: slice en lugar de map para thread-safety
}

// getWritingHandler busca un handler por nombre en el slice thread-safe
func (ts *tabSection) getWritingHandler(name string) *anyHandler {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	for _, h := range ts.writingHandlers {
		if h.Name() == name {
			return h
		}
	}
	return nil
}

func (hw *handlerWriter) Write(p []byte) (n int, err error) {
	msg := Convert(string(p)).TrimSpace().String()
	if msg != "" {
		message, msgType := Translate(msg).StringType()

		var operationID string
		var handlerColor string
		if handler := hw.tabSection.getWritingHandler(hw.handlerName); handler != nil {
			operationID = handler.GetLastOperationID()
			handlerColor = handler.handlerColor // NEW: Get handler color
		}

		hw.tabSection.tui.sendMessageWithHandler(message, msgType, hw.tabSection, hw.handlerName, operationID, handlerColor)

		if msgType == Msg.Error {
			hw.tabSection.tui.Logger(msg)
		}
	}
	return len(p), nil
}

// registerLoggerFunc creates a logger function that handles variadic arguments
func (ts *tabSection) registerLoggerFunc(handler HandlerLogger, color string) func(message ...any) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	var anyH *anyHandler
	// Automatically detect if handler implements HandlerLoggerTracker (Name + MessageTracker)
	if tracker, ok := handler.(interface {
		Name() string
		GetLastOperationID() string
		SetLastOperationID(string)
	}); ok {
		anyH = NewWriterTrackerHandler(tracker, color)
	} else {
		anyH = NewWriterHandler(handler, color)
	}

	ts.writingHandlers = append(ts.writingHandlers, anyH)
	return func(message ...any) {
		if len(message) == 0 {
			return
		}

		// Format the message similar to fmt.Sprint
		var msg string
		if len(message) == 1 {
			if str, ok := message[0].(string); ok {
				msg = str
			} else {
				msg = Fmt("%v", message[0])
			}
		} else {
			msg = Fmt("%v", message[0])
			for _, m := range message[1:] {
				msg += " " + Fmt("%v", m)
			}
		}

		var operationID string
		var handlerColor string
		if handler := ts.getWritingHandler(anyH.Name()); handler != nil {
			operationID = handler.GetLastOperationID()
			handlerColor = handler.handlerColor // NEW: Get handler color
		}

		messageStr, msgType := Translate(msg).StringType()
		ts.tui.sendMessageWithHandler(messageStr, msgType, ts, anyH.Name(), operationID, handlerColor)

		if msgType == Msg.Error {
			ts.tui.Logger(msg)
		}
	}
}

// HandlerLogger wraps tabSection with handler identification
type handlerWriter struct {
	tabSection  *tabSection
	handlerName string
}

func (t *tabSection) addNewContent(msgType MessageType, content string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tabContents = append(t.tabContents, t.tui.createTabContent(content, msgType, t, "", "", ""))
}

// NEW: updateOrAddContentWithHandler updates existing content by operationID or adds new if not found
// Returns true if content was updated, false if new content was added
func (t *tabSection) updateOrAddContentWithHandler(msgType MessageType, content string, handlerName string, operationID string, handlerColor string) (updated bool, newContent tabContent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If operationID is provided, try to find and update existing content
	if operationID != "" {
		for i := range t.tabContents {
			// Match by both operationID and handlerName to ensure each handler updates its own message
			if t.tabContents[i].operationID != nil &&
				*t.tabContents[i].operationID == operationID &&
				t.tabContents[i].RawHandlerName == handlerName {
				// Update existing content
				t.tabContents[i].Content = content
				t.tabContents[i].Type = msgType
				// Actualizar timestamp usando GetNewID directamente
				if t.tui.id != nil {
					t.tabContents[i].Timestamp = t.tui.id.GetNewID()
				} else {
					// Log the issue before using fallback
					if t.tui.Logger != nil {
						t.tui.Logger("Warning: unixid not initialized, using fallback timestamp for content update:", content)
					}
					// Graceful fallback when unixid initialization failed
					t.tabContents[i].Timestamp = time.Now().Format("15:04:05")
				}
				// Move updated content to end
				updatedContent := t.tabContents[i]
				t.tabContents = append(t.tabContents[:i], t.tabContents[i+1:]...)
				t.tabContents = append(t.tabContents, updatedContent)
				return true, updatedContent
			}
		}
	}

	// If not found or no operationID, add new content
	newContent = t.tui.createTabContent(content, msgType, t, handlerName, operationID, handlerColor)
	t.tabContents = append(t.tabContents, newContent)
	return false, newContent
}

// NewTabSection creates a new tab section and returns it as any for interface decoupling.
// The returned value must be passed to AddHandler/AddLogger methods.
//
// Example:
//
//	tab := tui.NewTabSection("BUILD", "Compiler Section")
//	tui.AddHandler(myHandler, 2*time.Second, "#3b82f6", tab)
func (t *DevTUI) NewTabSection(title, description string) any {
	tab := &tabSection{
		title:              title,
		sectionDescription: description,
		tui:                t,
	}

	// Automatically add to TabSections and initialize
	t.initTabSection(tab, len(t.TabSections))
	t.TabSections = append(t.TabSections, tab)

	return tab
}

// setActiveEditField sets the active edit field index
func (ts *tabSection) setActiveEditField(idx int) {
	ts.indexActiveEditField = idx
}

// Helper method to initialize a single tabSection
func (t *DevTUI) initTabSection(section *tabSection, index int) {
	section.index = index
	section.tui = t

	// Initialize field handlers
	handlers := section.fieldHandlers
	for j := range handlers {
		handlers[j].index = j
		handlers[j].cursor = 0
	}
	section.setFieldHandlers(handlers)
}
