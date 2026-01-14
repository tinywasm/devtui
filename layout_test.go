package devtui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestLayoutAlignment verifies that the Header title block and the Footer left block
// (Pagination + Spacer + Label) have exactly the same visual width.
// This ensures vertical alignment of the left column across the UI.
func TestLayoutAlignment(t *testing.T) {
	// Setup styles manually to mirror what's being used
	palette := DefaultPalette()
	style := newTuiStyle(palette)

	// --- HEADER CALCULATION ---
	// Emulate headerView logic
	// The header uses UIColumnWidth to truncate and render
	headerText := "TINYWASM/BUILD"
	truncatedHeader := fmt.Sprintf("%-*s", UIColumnWidth, headerText)
	if len(truncatedHeader) > UIColumnWidth {
		truncatedHeader = truncatedHeader[:UIColumnWidth]
	}

	// Apply style (Padding(0, 1))
	headerRendered := style.headerTitleStyle.Render(truncatedHeader)
	headerWidth := lipgloss.Width(headerRendered)

	// --- FOOTER CALCULATION ---
	// Emulate Pagination
	// Pagination uses PaginationColumnWidth (10)
	paginationText := " 2/ 3" // PaginationColumnWidth = 5
	if len(paginationText) != PaginationColumnWidth {
		// Ensure test data matches constant if possible, or just force it for the test
		paginationText = fmt.Sprintf("%-*s", PaginationColumnWidth, " 1/ 1")
	}
	paginationRendered := style.paginationStyle.Render(paginationText)
	paginationWidth := lipgloss.Width(paginationRendered)

	// Spacer
	spacerWidth := FooterSpacerWidth

	// Label
	// Label uses FooterLabelWidth
	labelText := "Compiler Mode"
	labelTruncated := fmt.Sprintf("%-*s", FooterLabelWidth, labelText)
	if len(labelTruncated) > FooterLabelWidth {
		labelTruncated = labelTruncated[:FooterLabelWidth]
	}

	// Footer Label uses headerTitleStyle (same as header)
	labelRendered := style.headerTitleStyle.Render(labelTruncated)
	labelWidth := lipgloss.Width(labelRendered)

	totalFooterLeftWidth := paginationWidth + spacerWidth + labelWidth

	t.Logf("\n--- LAYOUT ALIGNMENT VERIFICATION ---\n")
	t.Logf("Constants:")
	t.Logf("  UIColumnWidth:        %d", UIColumnWidth)
	t.Logf("  PaginationColumnWidth:%d", PaginationColumnWidth)
	t.Logf("  FooterSpacerWidth:    %d", FooterSpacerWidth)
	t.Logf("  FooterExtraPadding:   %d", FooterExtraPadding)
	t.Logf("  FooterLabelWidth:     %d (Calc: %d - %d - %d - %d)",
		FooterLabelWidth, UIColumnWidth, PaginationColumnWidth, FooterSpacerWidth, FooterExtraPadding)
	t.Logf("\nMeasurements:")
	t.Logf("  HEADER Width: %d", headerWidth)
	t.Logf("  FOOTER Width: %d (Pag:%d + Spc:%d + Lbl:%d)", totalFooterLeftWidth, paginationWidth, spacerWidth, labelWidth)
	t.Logf("  DIFF:         %d", headerWidth-totalFooterLeftWidth)

	if headerWidth != totalFooterLeftWidth {
		t.Errorf("MISALIGNMENT: Header width (%d) != Footer width (%d)", headerWidth, totalFooterLeftWidth)
	}
}
