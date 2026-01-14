package devtui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestCornerElementsHaveConsistentWidth verifies all 3 corner elements use the same width
func TestCornerElementsHaveConsistentWidth(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	// Verify PaginationColumnWidth is odd (allows centering)
	if PaginationColumnWidth%2 == 0 {
		t.Errorf("PaginationColumnWidth should be odd for centering, got %d", PaginationColumnWidth)
	}

	// Test all scroll icon widths are consistent
	h := DefaultTUIForTest()
	h.viewport.Width = 80
	h.viewport.Height = 24

	// Each scroll state should produce the same visual width
	scrollStates := []struct {
		name    string
		content string
		yOffset int
	}{
		{"AllVisible", "", 0},                                // atTop && atBottom
		{"CanScrollDown", strings.Repeat("line\n", 100), 0},  // atTop && !atBottom
		{"CanScrollUp", strings.Repeat("line\n", 100), 50},   // !atTop && atBottom
		{"CanScrollBoth", strings.Repeat("line\n", 200), 50}, // !atTop && !atBottom
	}

	var expectedWidth int
	for i, state := range scrollStates {
		h.viewport.SetContent(state.content)
		h.viewport.YOffset = state.yOffset

		info := h.renderScrollInfo()
		width := lipgloss.Width(info)

		if i == 0 {
			expectedWidth = width
		} else if width != expectedWidth {
			t.Errorf("Scroll state %s has width %d, expected %d", state.name, width, expectedWidth)
		}
	}
}

// TestPaginationAndScrollInfoSameWidth verifies pagination and scroll info have same width
func TestPaginationAndScrollInfoSameWidth(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	h := DefaultTUIForTest()
	h.viewport.Width = 80

	// Render pagination (uses PaginationColumnWidth + style padding)
	paginationText := lipgloss.NewStyle().Width(PaginationColumnWidth).Align(lipgloss.Center).Render(" 1/ 1")
	paginationStyled := h.paginationStyle.Render(paginationText)
	paginationWidth := lipgloss.Width(paginationStyled)

	// Render scroll info (should have same width as pagination)
	h.viewport.SetContent("") // atTop && atBottom
	scrollInfo := h.renderScrollInfo()
	scrollWidth := lipgloss.Width(scrollInfo)

	if paginationWidth != scrollWidth {
		t.Errorf("Pagination width (%d) should equal scroll info width (%d)", paginationWidth, scrollWidth)
	}
}

// TestScrollIconContentWidth verifies the raw scroll icon content is exactly PaginationColumnWidth
func TestScrollIconContentWidth(t *testing.T) {
	// Each icon string should be exactly 5 runes (PaginationColumnWidth)
	icons := []struct {
		name    string
		content string
	}{
		{"AllVisible", "  ■  "},
		{"CanScrollDown", "  ▼  "},
		{"CanScrollUp", "  ▲  "},
		{"CanScrollBoth", " ▼ ▲ "},
	}

	for _, icon := range icons {
		runeCount := len([]rune(icon.content))
		if runeCount != PaginationColumnWidth {
			t.Errorf("Scroll icon %s has %d runes, expected %d (PaginationColumnWidth)",
				icon.name, runeCount, PaginationColumnWidth)
		}
	}
}
