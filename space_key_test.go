package devtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// setupTestWithEditableField configures a test environment with an editable field
func setupTestWithEditableField(t *testing.T) (*DevTUI, *field) {
	// Create test handler and TUI
	testHandler := NewTestEditableHandler("Test Field", "initial value")
	h := DefaultTUIForTest(func(messages ...any) {
		// Test logger - do nothing
	})

	// Create test tab and register handler
	tab := h.NewTabSection("Test Tab", "Test description")
	// Provide the required HandlerEdit and time.Duration arguments
	h.AddHandler(testHandler, "", tab)

	// Initialize viewport with a reasonable size for testing FIRST
	h.viewport.Width = 80
	h.viewport.Height = 24

	// Use centralized function to get correct tab index
	testTabIndex := GetFirstTestTabIndex()
	if testTabIndex >= len(h.TabSections) {
		t.Fatalf("Expected at least %d tab sections, got %d", testTabIndex+1, len(h.TabSections))
	}

	tabSection := h.TabSections[testTabIndex]
	field := tabSection.fieldHandlers[0]

	// Enter editing mode on the correct tab
	h.activeTab = testTabIndex
	h.editModeActivated = true
	h.TabSections[testTabIndex].indexActiveEditField = 0

	// Clear field and reset cursor
	field.tempEditValue = ""
	field.cursor = 0

	return h, field
}

// TestSpaceKeyInEditMode tests that the space key works correctly in edit mode
func TestSpaceKeyInEditMode(t *testing.T) {
	t.Run("Space key should insert space in edit mode", func(t *testing.T) {
		h, field := setupTestWithEditableField(t)

		// Type "hello" - one character at a time
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'h'},
		})

		t.Logf("After typing 'h' - tempEditValue: '%s', cursor: %d", field.tempEditValue, field.cursor)

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

		// Should have "hello"
		if field.tempEditValue != "hello" {
			t.Errorf("Expected 'hello', got '%s'", field.tempEditValue)
		}

		// Now try to add a space using KeySpace
		h.handleKeyboard(tea.KeyMsg{
			Type: tea.KeySpace,
		})

		t.Logf("After pressing space - tempEditValue: '%s', cursor: %d", field.tempEditValue, field.cursor)

		// Should have "hello " (hello with space)
		expectedAfterSpace := "hello "
		if field.tempEditValue != expectedAfterSpace {
			t.Errorf("Expected '%s', got '%s'", expectedAfterSpace, field.tempEditValue)
		}

		// Continue typing "world"
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'w'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'o'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'r'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'l'},
		})

		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'d'},
		})

		// Should have "hello world"
		expectedFinal := "hello world"
		if field.tempEditValue != expectedFinal {
			t.Errorf("Expected '%s', got '%s'", expectedFinal, field.tempEditValue)
		}
	})

	t.Run("Space using Runes should work", func(t *testing.T) {
		h, field := setupTestWithEditableField(t)

		// Type "hello" using Runes
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'h', 'e', 'l', 'l', 'o'},
		})

		// Add space using Runes
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{' '},
		})

		t.Logf("After adding space via Runes - tempEditValue: '%s', cursor: %d", field.tempEditValue, field.cursor)

		// Add "world" using Runes
		h.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'w', 'o', 'r', 'l', 'd'},
		})

		// Should have "hello world"
		expectedFinal := "hello world"
		if field.tempEditValue != expectedFinal {
			t.Errorf("Expected '%s', got '%s'", expectedFinal, field.tempEditValue)
		}
	})

	t.Run("Complete text editing with spaces should work", func(t *testing.T) {
		h, field := setupTestWithEditableField(t)

		// Type a complete sentence with spaces
		// "Hello world from Go"
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeySpace}) // First space
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeySpace}) // Second space
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeySpace}) // Third space
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

		expectedText := "Hello world from Go"
		if field.tempEditValue != expectedText {
			t.Errorf("Expected '%s', got '%s'", expectedText, field.tempEditValue)
		}

		if field.cursor != len([]rune(expectedText)) {
			t.Errorf("Expected cursor at position %d, got %d", len([]rune(expectedText)), field.cursor)
		}

		t.Logf("Successfully typed: '%s'", field.tempEditValue)
	})
}