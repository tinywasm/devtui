package devtui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tinywasm/fmt"
)

// calculateInputWidths calculates the width available for text input based on viewport and other elements
// Returns valueWidth (total width for the input area) and availableTextWidth (width for the text itself)
func (h *DevTUI) calculateInputWidths(fieldLabel string) (valueWidth, availableTextWidth int) {
	horizontalPadding := 1

	// Process label (same logic as footerInput.go)
	labelText := fmt.Convert(fieldLabel).Truncate(h.labelWidth, 0).String()
	fixedWidthLabel := h.labelStyle.Render(labelText)
	paddedLabel := h.headerTitleStyle.Render(fixedWidthLabel)

	// Calculate other components (All elements in footer)
	infoWidth := lipgloss.Width(h.renderScrollInfo())

	// Force fixed width for alignment (matching footerInput.go)
	fieldPagination := lipgloss.NewStyle().Width(PaginationColumnWidth).Align(lipgloss.Center).Render(" 1/ 1")
	paginationWidth := lipgloss.Width(h.paginationStyle.Render(fieldPagination))

	// Layout: pagination|space|label|space|value|space|scroll
	usedWidth := infoWidth + lipgloss.Width(paddedLabel) + paginationWidth + horizontalPadding*3

	// Calculate final widths
	valueWidth = h.viewport.Width - usedWidth
	if valueWidth < 10 {
		valueWidth = 10 // Mínimo
	}

	availableTextWidth = valueWidth - (horizontalPadding * 2)

	return valueWidth, availableTextWidth
}
