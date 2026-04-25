package devtui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestRealUserScenario simula exactamente lo que describe el usuario
func TestRealUserScenario(t *testing.T) {
	t.Run("Change port from 8080 to 80 like user described", func(t *testing.T) {
		// Setup using centralized DefaultTUIForTest
		tui := DefaultTUIForTest()

		// Create port handler exactly like main.go
		portHandler := &PortTestHandler{currentPort: "8080"}

		// Configure tab exactly like main.go
		serverTab := tui.NewTabSection("Server", "Server configuration")
		tui.AddHandler(portHandler, "", serverTab)

		// Initialize viewport
		tui.viewport.Width = 80
		tui.viewport.Height = 24

		// Get server tab index and set active
		serverTabIndex := len(tui.TabSections) - 1
		tui.activeTab = serverTabIndex
		serverTabSection := serverTab.(*tabSection)
		portField := serverTabSection.FieldHandlers[0]

		// User sees "8080" and wants to change it to "80"
		// User presses Enter to edit
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// User clears the field (simulating selecting all and deleting)
		portField.tempEditValue = ""
		portField.cursor = 0

		// User types "8"
		tui.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'8'},
		})

		// User types "0"
		tui.handleKeyboard(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{'0'},
		})

		// At this point user sees "80" in the field but field.Value() still returns "8080"
		// This is expected during editing

		// User presses Enter to confirm
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// NOW THE CRITICAL TEST: field.Value() should return "80"
		finalValue := portField.Value()
		if finalValue != "80" {
			t.Errorf("CRITICAL BUG: After editing, field.Value() should return '80', got '%s'", finalValue)
		}

		// Handler should also be updated
		if portHandler.currentPort != "80" {
			t.Errorf("Handler not updated: expected currentPort '80', got '%s'", portHandler.currentPort)
		}

		// If user enters edit mode again, they should see "80"
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// This is where the user's issue manifests: when re-entering edit mode,
		// tempEditValue should be "80", not "8080"
		if portField.tempEditValue != "80" {
			t.Errorf("BUG FOUND: When re-entering edit mode, tempEditValue should be '80', got '%s'",
				portField.tempEditValue)
		}

		if portField.Value() != "80" {
			t.Errorf("BUG FOUND: When re-entering edit mode, field.Value() should be '80', got '%s'",
				portField.Value())
		}
	})

	t.Run("Test what UI actually displays during editing", func(t *testing.T) {
		// Setup using centralized DefaultTUIForTest
		tui := DefaultTUIForTest()

		portHandler := &PortTestHandler{currentPort: "8080"}
		serverTab := tui.NewTabSection("Server", "Server configuration")
		tui.AddHandler(portHandler, "", serverTab)

		tui.viewport.Width = 80
		tui.viewport.Height = 24

		serverTabIndex := len(tui.TabSections) - 1
		tui.activeTab = serverTabIndex

		serverTabSection := serverTab.(*tabSection)
		portField := serverTabSection.FieldHandlers[0]

		// Phase 2: During editing
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// Realistic: User clears field and types "80"
		for i := 0; i < 5; i++ { // Clear "8080"
			tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyBackspace})
		}
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'8'}})
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})

		_ = tui.ContentView()

		// Phase 3: After saving
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		_ = tui.ContentView()

		// The UI should now show the updated value
		finalValue := portField.Value()
		if finalValue != "80" {
			t.Errorf("UI should show updated value '80', but field.Value() is '%s'", finalValue)
		}
	})
}
