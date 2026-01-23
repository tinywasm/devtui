package devtui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ThreadSafePortTestHandler is a thread-safe version for race condition testing
type ThreadSafePortTestHandler struct {
	mu          sync.RWMutex
	currentPort string
}

func (h *ThreadSafePortTestHandler) Label() string          { return "Port" }
func (h *ThreadSafePortTestHandler) Editable() bool         { return true }
func (h *ThreadSafePortTestHandler) Timeout() time.Duration { return 3 * time.Second }

func (h *ThreadSafePortTestHandler) Value() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentPort
}

func (h *ThreadSafePortTestHandler) Change(newValue string, progress chan<- string) {
	portStr := strings.TrimSpace(newValue)
	if portStr == "" {
		if progress != nil {
			progress <- "port cannot be empty"
		}
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		if progress != nil {
			progress <- "port must be a number"
		}
		return
	}
	if port < 1 || port > 65535 {
		if progress != nil {
			progress <- "port must be between 1 and 65535"
		}
		return
	}

	// Thread-safe update
	h.mu.Lock()
	h.currentPort = portStr
	h.mu.Unlock()

	if progress != nil {
		progress <- fmt.Sprintf("Port configured: %d", port)
	}
}

// TestHandlerValueUpdateAfterEdit tests that the field displays the updated value from handler after editing
func TestHandlerValueUpdateAfterEdit(t *testing.T) {
	t.Run("Field should display updated value from handler after successful edit", func(t *testing.T) {
		// Setup
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		// Initialize viewport
		h.viewport.Width = 80
		h.viewport.Height = 24

		// Create a new tab with our port handler
		portHandler := &PortTestHandler{currentPort: "8080"}
		tab := h.NewTabSection("Server", "Server configuration")
		h.AddHandler(portHandler, "", tab)

		// Get the test tab index (should be the last one added)
		testTabIndex := len(h.TabSections) - 1
		h.activeTab = testTabIndex

		// Get the field
		tabSection := tab.(*tabSection)
		field := tabSection.fieldHandlers[0]

		// Verify initial state
		initialValue := field.Value()
		expectedInitial := "8080"
		if initialValue != expectedInitial {
			t.Errorf("Expected initial value '%s', got '%s'", expectedInitial, initialValue)
		}

		t.Logf("Initial field value: '%s'", field.Value())

		// Realistic: User presses Enter to enter edit mode
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// Verify edit mode was activated
		if !h.editModeActivated {
			t.Error("Expected edit mode to be activated after pressing Enter")
		}

		// Realistic: User clears the field with multiple backspaces then types "80"
		// First, simulate moving cursor to end and then backspace to clear
		for i := 0; i < 5; i++ { // Clear existing "8080" (4 chars + buffer)
			h.handleKeyboard(tea.KeyMsg{Type: tea.KeyBackspace})
		}
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})

		t.Logf("Before pressing Enter - tempEditValue: '%s', handler Value(): '%s'",
			field.tempEditValue, field.Value())

		// User presses Enter to save the change
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		t.Logf("After pressing Enter - tempEditValue: '%s', handler Value(): '%s'",
			field.tempEditValue, field.Value())

		// The critical test: field.Value() should now return the updated value from handler
		expectedNewValue := "80"
		actualValue := field.Value()
		if actualValue != expectedNewValue {
			t.Errorf("Expected field.Value() to return updated value '%s', got '%s'",
				expectedNewValue, actualValue)
		}

		// Also verify the handler's internal state was updated
		if portHandler.currentPort != expectedNewValue {
			t.Errorf("Expected handler.currentPort to be '%s', got '%s'",
				expectedNewValue, portHandler.currentPort)
		}

		// Verify edit mode is deactivated
		if h.editModeActivated {
			t.Error("Expected edit mode to be deactivated after pressing Enter")
		}

		// Verify tempEditValue is cleared
		if field.tempEditValue != "" {
			t.Errorf("Expected tempEditValue to be cleared after Enter, got '%s'", field.tempEditValue)
		}
	})

	t.Run("Field should display error message when validation fails", func(t *testing.T) {
		// Setup
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		// Initialize viewport
		h.viewport.Width = 80
		h.viewport.Height = 24

		// Create a new tab with our port handler
		portHandler := &PortTestHandler{currentPort: "8080"}
		tab := h.NewTabSection("Server", "Server configuration")
		h.AddHandler(portHandler, "", tab)

		// Get the test tab index
		testTabIndex := len(h.TabSections) - 1
		h.activeTab = testTabIndex

		// Get the field
		tabSection := tab.(*tabSection)
		field := tabSection.fieldHandlers[0]

		// Realistic: User presses Enter to enter edit mode
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// Realistic: User clears field with backspace then types invalid value "99999"
		// Clear existing text
		for i := 0; i < 5; i++ { // Clear existing "8080" (4 chars + buffer)
			h.handleKeyboard(tea.KeyMsg{Type: tea.KeyBackspace})
		}
		// Type "99999"
		for _, char := range "99999" {
			h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		}

		originalValue := field.Value()
		t.Logf("Before pressing Enter with invalid value - tempEditValue: '%s', handler Value(): '%s'",
			field.tempEditValue, originalValue)

		// User presses Enter to save the invalid change
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		t.Logf("After pressing Enter with invalid value - tempEditValue: '%s', handler Value(): '%s'",
			field.tempEditValue, field.Value())

		// The field value should remain unchanged when validation fails
		actualValue := field.Value()
		if actualValue != originalValue {
			t.Errorf("Expected field.Value() to remain unchanged at '%s', got '%s'",
				originalValue, actualValue)
		}

		// Handler's internal state should also remain unchanged
		if portHandler.currentPort != originalValue {
			t.Errorf("Expected handler.currentPort to remain '%s', got '%s'",
				originalValue, portHandler.currentPort)
		}
	})
}