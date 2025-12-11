package devtui

import (
	"testing"
	"time"

	"github.com/tinywasm/devtui/example"
	tea "github.com/charmbracelet/bubbletea"
)

func TestShortcutKeyboard_SingleCharacterHandling(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode
	tui.testMode = true

	// Create tab section with handler
	tabSection := tui.NewTabSection("Test", "Test tab")
	handler := &example.DatabaseHandler{ConnectionString: "postgres://localhost:5432/testdb"}
	tui.AddHandler(handler, 5*time.Second, "", tabSection)

	// Set up proper state (not in edit mode, proper tab selection)
	tui.editModeActivated = false
	tui.activeTab = 1 // TinyWasm tab (shortcuts tab is 0)

	// Create keyboard message for shortcut 't' (test connection)
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'t'},
	}

	// Handle keyboard input
	handled, _ := tui.handleKeyboard(keyMsg)
	if handled {
		t.Error("Expected shortcut handling to return false (stop processing)")
	}

	// Verify handler LastAction was set to 'test' and connection string remains unchanged
	if handler.LastAction != "test" {
		t.Errorf("Expected handler LastAction to be 'test', got '%s'", handler.LastAction)
	}
	if handler.Value() != "postgres://localhost:5432/testdb" {
		t.Errorf("Expected handler value to remain unchanged, got '%s'", handler.Value())
	}
}

func TestShortcutKeyboard_NonExistentShortcut(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode
	tui.testMode = true

	// Create tab section with handler
	tabSection := tui.NewTabSection("Test", "Test tab")
	handler := &example.DatabaseHandler{ConnectionString: "postgres://localhost:5432/testdb"}
	tui.AddHandler(handler, 5*time.Second, "", tabSection)

	// Set up proper state
	tui.editModeActivated = false
	tui.activeTab = 1

	// Store initial value
	initialValue := handler.Value()

	// Create keyboard message for non-existent shortcut 'x'
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'x'},
	}

	// Handle keyboard input
	handled, _ := tui.handleKeyboard(keyMsg)
	if !handled {
		t.Error("Expected non-shortcut key to return true (continue processing)")
	}

	// Verify handler was not executed
	if handler.Value() != initialValue {
		t.Errorf("Expected handler value to remain '%s', got '%s'", initialValue, handler.Value())
	}
}

func TestShortcutKeyboard_EditModeIgnoresShortcuts(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode
	tui.testMode = true

	// Create tab section with handler
	tabSection := tui.NewTabSection("Test", "Test tab")
	handler := &example.DatabaseHandler{ConnectionString: "postgres://localhost:5432/testdb"}
	tui.AddHandler(handler, 5*time.Second, "", tabSection)

	// Set up edit mode state
	tui.editModeActivated = true
	tui.activeTab = 1

	// Store initial value
	initialValue := handler.Value()

	// Create keyboard message for shortcut 'c'
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'c'},
	}

	// Handle keyboard input (should go to edit mode handler, not shortcuts)
	_, _ = tui.handleKeyboard(keyMsg)

	// In edit mode, shortcuts should not be processed
	// The exact return value depends on edit mode handling, but handler should not be triggered
	if handler.Value() == "coding" {
		t.Error("Expected shortcut to be ignored in edit mode")
	}

	// Handler value should remain unchanged by shortcut
	if handler.Value() != initialValue {
		t.Errorf("Expected handler value to remain '%s', got '%s'", initialValue, handler.Value())
	}
}

func TestShortcutKeyboard_MultipleCharacterIgnored(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode
	tui.testMode = true

	// Create tab section with handler
	tabSection := tui.NewTabSection("Test", "Test tab")
	handler := &example.DatabaseHandler{ConnectionString: "postgres://localhost:5432/testdb"}
	tui.AddHandler(handler, 5*time.Second, "", tabSection)

	// Set up proper state
	tui.editModeActivated = false
	tui.activeTab = 1

	// Store initial value
	initialValue := handler.Value()

	// Create keyboard message with multiple characters
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'c', 'd'}, // Multiple characters should be ignored
	}

	// Handle keyboard input
	handled, _ := tui.handleKeyboard(keyMsg)
	if !handled {
		t.Error("Expected multi-character input to return true (continue processing)")
	}

	// Verify handler was not executed
	if handler.Value() != initialValue {
		t.Errorf("Expected handler value to remain '%s', got '%s'", initialValue, handler.Value())
	}
}

func TestShortcutKeyboard_InvalidTabIndex(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode
	tui.testMode = true

	// Manually register a shortcut with invalid tab index
	entry := &ShortcutEntry{
		Key:         "x",
		Description: "invalid",
		TabIndex:    999, // Invalid tab index
		FieldIndex:  0,
		HandlerName: "Invalid",
		Value:       "x",
	}
	tui.shortcutRegistry.Register("x", entry)

	// Set up proper state
	tui.editModeActivated = false
	tui.activeTab = 0

	// Create keyboard message for invalid shortcut
	keyMsg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'x'},
	}

	// Handle keyboard input (should handle gracefully)
	handled, _ := tui.handleKeyboard(keyMsg)
	if handled {
		t.Error("Expected invalid shortcut to return false and stop processing")
	}

	// No crash should occur, and processing should stop
}