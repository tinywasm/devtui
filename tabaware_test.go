package devtui

import (
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type mockTabAwareHandler struct {
	name        string
	val         string
	activeCount int
	mu          sync.Mutex
	log         func(...any)
}

func (m *mockTabAwareHandler) Name() string          { return m.name }
func (m *mockTabAwareHandler) Label() string         { return m.name }
func (m *mockTabAwareHandler) Value() string         { return m.val }
func (m *mockTabAwareHandler) Change(s string)       { m.val = s }
func (m *mockTabAwareHandler) SetLog(l func(...any)) { m.log = l }

func (m *mockTabAwareHandler) OnTabActive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeCount++
	if m.log != nil {
		m.log("tab active!")
	}
}

func (m *mockTabAwareHandler) getCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeCount
}

func TestTabAwareFeature(t *testing.T) {
	config := &TuiConfig{
		AppName: "TestTui",
	}
	tui := NewTUI(config)
	tui.SetTestMode(true)

	// Add 3 tabs (SHORTCUTS is tab 0 by default, so we'll have 4 total)
	tab1 := tui.NewTabSection("Tab1", "Desc1")
	_ = tui.NewTabSection("Tab2", "Desc2")

	handler := &mockTabAwareHandler{name: "TestTabAware", val: "foo"}

	// Register it in tab1
	tui.AddHandler(handler, "#00ff00", tab1)

	// Set active tab to tab1 manualy
	// tab 0 is shortcuts, tab 1 is Tab1
	tui.SetActiveTab(tab1)

	// wait for goroutine
	time.Sleep(10 * time.Millisecond)

	count := handler.getCount()
	if count != 1 {
		t.Errorf("Expected OnTabActive to be called 1 time, got %d", count)
	}

	// Change to another tab and back via keyboard
	tui.activeTab = 0
	msg := tea.KeyMsg{Type: tea.KeyTab}
	tui.handleKeyboard(msg) // Now it goes to Tab 1 (index 1) Which triggers OnTabActive

	time.Sleep(10 * time.Millisecond)

	count = handler.getCount()
	if count != 2 {
		t.Errorf("Expected OnTabActive to be called 2 times, got %d", count)
	}
}
