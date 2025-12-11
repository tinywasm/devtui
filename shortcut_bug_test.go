package devtui

import (
	"strings"
	"testing"
	"time"

	example "github.com/tinywasm/devtui/example"
	tea "github.com/charmbracelet/bubbletea"
)

// TestShortcutBug reproduces the exact bug: when pressing shortcut from DIFFERENT tab,
// it navigates correctly but executes BOTH handlers in target tabSection instead of just one
func TestShortcutBug(t *testing.T) {
	// Setup exactly like main.go
	config := &TuiConfig{
		AppName:  "Demo",
		ExitChan: make(chan bool),
		Logger: func(messages ...any) {
			t.Logf("DevTUI Log: %v", messages)
		},
	}

	tui := NewTUI(config)

	// Disable test mode to get real behavior like in main.go
	tui.SetTestMode(false)

	// Recreate exact structure from main.go
	dashboard := tui.NewTabSection("Dashboard", "System Overview")
	tui.AddHandler(&example.StatusHandler{}, 0, "", dashboard)

	config_tab := tui.NewTabSection("Config", "System Configuration")
	databaseHandler := &example.DatabaseHandler{ConnectionString: "postgres://localhost:5432/mydb"}
	tui.AddHandler(databaseHandler, 2*time.Second, "", config_tab)

	backupHandler := &example.BackupHandler{}
	tui.AddHandler(backupHandler, 5*time.Second, "", config_tab)

	// Initialize viewport
	tui.viewport.Width = 80
	tui.viewport.Height = 24

	t.Logf("=== REPRODUCING SHORTCUT BUG FROM DIFFERENT TAB ===")

	// THIS IS THE KEY: Start from DIFFERENT tab (Dashboard), not Config tab
	tui.activeTab = 1 // Dashboard tab (NOT Config tab where the handlers are)

	t.Logf("Step 1: Currently on Dashboard tab (index %d)", tui.activeTab)
	t.Logf("Step 2: Config tab with handlers is at index 2")

	// Clear previous actions
	databaseHandler.LastAction = ""

	// Show initial state - no messages should be visible yet
	initialContent := tui.ContentView()
	t.Logf("Step 3: Initial UI content (should be empty):\n%s", initialContent)

	// NOW press 't' shortcut from Dashboard tab
	// This should:
	// 1. Navigate to Config tab (index 2)
	// 2. Execute ONLY DatabaseHandler (not BackupHandler)
	t.Logf("Step 4: Pressing 't' shortcut from Dashboard tab...")

	tui.handleKeyboard(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Give time for async operations
	time.Sleep(1 * time.Second)

	t.Logf("Step 5: After shortcut - activeTab: %d, DatabaseHandler.LastAction: '%s'",
		tui.activeTab, databaseHandler.LastAction)

	// Get UI content after shortcut
	content := tui.ContentView()
	t.Logf("Step 6: UI Content after 't' shortcut:\n%s", content)

	// BUG CHECK: Look for evidence that BOTH handlers executed
	hasDatabaseActivity := strings.Contains(content, "DatabaseConfig") || strings.Contains(content, "Testing database")
	hasBackupActivity := strings.Contains(content, "SystemBackup") || strings.Contains(content, "Preparing backup") || strings.Contains(content, "BackingUp")

	// Count the number of different handler activities in the output
	databaseCount := strings.Count(content, "[DatabaseConfig]")
	backupCount := strings.Count(content, "[SystemBackup]")

	t.Logf("Step 7: Activity count - DatabaseConfig: %d, SystemBackup: %d", databaseCount, backupCount)

	// Expected behavior: Only DatabaseHandler should have executed
	if databaseHandler.LastAction != "test" {
		t.Errorf("Expected DatabaseHandler.LastAction to be 'test', got '%s'", databaseHandler.LastAction)
	}

	// BUG: If BackupHandler also shows activity, that's the bug
	if hasBackupActivity {
		t.Errorf("BUG CONFIRMED: BackupHandler shows activity when only DatabaseHandler shortcut 't' was pressed")
		t.Logf("BackupHandler should NOT execute when DatabaseHandler shortcut is triggered")
	}

	if hasDatabaseActivity && hasBackupActivity {
		t.Errorf("BUG CONFIRMED: Both handlers executed when shortcut pressed from different tab")
		t.Logf("This matches the bug described - shortcut navigation executes multiple handlers")
	}

	// Verify correct navigation happened
	if tui.activeTab != 2 {
		t.Errorf("Expected to navigate to Config tab (index 2), but activeTab is %d", tui.activeTab)
	}

	t.Logf("=== END OF BUG REPRODUCTION TEST ===")
}