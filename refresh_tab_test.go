package devtui

import (
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/tinywasm/fmt"
)

// TestRefreshCurrentTab verifica que el método público RefreshUI
// funciona correctamente y puede ser llamado desde herramientas externas
func TestRefreshCurrentTab(t *testing.T) {
	config := &TuiConfig{
		AppName:  "TestRefresh",
		ExitChan: make(chan bool),
		Logger:   func(messages ...any) { t.Log(messages...) },
	}

	tui := NewTUI(config)

	// Create a tab section
	tab := tui.NewTabSection("TEST", "Test refresh functionality")
	tabSection := tab.(*tabSection)

	// Add initial content
	tabSection.addNewContent(Msg.Info, "Initial message")

	// Verify initial content via direct access (like other tests do)
	if len(tabSection.tabContents) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(tabSection.tabContents))
	}
	if tabSection.tabContents[0].Content != "Initial message" {
		t.Errorf("Expected 'Initial message', got '%s'", tabSection.tabContents[0].Content)
	}

	// Simulate external tool adding content
	tabSection.addNewContent(Msg.Info, "External tool message")

	// Verify updated content
	if len(tabSection.tabContents) != 2 {
		t.Errorf("Expected 2 content items, got %d", len(tabSection.tabContents))
	}
	if tabSection.tabContents[1].Content != "External tool message" {
		t.Errorf("Expected 'External tool message', got '%s'", tabSection.tabContents[1].Content)
	}

	// Call RefreshUI to ensure it doesn't panic
	tui.RefreshUI()
}

// TestRefreshCurrentTabBeforeReady verifica que RefreshUI
// no causa errores si se llama antes de que la TUI esté lista
func TestRefreshCurrentTabBeforeReady(t *testing.T) {
	config := &TuiConfig{
		AppName:  "TestRefreshBeforeReady",
		ExitChan: make(chan bool),
		Logger:   func(messages ...any) { t.Log(messages...) },
	}

	tui := NewTUI(config)

	// Call RefreshUI before Start() - should not panic
	tui.RefreshUI()

	// If we got here without panic, test passes
	t.Log("RefreshUI handled gracefully before TUI was ready")
}

// TestRefreshCurrentTabFromMultipleGoroutines verifica que RefreshUI
// es thread-safe y puede ser llamado desde múltiples goroutines simultáneamente
func TestRefreshCurrentTabFromMultipleGoroutines(t *testing.T) {
	config := &TuiConfig{
		AppName:  "TestConcurrentRefresh",
		ExitChan: make(chan bool),
		Logger:   func(messages ...any) { t.Log(messages...) },
	}

	tui := NewTUI(config)

	tab := tui.NewTabSection("CONCURRENT", "Test concurrent refresh")
	tabSection := tab.(*tabSection)

	// Simulate multiple external tools calling RefreshUI concurrently
	var refreshWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		refreshWg.Add(1)
		go func(id int) {
			defer refreshWg.Done()
			// Simulate external tool work
			time.Sleep(time.Millisecond * time.Duration(id*2))
			msg := Fmt("Tool %d update", id)
			tabSection.addNewContent(Msg.Info, msg)
			tui.RefreshUI() // Public method call - should not panic
		}(i)
	}

	refreshWg.Wait()

	// Verify all updates are present via direct access
	tabSection.mu.RLock()
	contentCount := len(tabSection.tabContents)
	tabSection.mu.RUnlock()

	if contentCount != 10 {
		t.Errorf("Expected 10 content items, got %d", contentCount)
	}

	// Verify content contains all updates
	tabSection.mu.RLock()
	defer tabSection.mu.RUnlock()
	for i := 0; i < 10; i++ {
		expected := Fmt("Tool %d update", i)
		found := false
		for _, content := range tabSection.tabContents {
			if strings.Contains(content.Content, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Content should contain '%s'", expected)
		}
	}
}
