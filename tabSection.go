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
	handlerName    string      // Formatted/padded Handler name for display
	RawHandlerName string      // Unformatted raw handler name used for matching/updating
	handlerColor   string      // NEW: Handler-specific color for message formatting
	handlerType    handlerType // NEW: Type of handler (Interactive, Display, etc.) for formatting
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

	// Animation state management
	animationStopChans map[string]chan struct{}
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

		var handlerColor string
		if handler := hw.tabSection.getWritingHandler(hw.handlerName); handler != nil {
			handlerColor = handler.handlerColor // NEW: Get handler color
		}

		// operationID is now always the handlerName for tracking
		trackingID := hw.handlerName
		var hType handlerType = handlerTypeLoggable
		if handler := hw.tabSection.getWritingHandler(hw.handlerName); handler != nil {
			hType = handler.handlerType
		}

		hw.tabSection.tui.sendMessageWithHandler(message, msgType, hw.tabSection, hw.handlerName, trackingID, handlerColor, hType)

		if msgType == Msg.Error {
			hw.tabSection.tui.Logger(msg)
		}
	}
	return len(p), nil
}

// HandlerLogger wraps tabSection with handler identification
type handlerWriter struct {
	tabSection  *tabSection
	handlerName string
}

func (t *tabSection) addNewContent(msgType MessageType, content string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tabContents = append(t.tabContents, t.tui.createTabContent(content, msgType, t, "", "", "", handlerTypeLoggable))
	if len(t.tabContents) > 500 {
		t.tabContents = t.tabContents[len(t.tabContents)-500:]
	}
}

// NEW: updateOrAddContentWithHandler updates existing content by handler name (trackingID)
// Returns true if content was updated, false if new content was added
func (t *tabSection) updateOrAddContentWithHandler(msgType MessageType, content string, handlerName string, trackingID string, handlerColor string, hType handlerType) (updated bool, newContent tabContent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// trackingID is now the handlerName for automatic tracking
	if trackingID != "" {
		for i := range t.tabContents {
			if t.tabContents[i].RawHandlerName == trackingID {
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

	// If not found or no trackingID, add new content
	newContent = t.tui.createTabContent(content, msgType, t, handlerName, trackingID, handlerColor, hType)
	t.tabContents = append(t.tabContents, newContent)

	// Keep only last 500 messages to prevent memory issues and slow rendering
	if len(t.tabContents) > 500 {
		t.tabContents = t.tabContents[len(t.tabContents)-500:]
	}

	return false, newContent
}

// NewTabSection creates a new tab section and returns it as any for interface decoupling.
// The returned value must be passed to the AddHandler method.
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
		animationStopChans: make(map[string]chan struct{}),
	}

	// Automatically add to TabSections and initialize
	t.initTabSection(tab, len(t.TabSections))
	t.TabSections = append(t.TabSections, tab)

	return tab
}

// SetActiveTab sets the currently active tab by section reference.
func (t *DevTUI) SetActiveTab(section any) {
	tab, ok := section.(*tabSection)
	if !ok || tab == nil {
		return
	}

	for i, ts := range t.TabSections {
		if ts == tab {
			t.activeTab = i
			t.notifyTabActive(i) // Notify handlers that tab is now active
			t.RefreshUI()
			return
		}
	}
}

// notifyTabActive notifies all handlers in the specified tab that it has become active.
// Used for lazy execution or logging that requires the screen logger to be present.
func (t *DevTUI) notifyTabActive(tabIndex int) {
	if tabIndex < 0 || tabIndex >= len(t.TabSections) {
		return
	}

	tab := t.TabSections[tabIndex]
	tab.mu.RLock()
	defer tab.mu.RUnlock()

	notified := make(map[any]bool)

	// Notify all field handlers
	for _, f := range tab.fieldHandlers {
		if f.handler != nil && f.handler.origHandler != nil {
			if aware, ok := f.handler.origHandler.(TabAware); ok {
				if !notified[aware] {
					notified[aware] = true
					go aware.OnTabActive()
				}
			}
		}
	}

	// Notify all writing-only handlers
	for _, h := range tab.writingHandlers {
		if h.origHandler != nil {
			if aware, ok := h.origHandler.(TabAware); ok {
				if !notified[aware] {
					notified[aware] = true
					go aware.OnTabActive()
				}
			}
		}
	}
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

// stopAnimation stops any running animation for a given handler
func (ts *tabSection) stopAnimation(handlerName string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if stopChan, ok := ts.animationStopChans[handlerName]; ok {
		close(stopChan)
		delete(ts.animationStopChans, handlerName)
	}
}

// startAnimation starts a new auto-animation for a given handler
func (ts *tabSection) startAnimation(handlerName, baseMessage string, msgType MessageType, color string) {
	// First stop any existing animation
	ts.stopAnimation(handlerName)

	stopChan := make(chan struct{})
	ts.mu.Lock()
	ts.animationStopChans[handlerName] = stopChan
	ts.mu.Unlock()

	go func() {
		dots := ""
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				dots += " ."
				if len(dots) > 6 { // Max 3 dots " . . ."
					dots = ""
				}
				// Update the same line (using handlerName as trackingID)
				ts.tui.sendMessageWithHandler(baseMessage+dots, msgType, ts, handlerName, handlerName, color, handlerTypeLoggable)
			}
		}
	}()
}
