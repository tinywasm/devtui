package devtui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPaginationWritersOnlyTab(t *testing.T) {
	h := DefaultTUIForTest(func(messages ...any) {})
	h.viewport.Width = 80
	h.viewport.Height = 24
	h.paginationStyle = lipgloss.NewStyle().Background(lipgloss.Color(h.Secondary)).Foreground(lipgloss.Color(h.Foreground))

	// Create a tab with only writers, no field handlers
	h.TabSections = h.TabSections[:0]
	logs := h.NewTabSection("Logs", "System Logs")

	h.activeTab = 0
	h.AddHandler(&SystemLogWriter{name: "SystemLog"}, "", logs)

	// Call the real footerView rendering logic
	output := h.footerView()
	expected := "1/ 1" // Look for the core pagination text without spacing
	if !strings.Contains(output, expected) {
		t.Errorf("Writers-only tab pagination failed: got %q, want %q", output, expected)
	}
}

// SystemLogWriter for test
type SystemLogWriter struct {
	name string
}

func (w *SystemLogWriter) Name() string          { return w.name }
func (w *SystemLogWriter) SetLog(f func(...any)) {}
