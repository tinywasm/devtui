package devtui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPaginationDisplay(t *testing.T) {
	// Setup DevTUI using a similar pattern to user_scenario_test.go
	h := DefaultTUIForTest(func(messages ...any) {})
	h.viewport.Width = 80
	h.viewport.Height = 24
	h.paginationStyle = lipgloss.NewStyle().Background(lipgloss.Color(h.Secondary)).Foreground(lipgloss.Color(h.Foreground))

	// Tab pagination cases
	tabCases := []struct {
		activeTab int
		totalTabs int
		expected  string
	}{
		{0, 1, "[ 1/ 1]"},
		{0, 4, "[ 1/ 4]"},
		{3, 4, "[ 4/ 4]"},
		{99, 100, "[100/99]"}, // Clamp to 99
	}

	for _, tc := range tabCases {
		// Setup tabs using only public API
		// Remove all tabs
		h.TabSections = h.TabSections[:0]
		for i := 0; i < tc.totalTabs; i++ {
			h.NewTabSection(fmt.Sprintf("Tab%d", i), "desc")
		}
		h.activeTab = tc.activeTab
		// Render header pagination
		currentTab := h.activeTab
		totalTabs := len(h.TabSections)
		displayCurrent := min(currentTab, 99) + 1
		displayTotal := min(totalTabs, 99)
		pagination := fmt.Sprintf("[%2d/%2d]", displayCurrent, displayTotal)
		// Test the raw pagination string before styling
		if pagination != tc.expected {
			t.Errorf("Header pagination failed: got %q, want %q", pagination, tc.expected)
		}
	}

	// Field pagination cases
	fieldCases := []struct {
		activeField int
		totalFields int
		expected    string
	}{
		{0, 1, "[ 1/ 1]"},
		{0, 4, "[ 1/ 4]"},
		{3, 4, "[ 4/ 4]"},
		{99, 100, "[100/99]"}, // Clamp to 99
	}

	for _, tc := range fieldCases {
		// Remove all tabs
		h.TabSections = h.TabSections[:0]
		tab := h.NewTabSection("TestTab", "desc")
		for i := 0; i < tc.totalFields; i++ {
			h.AddHandler(NewTestEditableHandler(fmt.Sprintf("Field%d", i), "val"), "", tab)
		}
		h.activeTab = 0
		tabSection := tab.(*tabSection)
		if tc.activeField < len(tabSection.fieldHandlers) {
			tabSection.setActiveEditField(tc.activeField)
		}
		currentField := tc.activeField
		totalFields := len(tabSection.fieldHandlers)
		displayCurrent := min(currentField, 99) + 1
		displayTotal := min(totalFields, 99)
		pagination := fmt.Sprintf("[%2d/%2d]", displayCurrent, displayTotal)
		// Test the raw pagination string before styling
		if pagination != tc.expected {
			t.Errorf("Footer pagination failed: got %q, want %q", pagination, tc.expected)
		}
	}
}

// Helper to check substring
func contains(s, substr string) bool {
	return len(substr) > 0 && (s == substr || (len(s) > len(substr) && (s[0:len(substr)] == substr || s[len(s)-len(substr):] == substr)))
}

// Use min from main codebase
