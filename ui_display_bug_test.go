package devtui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestUIDisplayBug reproduce el bug específico de visualización que describe el usuario
func TestUIDisplayBug(t *testing.T) {
	t.Run("UI should show updated value after editing, not old value", func(t *testing.T) {
		// Setup exactly like main.go but with TestMode disabled to get real async behavior
		config := &TuiConfig{
			AppName:  "DevTUI - Display Bug Test",
		}

		tui := NewTUI(config)

		// Keep test mode disabled to enable real async behavior for this test
		tui.SetTestMode(false)

		// Create port handler with initial value "433" (like in the image)
		portHandler := &PortTestHandler{currentPort: "433"}
		tab := tui.NewTabSection("Server", "Server configuration")
		tui.AddHandler(portHandler, "", tab)

		// Initialize viewport
		tui.viewport.Width = 80
		tui.viewport.Height = 24

		// Navigate to Server tab
		tui.activeTab = 0
		tabSection := tab.(*tabSection)
		portField := tabSection.FieldHandlers[0]

		// Step 3: User edits the field to "8080"
		// Enter edit mode
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// Realistic: Clear field with backspace and type new value
		// Clear existing text first
		for i := 0; i < 5; i++ { // Clear any existing text
			tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyBackspace})
		}
		// Type "8080"
		for _, char := range "8080" {
			tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}})
		}

		// Press Enter to save
		tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// Give some time for async operation to complete
		time.Sleep(200 * time.Millisecond)

		// CRITICAL TEST: The field should now show "8080", not "433"
		displayedValue := portField.Value()
		if displayedValue != "8080" {
			t.Errorf("BUG CONFIRMED: field.Value() should return '8080' but returns '%s'", displayedValue)
		}

		// Test the footerInput rendering
		footerContent := tui.renderFooterInput()

		// The footer should contain the updated value "8080", not "433"
		if strings.Contains(footerContent, "433") {
			t.Errorf("BUG CONFIRMED: UI still shows old value '433' instead of new value '8080'")
		}

		if !strings.Contains(footerContent, "8080") {
			t.Errorf("BUG CONFIRMED: UI does not show new value '8080'")
		}
	})

	t.Run("Test immediate UI update after value change", func(t *testing.T) {
		// Test if the problem is in the immediate update
		config := &TuiConfig{
			AppName:  "DevTUI - Immediate Update Test",
		}

		tui := NewTUI(config)
		portHandler := &PortTestHandler{currentPort: "433"}
		tab := tui.NewTabSection("Server", "Config")
		tui.AddHandler(portHandler, "", tab)

		tui.viewport.Width = 80
		tui.viewport.Height = 24
		tui.activeTab = 0

		tabSection := tab.(*tabSection)
		portField := tabSection.FieldHandlers[0]

		// Manually update the handler (simulating successful change)
		portHandler.currentPort = "8080" // Direct update to handler

		if portField.Value() != "8080" {
			t.Errorf("Expected field.Value() to be '8080' after direct handler update, got '%s'", portField.Value())
		}

		// Force viewport update
		tui.updateViewport()

		// Test footer specifically
		footerContent := tui.renderFooterInput()

		if strings.Contains(footerContent, "433") {
			t.Errorf("UI still shows old value after direct handler update")
		}

		if !strings.Contains(footerContent, "8080") {
			t.Errorf("UI does not show new value '8080' after direct handler update")
		}
	})
}
