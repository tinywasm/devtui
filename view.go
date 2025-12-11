package devtui

import (
	. "github.com/tinywasm/fmt"
	"github.com/charmbracelet/lipgloss"
)

func (h *DevTUI) View() string {
	if !h.ready {
		return "\n  Initializing..."
	}
	return Fmt("%s\n%s\n%s", h.headerView(), h.viewport.View(), h.footerView())
	// return Fmt("%s\n%s\n%s", h.headerView(), h.ContentView(), h.footerView())
}

// ContentView renderiza los mensajes para una sección de contenido
func (h *DevTUI) ContentView() string {
	if len(h.TabSections) == 0 {
		return "No tabs created yet"
	}
	if h.activeTab >= len(h.TabSections) {
		h.activeTab = 0
	}

	// Proteger el acceso a tabContents con mutex
	section := h.TabSections[h.activeTab]
	section.mu.RLock()
	tabContent := make([]tabContent, len(section.tabContents)) // Copia para evitar retener el lock
	copy(tabContent, section.tabContents)
	section.mu.RUnlock()

	var contentLines []string

	// NEW: Add display handler content if active field is a Display handler
	fieldHandlers := section.fieldHandlers
	if len(fieldHandlers) > 0 && section.indexActiveEditField < len(fieldHandlers) {
		activeField := fieldHandlers[section.indexActiveEditField]
		if activeField.hasContentMethod() {
			displayContent := activeField.getDisplayContent()
			if displayContent != "" {
				// Add display content at the top of the content view with Primary color
				highlightStyle := h.textContentStyle.Foreground(lipgloss.Color(h.Primary))
				contentLines = append(contentLines, highlightStyle.Render(displayContent))
				// Add separator line if there are also tab messages
				if len(tabContent) > 0 {
					contentLines = append(contentLines, "")
				}
			}
		}
	}

	// Add regular tab content messages
	for _, content := range tabContent {
		formattedMsg := h.formatMessage(content)
		contentLines = append(contentLines, h.textContentStyle.Render(formattedMsg))
	}
	return Convert(contentLines).Join("\n").String()
}

func (h *DevTUI) headerView() string {
	if len(h.TabSections) == 0 {
		return h.headerTitleStyle.Render(h.AppName + "/No tabs")
	}
	if h.activeTab >= len(h.TabSections) {
		h.activeTab = 0
	}

	tab := h.TabSections[h.activeTab]

	// Truncar el título si es necesario
	headerText := h.AppName + "/" + tab.title
	truncatedHeader := Convert(headerText).Truncate(h.labelWidth, 0).String()

	// Aplicar el estilo base para garantizar un ancho fijo
	fixedWidthHeader := h.labelStyle.Render(truncatedHeader)

	// Aplicar el estilo visual manteniendo el ancho fijo
	title := h.headerTitleStyle.Render(fixedWidthHeader)

	// Pagination logic
	currentTab := h.activeTab
	totalTabs := len(h.TabSections)
	if currentTab > 99 || totalTabs > 99 {
		if h.Logger != nil {
			h.Logger("Tab limit exceeded:", currentTab, "/", totalTabs)
		}
	}
	displayCurrent := min(currentTab, 99) + 1 // 1-based for display
	displayTotal := min(totalTabs, 99)
	pagination := Fmt("%2d/%2d", displayCurrent, displayTotal)
	paginationStyled := h.paginationStyle.Render(pagination)
	lineWidth := h.viewport.Width - lipgloss.Width(title) - lipgloss.Width(paginationStyled)
	line := h.lineHeadFootStyle.Render(Convert("─").Repeat(max(0, lineWidth)).String())
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line, paginationStyled)
}