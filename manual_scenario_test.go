package devtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestManualScenarioValueUpdate reproduce exactamente el escenario manual del usuario
func TestManualScenarioValueUpdate(t *testing.T) {
	t.Run("Manual scenario: Enter edit mode, change value, press Enter, verify value updates", func(t *testing.T) {
		// Setup: Use DefaultTUIForTest para consistencia
		tui := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		// Add a custom port handler to the existing TUI
		portHandler := &PortTestHandler{currentPort: "8080"}
		serverTab := tui.NewTabSection("Server", "Server configuration")
		tui.AddHandler(portHandler, "", serverTab)

		// Initialize viewport
		tui.viewport.Width = 80
		tui.viewport.Height = 24

		// Get the index of the server tab (should be the last one added)
		serverTabIndex := len(tui.TabSections) - 1
		tui.activeTab = serverTabIndex

		// Get the port field
		serverTabSection := serverTab.(*tabSection)
		portField := serverTabSection.fieldHandlers[0]

		// Verify initial state
		initialValue := portField.Value()
		t.Logf("STEP 1: Initial field value: '%s'", initialValue)

		if initialValue != "8080" {
			t.Errorf("Expected initial value '8080', got '%s'", initialValue)
		}

		// STEP 2: Simulate user pressing Enter to edit the port field
		// This should trigger edit mode
		t.Logf("STEP 2: User presses Enter to edit field...")

		// Simulate pressing Enter on the port field (like in manual scenario)
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// Verify we're in edit mode
		if !tui.editModeActivated {
			t.Error("Expected to be in edit mode after pressing Enter on editable field")
		}

		// Verify tempEditValue is initialized with current value
		t.Logf("After entering edit mode - tempEditValue: '%s', field.Value(): '%s'",
			portField.tempEditValue, portField.Value())

		if portField.tempEditValue != initialValue {
			t.Errorf("Expected tempEditValue to be '%s', got '%s'", initialValue, portField.tempEditValue)
		}

		// STEP 3: User clears the field and types "80" (like in the image)
		t.Logf("STEP 3: User edits field to '80'...")

		// Realistic: Clear field completely with backspace
		// Clear existing "8080" (4 characters)
		for i := 0; i < 5; i++ { // 4 chars + buffer
			tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyBackspace})
		}

		// Type "8"
		tui.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'8'},
		})

		// Type "0"
		tui.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'0'},
		})

		// Verify the tempEditValue is now "80"
		t.Logf("After typing '80' - tempEditValue: '%s', field.Value(): '%s'",
			portField.tempEditValue, portField.Value())

		if portField.tempEditValue != "80" {
			t.Errorf("Expected tempEditValue to be '80', got '%s'", portField.tempEditValue)
		}

		// The field.Value() should still return the old value until Enter is pressed
		if portField.Value() != "8080" {
			t.Errorf("Expected field.Value() to still return '8080' during editing, got '%s'", portField.Value())
		}

		// STEP 4: User presses Enter to confirm the change
		t.Logf("STEP 4: User presses Enter to confirm change...")

		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// CRITICAL TEST: After pressing Enter, field.Value() should return the new value
		newValue := portField.Value()
		t.Logf("STEP 5: After pressing Enter - tempEditValue: '%s', field.Value(): '%s'",
			portField.tempEditValue, newValue)

		// This is the key assertion - the field should now show the updated value
		if newValue != "80" {
			t.Errorf("CRITICAL BUG: Expected field.Value() to return '80' after successful edit, got '%s'", newValue)
		}

		// Verify the handler's internal state was updated
		if portHandler.currentPort != "80" {
			t.Errorf("Expected handler.currentPort to be '80', got '%s'", portHandler.currentPort)
		}

		// Verify edit mode is deactivated
		if tui.editModeActivated {
			t.Error("Expected edit mode to be deactivated after pressing Enter")
		}

		// Verify tempEditValue is cleared
		if portField.tempEditValue != "" {
			t.Errorf("Expected tempEditValue to be cleared after Enter, got '%s'", portField.tempEditValue)
		}

		// STEP 6: Simulate entering edit mode again to verify the value persists
		t.Logf("STEP 6: Enter edit mode again to verify value persists...")

		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// The tempEditValue should now be initialized with the NEW value
		t.Logf("Re-entering edit mode - tempEditValue: '%s', field.Value(): '%s'",
			portField.tempEditValue, portField.Value())

		if portField.tempEditValue != "80" {
			t.Errorf("When re-entering edit mode, expected tempEditValue to be '80', got '%s'", portField.tempEditValue)
		}

		if portField.Value() != "80" {
			t.Errorf("When re-entering edit mode, expected field.Value() to be '80', got '%s'", portField.Value())
		}
	})
}

// TestDisplayedValueInUI tests what the user actually sees on screen
func TestDisplayedValueInUI(t *testing.T) {
	t.Run("UI should display the updated value after successful edit", func(t *testing.T) {
		// Setup: Use DefaultTUIForTest para consistencia
		tui := DefaultTUIForTest()

		// Add port handler
		portHandler := &PortTestHandler{currentPort: "8080"}
		serverTab := tui.NewTabSection("Server", "Server configuration")
		// Pass the handler and a duration (e.g., 0 for no delay)
		tui.AddHandler(portHandler, "", serverTab)

		tui.viewport.Width = 80
		tui.viewport.Height = 24

		// Get the server tab index
		serverTabIndex := len(tui.TabSections) - 1
		tui.activeTab = serverTabIndex

		serverTabSection := serverTab.(*tabSection)
		portField := serverTabSection.fieldHandlers[0]

		// Simulate the editing process realistically
		// 1. Enter edit mode
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// 2. Clear field and type new value
		for i := 0; i < 5; i++ { // Clear existing "8080"
			tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyBackspace})
		}
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}})
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})

		// 3. Press Enter to confirm
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// 4. Check what the UI renders
		// Get the content that would be displayed
		content := tui.ContentView()
		t.Logf("UI Content after update:\n%s", content)

		// The UI should now show the updated port value
		// We can't easily test the exact rendered output, but we can verify
		// that the field's Value() method returns the correct value
		displayedValue := portField.Value()
		if displayedValue != "80" {
			t.Errorf("UI should display updated value '80', but field.Value() returns '%s'", displayedValue)
		}
	})
}