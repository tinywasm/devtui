package devtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestChangeFuncControlsEmptyFieldBehavior demonstrates that changeFunc has full control
// over what happens when a field is cleared, not DevTUI
func TestChangeFuncControlsEmptyFieldBehavior(t *testing.T) {
	t.Run("changeFunc can reject empty values", func(t *testing.T) {
		// Handler centralizado que rechaza valores vacíos
		customHandler := NewTestRequiredFieldHandler("Required Field", "initial value")

		// Create TUI with custom field
		h := DefaultTUIForTest()
		// Create a test tab and add the handler using new API
		tab := h.NewTabSection("Test Tab", "Test description")
		h.AddHandler(customHandler, "", tab)

		h.viewport.Width = 80
		h.viewport.Height = 24

		// Get the field from the test tab
		tabSection := tab.(*tabSection)
		field := tabSection.fieldHandlers[0]

		// Switch to test tab and enter editing mode
		h.activeTab = len(h.TabSections) - 1 // Use the last added tab
		h.editModeActivated = true
		h.TabSections[h.activeTab].indexActiveEditField = 0

		// Initialize editing
		field.tempEditValue = field.Value()
		field.cursor = len([]rune(field.Value()))

		// Clear the field
		field.tempEditValue = ""
		field.cursor = 0

		// Press Enter - changeFunc should reject the empty value
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// The field should still have the original value because changeFunc rejected the empty value
		expectedValue := "initial value"
		if field.Value() != expectedValue {
			t.Errorf("Expected field to keep original value '%s' after changeFunc rejects empty, got '%s'", expectedValue, field.Value())
		}

		// Edit mode should be deactivated even if changeFunc fails
		if h.editModeActivated {
			t.Error("Expected edit mode to be deactivated after Enter, even when changeFunc fails")
		}
	})

	t.Run("changeFunc can accept and transform empty values", func(t *testing.T) {
		// Handler centralizado que acepta valores vacíos
		customHandler := NewTestOptionalFieldHandler("Optional Field", "original value")

		// Create TUI with custom field
		h := DefaultTUIForTest(func(messages ...any) {})
		// Create a test tab and add the handler using new API
		tab := h.NewTabSection("Test Tab", "Test description")
		h.AddHandler(customHandler, "", tab)

		h.viewport.Width = 80
		h.viewport.Height = 24

		// Get the field from the test tab
		tabSection := tab.(*tabSection)
		field := tabSection.fieldHandlers[0]

		// Switch to test tab and enter editing mode
		h.activeTab = len(h.TabSections) - 1 // Use the last added tab
		h.editModeActivated = true
		h.TabSections[h.activeTab].indexActiveEditField = 0

		// Initialize editing
		field.tempEditValue = field.Value()
		field.cursor = len([]rune(field.Value()))

		// Clear the field
		field.tempEditValue = ""
		field.cursor = 0

		// Press Enter - changeFunc should accept and transform the empty value
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// The field should have the transformed value from changeFunc
		expectedValue := "Default Value"
		if field.Value() != expectedValue {
			t.Errorf("Expected field value to be '%s' after changeFunc transforms empty value, got '%s'", expectedValue, field.Value())
		}
	})

	t.Run("changeFunc can preserve empty values", func(t *testing.T) {
		// Handler centralizado que preserva valores vacíos tal como son
		customHandler := NewTestClearableFieldHandler("Clearable Field", "some value")

		// Create TUI with custom field
		h := DefaultTUIForTest(func(messages ...any) {})
		// Create a test tab and add the handler using new API
		tab := h.NewTabSection("Test Tab", "Test description")
		h.AddHandler(customHandler, "", tab)

		h.viewport.Width = 80
		h.viewport.Height = 24

		// Get the field from the test tab
		tabSection := tab.(*tabSection)
		field := tabSection.fieldHandlers[0]

		// Switch to test tab and enter editing mode
		h.activeTab = len(h.TabSections) - 1 // Use the last added tab
		h.editModeActivated = true
		h.TabSections[h.activeTab].indexActiveEditField = 0

		// Initialize editing
		field.tempEditValue = field.Value()
		field.cursor = len([]rune(field.Value()))

		// Clear the field
		field.tempEditValue = ""
		field.cursor = 0

		// Press Enter - changeFunc should preserve the empty value
		h.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// The field should be empty
		expectedValue := ""
		if field.Value() != expectedValue {
			t.Errorf("Expected field value to be empty '%s', got '%s'", expectedValue, field.Value())
		}
	})
}
