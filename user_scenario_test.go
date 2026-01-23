package devtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestUserScenarioExactReplication tests the exact user scenario described:
// 1. Field shows "initial test value"
// 2. User clears the field (shows empty)
// 3. User types "g" and it should show only "g", not "g initial test value"
func TestUserScenarioExactReplication(t *testing.T) {
	t.Run("Exact user scenario: clear field then type should not show old value", func(t *testing.T) {
		// Setup: Create TUI with test handler
		testHandler := NewTestEditableHandler("Test Field", "initial test value")
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		// Create test tab and register handler
		tab := h.NewTabSection("Test Tab", "Test description")
		h.AddHandler(testHandler, "", tab)

		// Initialize viewport with a reasonable size for testing
		h.viewport.Width = 80
		h.viewport.Height = 24

		// Get the first field from the default configuration
		tabSection := h.TabSections[GetFirstTestTabIndex()]
		field := tabSection.fieldHandlers[0]

		// Step 1: Field shows "initial test value" (this is the initial state)
		expectedInitialValue := "initial test value"
		if field.Value() != expectedInitialValue {
			t.Fatalf("Expected initial field value to be '%s', got '%s'", expectedInitialValue, field.Value())
		}

		// Enter editing mode (this happens when user presses Enter on the field)
		h.editModeActivated = true
		h.activeTab = GetFirstTestTabIndex() // Set the active tab to the test tab
		h.TabSections[GetFirstTestTabIndex()].indexActiveEditField = 0

		// When entering edit mode, tempEditValue is initialized with the current value
		field.tempEditValue = field.Value()
		field.cursor = len([]rune(field.Value())) // Cursor at the end

		t.Logf("Step 1 - Initial state: Value='%s', tempEditValue='%s', cursor=%d",
			field.Value(), field.tempEditValue, field.cursor)

		// Step 2: User selects all content and deletes it (field becomes empty)
		// This simulates the user clearing the entire field content
		field.tempEditValue = ""
		field.cursor = 0

		t.Logf("Step 2 - After clearing: Value='%s', tempEditValue='%s', cursor=%d",
			field.Value(), field.tempEditValue, field.cursor)

		// Verify the field appears empty to the user
		if field.tempEditValue != "" {
			t.Errorf("After clearing, tempEditValue should be empty, got '%s'", field.tempEditValue)
		}

		// Step 3: User types "g"
		// The bug was: this would result in "g initial test value"
		// The fix: this should result in just "g"
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'g'},
		})

		t.Logf("Step 3 - After typing 'g': Value='%s', tempEditValue='%s', cursor=%d",
			field.Value(), field.tempEditValue, field.cursor)

		// Verify the fix: should be just "g"
		expectedAfterTyping := "g"
		if field.tempEditValue != expectedAfterTyping {
			t.Errorf("FAILED: After typing 'g', expected tempEditValue='%s', got '%s'",
				expectedAfterTyping, field.tempEditValue)
		}

		if field.cursor != 1 {
			t.Errorf("After typing 'g', expected cursor=1, got %d", field.cursor)
		}

		// Additional verification: type more characters to ensure continued functionality
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'o'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'o'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'d'},
		})

		expectedFinalValue := "good"
		if field.tempEditValue != expectedFinalValue {
			t.Errorf("After typing 'good', expected tempEditValue='%s', got '%s'",
				expectedFinalValue, field.tempEditValue)
		}

		if field.cursor != 4 {
			t.Errorf("After typing 'good', expected cursor=4, got %d", field.cursor)
		}

		t.Logf("SUCCESS: Field editing works correctly. Final value: '%s'", field.tempEditValue)
	})
}

// TestBackspaceAfterClear tests that backspace also works correctly after clearing
func TestBackspaceAfterClear(t *testing.T) {
	t.Run("Backspace should work correctly when field is cleared", func(t *testing.T) {
		// Setup with test handler
		testHandler := NewTestEditableHandler("Test Field", "test value")
		h := DefaultTUIForTest(func(messages ...any) {})

		// Create test tab and register handler
		tab := h.NewTabSection("Test Tab", "Test description")
		h.AddHandler(testHandler, "", tab)

		h.viewport.Width = 80
		h.viewport.Height = 24

		tabSection := h.TabSections[GetFirstTestTabIndex()]
		field := tabSection.fieldHandlers[0]

		// Enter editing mode
		h.editModeActivated = true
		h.activeTab = GetFirstTestTabIndex() // Set the active tab to the test tab
		h.TabSections[GetFirstTestTabIndex()].indexActiveEditField = 0

		// Initialize editing
		field.tempEditValue = field.Value()
		field.cursor = len([]rune(field.Value()))

		// Clear field
		field.tempEditValue = ""
		field.cursor = 0

		// Type some text
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'t'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'e'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'s'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'t'},
		})

		// Should have "test"
		if field.tempEditValue != "test" {
			t.Errorf("Expected 'test', got '%s'", field.tempEditValue)
		}

		// Use backspace to remove last character
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyBackspace})

		// Should have "tes"
		if field.tempEditValue != "tes" {
			t.Errorf("After backspace, expected 'tes', got '%s'", field.tempEditValue)
		}

		if field.cursor != 3 {
			t.Errorf("After backspace, expected cursor=3, got %d", field.cursor)
		}
	})
}
