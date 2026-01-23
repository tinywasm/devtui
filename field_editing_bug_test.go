package devtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestFieldEditingBugReplication tests the bug where after clearing a field
// and typing new content, the old value gets concatenated with the new input
func TestFieldEditingBugReplication(t *testing.T) {
	t.Run("Bug replication: After clearing field and typing, old value gets concatenated", func(t *testing.T) {
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

		// Use centralized function to get correct tab index
		testTabIndex := GetFirstTestTabIndex()
		tabSection := h.TabSections[testTabIndex]
		field := tabSection.fieldHandlers[0]
		// The field already has "initial test value" from DefaultTUIForTest

		// Switch to the test tab and enter editing mode
		h.activeTab = testTabIndex
		h.editModeActivated = true
		h.TabSections[testTabIndex].indexActiveEditField = 0

		// Initialize tempEditValue with the current value (this happens when entering edit mode)
		field.tempEditValue = field.Value()
		field.cursor = len([]rune(field.Value())) // Cursor at the end

		t.Logf("Initial state - Value: '%s', tempEditValue: '%s', cursor: %d",
			field.Value(), field.tempEditValue, field.cursor)

		// Check the available text width to understand why text isn't being inserted
		_, availableTextWidth := h.calculateInputWidths(field.handler.Label())
		t.Logf("Available text width: %d", availableTextWidth)

		// Step 1: User selects all content and deletes it (simulating Ctrl+A + Delete)
		// This should clear tempEditValue completely
		field.tempEditValue = ""
		field.cursor = 0

		t.Logf("After clearing field - Value: '%s', tempEditValue: '%s', cursor: %d",
			field.Value(), field.tempEditValue, field.cursor)

		// Step 2: User types a new character 'g'
		// This should now work correctly and only show 'g'
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'g'},
		})

		t.Logf("After typing 'g' - Value: '%s', tempEditValue: '%s', cursor: %d",
			field.Value(), field.tempEditValue, field.cursor)

		// Now it should work correctly
		expectedCorrectValue := "g"

		if field.tempEditValue != expectedCorrectValue {
			t.Errorf("Expected tempEditValue to be '%s', got '%s'", expectedCorrectValue, field.tempEditValue)
		}

		if field.cursor != 1 {
			t.Errorf("Expected cursor to be at position 1, got %d", field.cursor)
		}
	})
}

// TestFieldEditingCorrectBehavior tests the correct behavior after fixing the bug
func TestFieldEditingCorrectBehavior(t *testing.T) {
	t.Run("Field editing should work correctly when tempEditValue is empty", func(t *testing.T) {
		// Setup with test handler
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

		// Use centralized function to get correct tab index
		testTabIndex := GetFirstTestTabIndex()
		tabSection := h.TabSections[testTabIndex]
		field := tabSection.fieldHandlers[0]

		// The field already has "initial test value" from DefaultTUIForTest
		// No need to set it again as SetValue is deprecated

		// Switch to the test tab and enter editing mode
		h.activeTab = testTabIndex
		h.editModeActivated = true
		h.TabSections[testTabIndex].indexActiveEditField = 0

		// Simulate user clearing the field completely
		field.tempEditValue = ""
		field.cursor = 0

		// Type multiple characters
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'h'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'e'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'l'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'l'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'o'},
		})

		expectedValue := "hello"
		if field.tempEditValue != expectedValue {
			t.Errorf("Expected tempEditValue to be '%s', got '%s'", expectedValue, field.tempEditValue)
		}

		if field.cursor != 5 {
			t.Errorf("Expected cursor to be at position 5, got %d", field.cursor)
		}
	})
}
