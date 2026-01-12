package devtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestInteractiveHandler is a mock handler that implements HandlerInteractive
// for testing cursor behavior, ESC handling, and multi-step wizards
type TestInteractiveHandler struct {
	name           string
	label          string
	value          string
	waitingForUser bool
	changeCalled   bool
	lastValue      string
	cancelCalled   bool
	log            func(message ...any)
}

func NewTestInteractiveHandler(name, label, value string) *TestInteractiveHandler {
	return &TestInteractiveHandler{
		name:           name,
		label:          label,
		value:          value,
		waitingForUser: true,
		log:            func(...any) {},
	}
}

func (h *TestInteractiveHandler) Name() string            { return h.name }
func (h *TestInteractiveHandler) Label() string           { return h.label }
func (h *TestInteractiveHandler) Value() string           { return h.value }
func (h *TestInteractiveHandler) WaitingForUser() bool    { return h.waitingForUser }
func (h *TestInteractiveHandler) SetLog(f func(...any))   { h.log = f }
func (h *TestInteractiveHandler) AlwaysShowAllLogs() bool { return true }

func (h *TestInteractiveHandler) Change(newValue string) {
	h.changeCalled = true
	h.lastValue = newValue
	// Simulate a wizard step: after receiving input, suggest next value
	if h.label == "Project Name" {
		h.label = "Project Location"
		h.value = "parentfolder/" + newValue
		// Still waiting for user to confirm location
	} else {
		h.waitingForUser = false
	}
}

func (h *TestInteractiveHandler) Cancel() {
	h.cancelCalled = true
	h.waitingForUser = false
}

// --- Bug 1 Test: Cursor should be at end after step change ---

func TestInteractiveHandler_CursorAtEndAfterStepChange(t *testing.T) {
	h := DefaultTUIForTest()
	tab := h.NewTabSection("WIZARD", "Test wizard")

	handler := NewTestInteractiveHandler("Wizard", "Project Name", "")
	addInteractiveHandlerForTest(h, tab, handler)

	// Get tab section properly
	tabSection := h.TabSections[len(h.TabSections)-1]
	h.activeTab = tabSection.index
	h.ready = true

	field := tabSection.fieldHandlers[0]

	// Simulate user typing "myapp" and pressing Enter
	field.tempEditValue = "myapp"
	h.editModeActivated = true
	tabSection.indexActiveEditField = 0

	// Simulate Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	h.handleEditingConfigKeyboard(msg)

	// BUG: After step change, cursor should be at end of new value
	// The new value is "parentfolder/myapp" (18 chars as runes)
	expectedCursor := len([]rune(handler.Value()))
	if field.cursor != expectedCursor {
		t.Errorf("Bug 1: Cursor should be at end (%d), got %d", expectedCursor, field.cursor)
	}
}

// --- Bug 2 Test: ESC should always close edit mode ---

func TestInteractiveHandler_ESCAlwaysClosesEditMode(t *testing.T) {
	h := DefaultTUIForTest()
	tab := h.NewTabSection("WIZARD", "Test wizard")

	handler := NewTestInteractiveHandler("Wizard", "Project Name", "")
	addInteractiveHandlerForTest(h, tab, handler)

	// Get tab section properly
	tabSection := h.TabSections[len(h.TabSections)-1]
	h.activeTab = tabSection.index
	h.ready = true

	field := tabSection.fieldHandlers[0]

	// Activate edit mode (handler is waiting for user)
	h.editModeActivated = true
	field.tempEditValue = "somevalue"
	tabSection.indexActiveEditField = 0

	// Verify handler is still waiting for user
	if !handler.WaitingForUser() {
		t.Fatal("Handler should be waiting for user")
	}

	// Simulate ESC key
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	h.handleEditingConfigKeyboard(msg)

	// BUG: ESC should close edit mode even if WaitingForUser is true
	if h.editModeActivated {
		t.Error("Bug 2: ESC should close edit mode regardless of WaitingForUser")
	}
}

// --- Bug 3 Test: Enter should execute and update tempEditValue ---

func TestInteractiveHandler_EnterExecutesAndUpdatesTempEditValue(t *testing.T) {
	h := DefaultTUIForTest()
	tab := h.NewTabSection("WIZARD", "Test wizard")

	handler := NewTestInteractiveHandler("Wizard", "Project Name", "")
	addInteractiveHandlerForTest(h, tab, handler)

	// Get tab section properly
	tabSection := h.TabSections[len(h.TabSections)-1]
	h.activeTab = tabSection.index
	h.ready = true

	field := tabSection.fieldHandlers[0]

	// Simulate user typing "myapp"
	field.tempEditValue = "myapp"
	h.editModeActivated = true
	tabSection.indexActiveEditField = 0

	// Simulate Enter key
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	h.handleEditingConfigKeyboard(msg)

	// Handler.Change should have been called
	if !handler.changeCalled {
		t.Error("Handler.Change should have been called")
	}

	// BUG: Since handler still wants input (step 2), edit mode should stay open
	// AND tempEditValue should be updated to the new suggested value
	if handler.WaitingForUser() && h.editModeActivated {
		// Edit mode is open, tempEditValue should have the new value
		expectedValue := handler.Value() // "parentfolder/myapp"
		if field.tempEditValue != expectedValue {
			t.Errorf("Bug 3: tempEditValue should be '%s', got '%s'", expectedValue, field.tempEditValue)
		}
	}
}

// --- Bug 4 Test: Cursor blinking should not shift text layout ---

func TestInteractiveHandler_CursorBlinkingDoesNotShiftLayout(t *testing.T) {
	h := DefaultTUIForTest()
	tab := h.NewTabSection("WIZARD", "Test wizard")

	handler := NewTestInteractiveHandler("Wizard", "Project Location", "Test/myapp")
	addInteractiveHandlerForTest(h, tab, handler)

	// Get tab section properly
	tabSection := h.TabSections[len(h.TabSections)-1]
	h.activeTab = tabSection.index
	h.ready = true
	h.viewport.Width = 80
	h.viewport.Height = 24

	field := tabSection.fieldHandlers[0]

	// Activate edit mode
	h.editModeActivated = true
	field.tempEditValue = "Test/myapp"
	field.cursor = 4 // Cursor in the middle of text
	tabSection.indexActiveEditField = 0

	// Render with cursor visible
	h.cursorVisible = true
	renderedVisible := h.footerView()

	// Render with cursor invisible (blinking off)
	h.cursorVisible = false
	renderedInvisible := h.footerView()

	// BUG: Both renders should have the same VISUAL width
	// The cursor space should always be reserved, even when invisible
	// Note: We compare visual width (rune count in rendered area), not byte length
	// because the cursor character (▋) is 3 bytes while space is 1 byte
	visibleWidth := len([]rune(renderedVisible))
	invisibleWidth := len([]rune(renderedInvisible))
	if visibleWidth != invisibleWidth {
		t.Errorf("Bug 4: Rendered visual width differs between cursor states. Visible: %d runes, Invisible: %d runes",
			visibleWidth, invisibleWidth)
	}
}

// --- Test: Full wizard flow including confirming suggested value ---

func TestInteractiveHandler_FullWizardFlow(t *testing.T) {
	h := DefaultTUIForTest()
	tab := h.NewTabSection("WIZARD", "Test wizard")

	handler := NewTestInteractiveHandler("Wizard", "Project Name", "")
	addInteractiveHandlerForTest(h, tab, handler)

	// Get tab section properly
	tabSection := h.TabSections[len(h.TabSections)-1]
	h.activeTab = tabSection.index
	h.ready = true
	h.viewport.Width = 80
	h.viewport.Height = 24

	field := tabSection.fieldHandlers[0]

	// Step 1: User types "myapp" and presses Enter
	field.tempEditValue = "myapp"
	h.editModeActivated = true
	tabSection.indexActiveEditField = 0

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	h.handleEditingConfigKeyboard(msg)

	// After step 0 completes:
	// - handler.label = "Project Location"
	// - handler.value = "parentfolder/myapp"
	// - handler.waitingForUser = true
	// - edit mode should still be active
	if !h.editModeActivated {
		t.Fatal("Edit mode should still be active after step 0")
	}

	// Step 2: User presses Enter WITHOUT modifying the suggested value
	// The tempEditValue should have been updated to the new suggested value
	expectedValue := handler.Value()
	if field.tempEditValue != expectedValue {
		t.Errorf("tempEditValue should be '%s', got '%s'", expectedValue, field.tempEditValue)
	}

	// Simulate pressing Enter to confirm the suggested value
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	h.handleEditingConfigKeyboard(msg)

	// After step 1 completes:
	// - handler.Change should have been called
	// - handler.waitingForUser = false
	// - edit mode should be closed
	if handler.WaitingForUser() {
		t.Error("Handler should not be waiting for user after step 1")
	}

	if h.editModeActivated {
		t.Error("Edit mode should be closed after step 1")
	}
}

// --- Helper: addInteractiveHandlerForTest ---

func addInteractiveHandlerForTest(h *DevTUI, tabAny any, handler *TestInteractiveHandler) {
	ts := tabAny.(*tabSection)

	ah := &anyHandler{
		handlerType:  handlerTypeInteractive,
		nameFunc:     handler.Name,
		labelFunc:    handler.Label,
		valueFunc:    handler.Value,
		changeFunc:   func(v string) { handler.Change(v) },
		editModeFunc: handler.WaitingForUser, // This is how WaitingForUser is wired
		editableFunc: func() bool { return true },
	}

	f := &field{
		handler:   ah,
		parentTab: ts,
	}

	ts.fieldHandlers = append(ts.fieldHandlers, f)
}
