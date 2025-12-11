package devtui

import (
	"testing"
	"time"

	"github.com/tinywasm/devtui/example"
)

func TestShortcutIntegration_DatabaseHandler(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode for synchronous behavior
	tui.testMode = true

	// Create a tab section
	tabSection := tui.NewTabSection("Database", "Database configuration")

	// Add database handler with shortcuts
	handler := &example.DatabaseHandler{ConnectionString: "initial"}
	tui.AddHandler(handler, 5*time.Second, "", tabSection)

	// Verify shortcuts were registered
	expectedShortcuts := map[string]string{
		"t": "test connection",
		"b": "backup database",
	}

	for key, expectedDesc := range expectedShortcuts {
		entry, exists := tui.shortcutRegistry.Get(key)
		if !exists {
			t.Errorf("Expected shortcut '%s' to be registered", key)
			continue
		}
		if entry.Description != expectedDesc {
			t.Errorf("Expected description '%s' for key '%s', got '%s'", expectedDesc, key, entry.Description)
		}
	}

	// Test test connection shortcut
	entry, _ := tui.shortcutRegistry.Get("t")
	tui.executeShortcut(entry)
	if handler.LastAction != "test" {
		t.Errorf("Expected LastAction to be 'test', got '%s'", handler.LastAction)
	}

	// Test backup shortcut
	entry, _ = tui.shortcutRegistry.Get("b")
	tui.executeShortcut(entry)
	if handler.LastAction != "backup" {
		t.Errorf("Expected LastAction to be 'backup', got '%s'", handler.LastAction)
	}
}

func TestShortcutIntegration_NavigationBetweenTabs(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode
	tui.testMode = true

	// Create tab sections mimicking main.go structure
	// Dashboard tab with StatusHandler (DisplayHandler - no shortcuts)
	dashboard := tui.NewTabSection("Dashboard", "System Overview")
	statusHandler := &example.StatusHandler{}
	tui.AddHandler(statusHandler, 0, "", dashboard)

	// Config tab with DatabaseHandler (EditHandler with shortcuts) and BackupHandler (ExecutionHandler)
	config := tui.NewTabSection("Config", "System Configuration")
	databaseHandler := &example.DatabaseHandler{ConnectionString: "postgres://localhost:5432/mydb"}
	tui.AddHandler(databaseHandler, 2*time.Second, "", config)

	backupHandler := &example.BackupHandler{}
	tui.AddHandler(backupHandler, 5*time.Second, "", config)

	// Initially on first tab (shortcuts tab is index 0, Dashboard is index 1)
	tui.activeTab = 1

	// Execute shortcut from DatabaseHandler in Config tab (Config is index 2)
	entry, exists := tui.shortcutRegistry.Get("t")
	if !exists {
		t.Fatal("Expected shortcut 't' to be registered")
	}

	// Verify the entry points to the correct tab (Config tab)
	if entry.TabIndex != 2 {
		t.Errorf("Expected tab index 2, got %d", entry.TabIndex)
	}

	// Verify the entry points to the correct field (DatabaseHandler should be field 0, BackupHandler field 1)
	if entry.FieldIndex != 0 {
		t.Errorf("Expected field index 0 for DatabaseHandler, got %d", entry.FieldIndex)
	}

	// Execute shortcut
	tui.executeShortcut(entry)

	// Verify navigation happened to Config tab
	if tui.activeTab != 2 {
		t.Errorf("Expected active tab to be 2, got %d", tui.activeTab)
	}

	// Verify DatabaseHandler was executed (not BackupHandler)
	if databaseHandler.LastAction != "test" {
		t.Errorf("Expected DatabaseHandler LastAction to be 'test', got '%s'", databaseHandler.LastAction)
	}

	// Verify BackupHandler was NOT executed
	if backupHandler.GetLastOperationID() != "" {
		t.Error("BackupHandler should not have been executed")
	}

	// Test backup shortcut 'b'
	entry, exists = tui.shortcutRegistry.Get("b")
	if !exists {
		t.Fatal("Expected shortcut 'b' to be registered")
	}

	// Clear previous action
	databaseHandler.LastAction = ""

	// Execute backup shortcut
	tui.executeShortcut(entry)

	// Verify DatabaseHandler backup action was executed
	if databaseHandler.LastAction != "backup" {
		t.Errorf("Expected DatabaseHandler LastAction to be 'backup', got '%s'", databaseHandler.LastAction)
	}
}

// TestShortcutIntegration_ConflictingShortcuts tests what happens when multiple handlers
// try to register the same shortcut key
func TestShortcutIntegration_ConflictingShortcuts(t *testing.T) {
	// Create TUI instance
	exitChan := make(chan bool, 1)
	tui := NewTUI(&TuiConfig{
		AppName:  "TestApp",
		ExitChan: exitChan,
	})

	// Enable test mode
	tui.testMode = true

	// Create first tab with DatabaseHandler that has shortcuts "t" and "b"
	tab1 := tui.NewTabSection("Database1", "First database")
	handler1 := &example.DatabaseHandler{ConnectionString: "db1"}
	tui.AddHandler(handler1, 5*time.Second, "", tab1)

	// Create second tab with another DatabaseHandler that has same shortcuts
	tab2 := tui.NewTabSection("Database2", "Second database")
	handler2 := &example.DatabaseHandler{ConnectionString: "db2"}
	tui.AddHandler(handler2, 5*time.Second, "", tab2)

	// The last registered handler should win (handler2)
	entry, exists := tui.shortcutRegistry.Get("t")
	if !exists {
		t.Fatal("Expected shortcut 't' to be registered")
	}

	// Should point to the second tab (Database2 is index 2)
	if entry.TabIndex != 2 {
		t.Errorf("Expected tab index 2 for handler2, got %d", entry.TabIndex)
	}

	// Execute shortcut
	tui.executeShortcut(entry)

	// Only handler2 should be executed
	if handler2.LastAction != "test" {
		t.Errorf("Expected handler2 LastAction to be 'test', got '%s'", handler2.LastAction)
	}

	// handler1 should NOT be executed
	if handler1.LastAction != "" {
		t.Errorf("Expected handler1 LastAction to be empty, got '%s'", handler1.LastAction)
	}
}