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
	fieldPagination := fmt.Fmt("%2d/%2d", displayCurrent, displayTotal)
	paginationStyled := h.paginationStyle.Render(fieldPagination)
	info := h.renderScrollInfo()
	horizontalPadding := 1
	spacerStyle := lipgloss.NewStyle().Width(horizontalPadding).Render("")
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

	switch {
	case atTop && atBottom:
		scrollIcon = " ■ " // All content visible (empty square)
	case atTop && !atBottom:
		scrollIcon = " ▼ " // Can scroll down (down triangle)
	case !atTop && atBottom:
		scrollIcon = " ▲ " // Can scroll up (up triangle)
	default:
		scrollIcon = "▼ ▲" // Can scroll both directions (both arrows)
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
	labelText := fmt.Convert(field.handler.Label()).Truncate(labelWidth-1, 0).String()

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
	fieldPagination := fmt.Fmt("%2d/%2d", displayCurrent, displayTotal)
	paginationStyled := h.paginationStyle.Render(fieldPagination)

	// Calcular ancho para el valor incluyendo TODOS los elementos: [Pagination] [Label] [Value] [Scroll%]
	// Layout tiene 3 espacios: pagination|space|label|space|value|space|scroll
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
	if textWidth < 1 {
		textWidth = 1
	}
	valueText = fmt.Convert(valueText).Truncate(textWidth, 0).String()

	// Mostrar cursor solo si estamos en modo edición y el campo es editable
	if h.editModeActivated && field.editable() {
		showCursor = true
	}

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
			Foreground(lipgloss.Color(h.Background))
	}

	// Añadir cursor si corresponde
	if showCursor {
		// Asegurar que el cursor está dentro de los límites
		runes := []rune(field.tempEditValue)
		if field.cursor < 0 {
			field.cursor = 0
		}
		if field.cursor > len(runes) {
			field.cursor = len(runes)
		}

		// Insertar el cursor en la posición correcta
		if field.cursor <= len(runes) {
			beforeCursor := string(runes[:field.cursor])
			afterCursor := string(runes[field.cursor:])
			valueText = beforeCursor + "▋" + afterCursor
		} else {
			valueText = field.tempEditValue + "▋"
		}
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
