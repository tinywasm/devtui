package devtui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tinywasm/fmt"
)

// footerView renderiza la vista del footer
// Si hay campos activos, muestra el campo actual como input
// Si no hay campos, muestra una barra de desplazamiento estándar

func (h *DevTUI) footerView() string {
	// Verificar que haya tabs disponibles
	if len(h.TabSections) == 0 {
		return h.footerInfoStyle.Render("No tabs available")
	}
	if h.activeTab >= len(h.TabSections) {
		h.activeTab = 0
	}

	// Si hay campos disponibles, mostrar el input (independiente de si estamos en modo edición)
	if len(h.TabSections[h.activeTab].fieldHandlers) > 0 {
		return h.renderFooterInput()
	}

	// Si no hay campos, mostrar paginación de writers-only y scrollbar estándar
	tabSection := h.TabSections[h.activeTab]
	fieldHandlers := tabSection.fieldHandlers
	currentField := tabSection.indexActiveEditField
	totalFields := len(fieldHandlers)
	if currentField > 99 || totalFields > 99 {
		if h.Logger != nil {
			h.Logger("Field limit exceeded:", currentField, "/", totalFields)
		}
	}
	// Writers-only tab: show  1/ 1 for clarity
	var displayCurrent, displayTotal int
	if totalFields == 0 {
		displayCurrent = 1
		displayTotal = 1
	} else {
		displayCurrent = min(currentField, 99) + 1 // 1-based for display
		displayTotal = min(totalFields, 99)
	}
	// Force fixed width for alignment
	rawPag := fmt.Fmt("%2d/%2d", displayCurrent, displayTotal)
	fieldPagination := lipgloss.NewStyle().Width(PaginationColumnWidth).Align(lipgloss.Center).Render(rawPag)
	// Clip if necessary (though strictly not needed if controlled)
	if len(fieldPagination) > PaginationColumnWidth {
		fieldPagination = fieldPagination[:PaginationColumnWidth]
	}
	paginationStyled := h.paginationStyle.Render(fieldPagination)
	info := h.renderScrollInfo()
	horizontalPadding := 1
	spacerStyle := lipgloss.NewStyle().Width(horizontalPadding).Render("")
	// Remove safety margin to fill full width as requested
	lineWidth := h.viewport.Width - lipgloss.Width(info) - lipgloss.Width(paginationStyled) - horizontalPadding*2
	if lineWidth < 0 {
		lineWidth = 0
	}
	line := h.lineHeadFootStyle.Render(fmt.Convert("─").Repeat(lineWidth).String())
	// Layout: [Pagination] [Line] [Scroll%]
	return lipgloss.JoinHorizontal(lipgloss.Left, paginationStyled, spacerStyle, line, spacerStyle, info)
}

// renderScrollInfo returns the formatted scroll percentage with fixed width
func (h *DevTUI) renderScrollInfo() string {
	var scrollIcon string

	atTop := h.viewport.AtTop()
	atBottom := h.viewport.AtBottom()

	// Use fixed width of 6 chars for consistency with pagination
	switch {
	case atTop && atBottom:
		scrollIcon = "  ■   " // All content visible (empty square)
	case atTop && !atBottom:
		scrollIcon = "  ▼   " // Can scroll down (down triangle)
	case !atTop && atBottom:
		scrollIcon = "  ▲   " // Can scroll up (up triangle)
	default:
		scrollIcon = " ▼ ▲  " // Can scroll both directions (both arrows)
	}

	return h.footerInfoStyle.Render(scrollIcon)
}

// renderFooterInput renderiza un campo de entrada en el footer
// Si el campo es editable y estamos en modo edición, muestra un cursor en la posición actual
func (h *DevTUI) renderFooterInput() string {
	// Obtener el campo activo
	tabSection := h.TabSections[h.activeTab]

	// Verificar que el índice activo esté en rango
	fieldHandlers := tabSection.fieldHandlers
	if tabSection.indexActiveEditField >= len(fieldHandlers) {
		tabSection.indexActiveEditField = 0 // Reiniciar a 0 si está fuera de rango
	}

	field := fieldHandlers[tabSection.indexActiveEditField]
	info := h.renderScrollInfo()
	horizontalPadding := 1

	// Check if this handler uses expanded footer (Display only)
	if field.isDisplayOnly() {
		// Pagination logic
		currentField := tabSection.indexActiveEditField
		totalFields := len(fieldHandlers)
		if currentField > 99 || totalFields > 99 {
			if h.Logger != nil {
				h.Logger("Field limit exceeded:", currentField, "/", totalFields)
			}
		}
		displayCurrent := min(currentField, 99) + 1 // 1-based for display
		displayTotal := min(totalFields, 99)
		fieldPagination := fmt.Fmt("%2d/%2d", displayCurrent, displayTotal)
		paginationStyled := h.paginationStyle.Render(fieldPagination)
		// Remove safety margin to fill full width
		remainingWidth := h.viewport.Width - lipgloss.Width(info) - lipgloss.Width(paginationStyled) - horizontalPadding*2
		labelText := fmt.Convert(field.getExpandedFooterLabel()).Truncate(remainingWidth-1, 0).String()
		displayStyle := lipgloss.NewStyle().
			Width(remainingWidth).
			Padding(0, horizontalPadding).
			Background(lipgloss.Color(h.Secondary)).
			Foreground(lipgloss.Color(h.Foreground))
		styledLabel := displayStyle.Render(labelText)
		spacerStyle := lipgloss.NewStyle().Width(horizontalPadding).Render("")
		return lipgloss.JoinHorizontal(lipgloss.Left, paginationStyled, spacerStyle, styledLabel, spacerStyle, info)
	}

	// Diferente layout para Edit vs Execution handlers
	if field.isExecutionHandler() {
		// Execution handler: Solo mostrar [Pagination] [Value expandido] [Scroll%]
		// El valor usa todo el espacio disponible, sin label separado

		// Calcular la paginación PRIMERO para incluirla en el cálculo del ancho
		currentField := tabSection.indexActiveEditField
		totalFields := len(fieldHandlers)
		if currentField > 99 || totalFields > 99 {
			if h.Logger != nil {
				h.Logger("Field limit exceeded:", currentField, "/", totalFields)
			}
		}
		displayCurrent := min(currentField, 99) + 1 // 1-based for display
		displayTotal := min(totalFields, 99)
		fieldPagination := fmt.Fmt("%2d/%2d", displayCurrent, displayTotal)
		paginationStyled := h.paginationStyle.Render(fieldPagination)

		// Para execution: el valor usa todo el espacio disponible (sin label separado)
		// Remove safety margin to fill full width
		usedWidth := lipgloss.Width(info) + lipgloss.Width(paginationStyled) + horizontalPadding*2
		valueWidth := h.viewport.Width - usedWidth
		if valueWidth < 10 {
			valueWidth = 10 // Mínimo
		}

		// Preparar el texto del valor (usar label como contenido del valor)
		valueText := field.handler.Label()

		// Truncar el valor para que no afecte el diseño del footer
		textWidth := valueWidth - (horizontalPadding * 2)
		if textWidth < 1 {
			textWidth = 1
		}
		valueText = fmt.Convert(valueText).Truncate(textWidth, 0).String()

		// Definir el estilo para el valor del campo (Execution: Fondo blanco con letras oscuras)
		inputValueStyle := lipgloss.NewStyle().
			Width(valueWidth).
			Padding(0, horizontalPadding).
			Background(lipgloss.Color(h.Foreground)).
			Foreground(lipgloss.Color(h.Background))

		// Renderizar el valor con el estilo adecuado
		styledValue := inputValueStyle.Render(valueText)

		// Crear un estilo para el espacio entre elementos
		spacerStyle := lipgloss.NewStyle().Width(horizontalPadding).Render("")

		// Layout: [Pagination] [Value expandido] [Scroll%]
		return lipgloss.JoinHorizontal(
			lipgloss.Left,
			paginationStyled,
			spacerStyle,
			styledValue,
			spacerStyle,
			info,
		)
	}

	// Normal layout for Edit handlers only: [Pagination] [Label] [Value] [Scroll%]
	labelWidth := h.labelWidth

	// Truncar la etiqueta si es necesario
	labelText := fmt.Convert(field.handler.Label()).Truncate(labelWidth, 0).String()

	// Aplicar el estilo base para garantizar un ancho fijo
	fixedWidthLabel := h.labelStyle.Render(labelText)
	paddedLabel := h.headerTitleStyle.Render(fixedWidthLabel)

	// Calcular la paginación PRIMERO para incluirla en el cálculo del ancho
	currentField := tabSection.indexActiveEditField
	totalFields := len(fieldHandlers)
	if currentField > 99 || totalFields > 99 {
		if h.Logger != nil {
			h.Logger("Field limit exceeded:", currentField, "/", totalFields)
		}
	}
	displayCurrent := min(currentField, 99) + 1 // 1-based for display
	displayTotal := min(totalFields, 99)
	// Force fixed width for alignment
	rawPag := fmt.Fmt("%2d/%2d", displayCurrent, displayTotal)
	fieldPagination := lipgloss.NewStyle().Width(PaginationColumnWidth).Align(lipgloss.Center).Render(rawPag)
	paginationStyled := h.paginationStyle.Render(fieldPagination)

	// Calcular ancho para el valor incluyendo TODOS los elementos: [Pagination] [Label] [Value] [Scroll%]
	// Layout tiene 3 espacios: pagination|space|label|space|value|space|scroll
	// Remove safety margin to fill full width
	usedWidth := lipgloss.Width(info) + lipgloss.Width(paddedLabel) + lipgloss.Width(paginationStyled) + horizontalPadding*3
	valueWidth := h.viewport.Width - usedWidth
	if valueWidth < 10 {
		valueWidth = 10 // Mínimo
	}

	var showCursor bool
	// Preparar el valor del campo
	valueText := field.Value()
	// Usar tempEditValue si existe (modo edición)
	if field.tempEditValue != "" {
		valueText = field.tempEditValue
	}

	// Truncar el valor para que no afecte el diseño del footer
	// Descontar el padding que se aplicará al estilo
	textWidth := valueWidth - (horizontalPadding * 2)

	// Mostrar cursor solo si estamos en modo edición y el campo es editable
	showCursor = false
	if h.editModeActivated && field.editable() {
		showCursor = true
	}

	// Determine text limit to avoid layout shifts (reserve space for cursor)
	// IMPORTANT: Reserve space ALWAYS when showCursor is true, not just when cursorVisible
	// This prevents layout shift during cursor blinking
	textLimit := textWidth
	if showCursor {
		textLimit -= 1 // Reserve space for cursor character (visible or invisible)
	}
	if textLimit < 1 {
		textLimit = 1
	}

	// Calculate the visible part of the text using the viewport
	truncated, cursorInView := field.viewport.CalculateVisibleWindow(valueText, field.cursor, textLimit)

	// Definir el estilo para el valor del campo
	inputValueStyle := lipgloss.NewStyle().
		Width(valueWidth).
		Padding(0, horizontalPadding)

	// Aplicar estilos para Edit handlers según el estado
	if h.editModeActivated && field.editable() {
		// Edit en modo edición activa
		inputValueStyle = inputValueStyle.
			Background(lipgloss.Color(h.Secondary)).
			Foreground(lipgloss.Color(h.Foreground))
	} else {
		// Edit en modo no edición
		inputValueStyle = inputValueStyle.
			Background(lipgloss.Color(h.Secondary)).
			Background(lipgloss.Color(h.Secondary)).
			Foreground(lipgloss.Color(h.Background))
	}

	// Añadir cursor si corresponde
	if showCursor {
		runes := []rune(truncated)
		cursorPos := cursorInView
		if cursorPos < 0 {
			cursorPos = 0
		}
		if cursorPos > len(runes) {
			cursorPos = len(runes)
		}

		// Insertar el cursor en la posición correcta dentro de la cadena truncada (ventana visible)
		beforeCursor := string(runes[:cursorPos])
		afterCursor := string(runes[cursorPos:])

		if h.cursorVisible {
			valueText = beforeCursor + "▋" + afterCursor
		} else {
			// When cursor is invisible (blinking off), use a space to maintain consistent width
			valueText = beforeCursor + " " + afterCursor
		}
	} else {
		// Use the truncated text without cursor
		valueText = truncated
	}

	// Renderizar el valor con el estilo adecuado
	styledValue := inputValueStyle.Render(valueText)

	// Crear un estilo para el espacio entre elementos
	spacerStyle := lipgloss.NewStyle().Width(horizontalPadding).Render("")

	// Layout: [Pagination] [Label] [Value] [Scroll%]
	return lipgloss.JoinHorizontal(
		lipgloss.Left,
		paginationStyled,
		spacerStyle,
		paddedLabel,
		spacerStyle,
		styledValue,
		spacerStyle,
		info,
	)
}
