package devtui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestFooterView verifica el comportamiento del renderizado del footer
func TestFooterView(t *testing.T) {
	h := DefaultTUIForTest(func(messages ...any) {
		// Test logger - do nothing
	})

	// Caso 1: Tab sin fields debe mostrar el scrollbar estándar
	t.Run("Footer with no fields shows scrollbar", func(t *testing.T) {
		// Guardar estado actual para restaurar después de la prueba
		tab := h.TabSections[h.activeTab]
		originalFields := tab.fieldHandlers

		// Configurar pestaña sin fields
		tab.setFieldHandlers([]*field{})

		// Renderizar footer
		result := h.footerView()

		// Verificar que contiene indicador de scroll con iconos (no porcentaje)
		hasScrollIcon := strings.Contains(result, "■") ||
			strings.Contains(result, "▼") ||
			strings.Contains(result, "▲")
		if !hasScrollIcon {
			t.Error("El footer sin campos debería mostrar indicador de scroll con iconos (■, ▼, ▲)")
		}

		// Restaurar estado
		tab.setFieldHandlers(originalFields)
	})

	// Caso 2: Tab con fields debe mostrar el campo actual como input (ahora siempre, no solo en modo edición)
	t.Run("Footer with fields shows field as input even when not editing", func(t *testing.T) {

		// Crear un nuevo field con handler para la prueba
		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{})
		testHandler := NewTestEditableHandler("TestLabel", "TestValue Rendered")
		h.AddHandler(testHandler, "", tab)

		// Set viewport width for proper layout calculation
		h.viewport.Width = 80

		// Desactivar modo edición para verificar que aún así se muestra el campo
		h.editModeActivated = false
		tabSection := h.TabSections[h.activeTab]
		tabSection.indexActiveEditField = 0

		// Renderizar footer
		result := h.footerView()

		// Verificar que contiene la etiqueta y valor del field
		field := tab.fieldHandlers[0]
		if !strings.Contains(result, field.Value()) {
			t.Errorf("El footer debería mostrar:\n%v\n incluso sin estar en modo edición, pero muestra:\n%s\n", field.Value(), result)
		}
	})
}

// TestRenderFooterInput verifica el comportamiento específico del renderizado del input
func TestRenderFooterInput(t *testing.T) {
	// Caso 1: Campo editable en modo edición debe mostrar cursor
	t.Run("editable field in edit mode shows cursor", func(t *testing.T) {
		h := DefaultTUIForTest(func(messages ...any) {})
		h.editModeActivated = true
		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{})
		testHandler := NewTestEditableHandler("Test", "test value")
		h.AddHandler(testHandler, "", tab)

		// Set viewport width for proper layout calculation
		h.viewport.Width = 80

		field := tab.fieldHandlers[0]
		field.setCursorForTest(2) // Cursor en posición 's': te[s]t value
		field.setTempEditValueForTest("test value")

		// Ensure cursor is visible for this test
		h.cursorVisible = true

		// Renderizar input
		result := h.renderFooterInput()

		// Con cursor Overlay, ya no buscamos "te▋st value"
		// Sino que buscamos el carácter 's' resaltado (con estilo invertido)
		char := "s"
		expectedRender := lipgloss.NewStyle().
			Background(lipgloss.Color(h.Foreground)).
			Foreground(lipgloss.Color(h.Secondary)).
			Render(char)

		if !strings.Contains(result, expectedRender) {
			t.Errorf("El cursor overlay para el carácter 's' no se encuentra en el resultado")
		}
	})

	// Caso 2: Campo no editable no debe mostrar cursor
	t.Run("Non-editable field doesn't show cursor", func(t *testing.T) {
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		// Limpiar todos los handlers existentes y crear explícitamente un handler no editable
		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{}) // Limpiar cualquier handler por defecto

		// Crear explícitamente un handler no editable (ExecutionHandler)
		testHandler := NewTestNonEditableHandler("Test", "Value")
		h.AddHandler(testHandler, "", tab)

		h.editModeActivated = true
		h.TabSections[h.activeTab].indexActiveEditField = 0

		// Renderizar input
		result := h.renderFooterInput()

		// No debe contener cursor porque es un handler de ejecución (no editable)
		if strings.Contains(result, "▋") {
			t.Error("Campo no editable no debería mostrar cursor")
		}
	})

	// Nuevo test - Caso 4: Verificar que se maneja correctamente el índice fuera de rango
	t.Run("Index out of range is handled correctly", func(t *testing.T) {
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		expectedLabel := "Test Handler"
		// Configurar un índice activo fuera de rango
		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{})
		testHandler := NewTestNonEditableHandler(expectedLabel, "Some Value")
		h.AddHandler(testHandler, "", tab)

		// Set viewport width for proper layout calculation
		h.viewport.Width = 80

		h.TabSections[h.activeTab].indexActiveEditField = 5 // Índice fuera de rango

		// Renderizar - no debería producir pánico
		result := h.renderFooterInput()

		// Verificar que se resetea el índice y se muestra el primer campo
		// Para handlers de ejecución, se muestra el Label() en el footer
		if !strings.Contains(result, expectedLabel) {
			t.Fatalf("No se manejó correctamente el índice fuera de rango. Esperado: %q, resultado:\n%s", expectedLabel, result)
		}
	})

	// Nuevo test - Caso 5: Verificar el estilo correcto cuando está seleccionado pero no en modo edición
	t.Run("Field has correct style when selected but not in edit mode", func(t *testing.T) {
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{})
		testHandler := NewTestEditableHandler("Test", "Value")
		h.AddHandler(testHandler, "", tab)
		h.TabSections[h.activeTab].indexActiveEditField = 0
		h.editModeActivated = false // No en modo edición

		// El estilo debe ser fieldSelectedStyle en vez de fieldEditingStyle
		originalFieldSelectedStyle := h.fieldSelectedStyle
		originalFieldEditingStyle := h.fieldEditingStyle

		// Modificar temporalmente los estilos para distinguirlos claramente
		h.fieldSelectedStyle = h.fieldSelectedStyle.Background(lipgloss.Color("blue"))
		h.fieldEditingStyle = h.fieldEditingStyle.Background(lipgloss.Color("red"))

		result := h.renderFooterInput()

		// Restaurar estilos originales
		h.fieldSelectedStyle = originalFieldSelectedStyle
		h.fieldEditingStyle = originalFieldEditingStyle

		// Verificar que no contiene el cursor de edición
		if strings.Contains(result, "▋") {
			t.Error("Campo seleccionado pero no en modo edición no debería mostrar cursor")
		}
	})
}

// Nuevos tests para la navegación y comportamiento de teclas
func TestInputNavigation(t *testing.T) {
	h := DefaultTUIForTest(func(messages ...any) {
		// Test logger - do nothing
	})

	// Configurar múltiples campos para prueba de navegación
	tab := h.TabSections[h.activeTab]
	tab.setFieldHandlers([]*field{})
	testHandler1 := NewTestEditableHandler("Field1", "Value1")
	testHandler2 := NewTestEditableHandler("Field2", "Value2")
	testHandler3 := NewTestEditableHandler("Field3", "Value3")
	h.AddHandler(testHandler1, "", tab)
	h.AddHandler(testHandler2, "", tab)
	h.AddHandler(testHandler3, "", tab)
	h.TabSections[h.activeTab].indexActiveEditField = 0
	h.editModeActivated = false

	t.Run("Right key navigates to next field", func(t *testing.T) {
		// Simular pulsación de tecla derecha
		_, _ = h.handleNormalModeKeyboard(tea.KeyMsg{Type: tea.KeyRight})

		// Verificar que nos movimos al siguiente campo
		if h.TabSections[h.activeTab].indexActiveEditField != 1 {
			t.Errorf("La tecla derecha debería navegar al siguiente campo, pero se quedó en: %d",
				h.TabSections[h.activeTab].indexActiveEditField)
		}
	})

	t.Run("Left key navigates to previous field", func(t *testing.T) {
		// Nos aseguramos de estar en el campo del medio
		h.TabSections[h.activeTab].indexActiveEditField = 1

		// Simular pulsación de tecla izquierda
		_, _ = h.handleNormalModeKeyboard(tea.KeyMsg{Type: tea.KeyLeft})

		// Verificar que nos movimos al campo anterior
		if h.TabSections[h.activeTab].indexActiveEditField != 0 {
			t.Errorf("La tecla izquierda debería navegar al campo anterior, pero se quedó en: %d",
				h.TabSections[h.activeTab].indexActiveEditField)
		}
	})

	t.Run("Cyclical navigation wraps around at boundaries", func(t *testing.T) {
		// Ir al primer campo
		h.TabSections[h.activeTab].indexActiveEditField = 0

		// Simular pulsación de tecla izquierda (debe ir al último campo)
		_, _ = h.handleNormalModeKeyboard(tea.KeyMsg{Type: tea.KeyLeft})

		// Verificar que se movió al último campo
		if h.TabSections[h.activeTab].indexActiveEditField != 2 {
			t.Errorf("La navegación cíclica debería ir al último campo, pero está en: %d",
				h.TabSections[h.activeTab].indexActiveEditField)
		}

		// Simular pulsación de tecla derecha (debe volver al primer campo)
		_, _ = h.handleNormalModeKeyboard(tea.KeyMsg{Type: tea.KeyRight})

		// Verificar que volvió al primer campo
		if h.TabSections[h.activeTab].indexActiveEditField != 0 {
			t.Errorf("La navegación cíclica debería volver al primer campo, pero está en: %d",
				h.TabSections[h.activeTab].indexActiveEditField)
		}
	})

	t.Run("Enter enters edit mode", func(t *testing.T) {
		// Reset para esta prueba
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		// Configurar un campo editable
		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{})
		testHandler := NewTestEditableHandler("Test", "Value")
		h.AddHandler(testHandler, "", tab)

		// Asegurar que no estamos en modo edición
		h.editModeActivated = false
		h.TabSections[h.activeTab].indexActiveEditField = 0

		// Simular pulsación de Enter
		_, _ = h.handleNormalModeKeyboard(tea.KeyMsg{Type: tea.KeyEnter})

		// Verificar que entramos en modo edición
		if !h.editModeActivated {
			t.Error("Enter debería activar el modo edición")
		}
	})

	t.Run("Esc exits edit mode", func(t *testing.T) {
		// Reset para esta prueba
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})

		// Configurar un campo editable
		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{})
		testHandler := NewTestEditableHandler("Test", "Value")
		h.AddHandler(testHandler, "", tab)

		// Asegurar que estamos en modo edición
		h.editModeActivated = true
		h.TabSections[h.activeTab].indexActiveEditField = 0

		// Simular pulsación de Esc
		_, _ = h.handleEditingConfigKeyboard(tea.KeyMsg{Type: tea.KeyEscape})

		// Verificar que salimos del modo edición
		if h.editModeActivated {
			t.Error("Esc debería salir del modo edición")
		}
	})

	t.Run("Left/right moves cursor in edit mode", func(t *testing.T) {
		// Reset para esta prueba y configurar un campo editable
		h := DefaultTUIForTest(func(messages ...any) {
			// Test logger - do nothing
		})
		tab := h.TabSections[h.activeTab]
		tab.setFieldHandlers([]*field{})
		testHandler := NewTestEditableHandler("Test", "Value1")
		h.AddHandler(testHandler, "", tab)

		// Configurar para edición
		h.editModeActivated = true
		h.TabSections[h.activeTab].indexActiveEditField = 0
		field := tab.fieldHandlers[0]
		field.setCursorAtEnd()
		// Move cursor to position 3 for test
		field.setCursorForTest(3)

		// Simular pulsación de tecla izquierda
		_, _ = h.handleEditingConfigKeyboard(tea.KeyMsg{Type: tea.KeyLeft})

		// Verificar que el cursor se movió a la izquierda
		if field.cursor != 2 {
			t.Errorf("La tecla izquierda en modo edición debería mover el cursor a la izquierda, pero quedó en: %d",
				field.cursor)
		}

		// Simular pulsación de tecla derecha
		_, _ = h.handleEditingConfigKeyboard(tea.KeyMsg{Type: tea.KeyRight})

		// Verificar que el cursor volvió a la posición original
		if field.cursor != 3 {
			t.Errorf("La tecla derecha en modo edición debería mover el cursor a la derecha, pero quedó en: %d",
				field.cursor)
		}
	})
}

func TestCursorNoExtraSpace(t *testing.T) {
	h := DefaultTUIForTest(func(messages ...any) {})
	h.viewport.Width = 80
	h.editModeActivated = true

	tab := h.TabSections[h.activeTab]
	tab.setFieldHandlers([]*field{})

	// Texto original tiene largo 5
	originalText := "value"
	testHandler := NewTestEditableHandler("Test", originalText)
	h.AddHandler(testHandler, "", tab)

	f := tab.fieldHandlers[0]
	f.setTempEditValueForTest(originalText)
	f.setCursorForTest(3) // Cursor en medio: val|ue

	// Render with cursor invisible (blinking off)
	h.cursorVisible = false
	result := h.renderFooterInput()

	// El bug actual inserta un espacio: "val ue" (largo 6)
	// El comportamiento deseado (Overlay) mantendrá "value" (largo 5)

	if strings.Contains(result, "val ue") {
		t.Errorf("DEBUG: Se detectó el bug del espacio extra ('val ue')")
	}

	if !strings.Contains(result, originalText) {
		t.Errorf("El texto original %q debería estar presente sin espacios extra en medio", originalText)
	}
}

func TestCursorNoTrail(t *testing.T) {
	// Forzar perfil de color para que los tests sean consistentes
	lipgloss.SetColorProfile(termenv.TrueColor)

	h := DefaultTUIForTest(func(messages ...any) {})
	h.viewport.Width = 80
	h.editModeActivated = true
	h.cursorVisible = true

	tab := h.TabSections[h.activeTab]
	tab.setFieldHandlers([]*field{})
	testHandler := NewTestEditableHandler("Path", "abcdef")
	h.AddHandler(testHandler, "", tab)

	f := tab.fieldHandlers[0]
	f.setTempEditValueForTest("abcdef")
	f.setCursorForTest(2) // Cursor en 'c': ab[c]def

	result := h.renderFooterInput()

	// 1. Verificar que el texto 'def' está presente
	if !strings.Contains(result, "def") {
		t.Errorf("El texto 'def' después del cursor debería estar visible")
	}

	// 2. Verificar que NO hay trail (fondo negro después del cursor)
	// El trail ocurre cuando hay un reset ANSI (\x1b[0m) seguido directamente por texto sin estilo
	// Buscamos la secuencia: cursor + reset + texto sin estilo
	cursorRender := lipgloss.NewStyle().
		Background(lipgloss.Color(h.Foreground)).
		Foreground(lipgloss.Color(h.Secondary)).
		Render("c")

	// Si hay trail, el patrón sería: cursorRender + "def" (sin estilo intermedio)
	// Con el fix correcto, "def" está dentro del contenedor con fondo correcto
	if strings.Contains(result, cursorRender+"def") {
		t.Errorf("TRAIL DETECTADO: El texto posterior 'def' no tiene fondo (hereda reset del cursor)")
	}

	// 3. Verificar que todo el texto está dentro del mismo contenedor (sin trail visual)
	// El ancho del resultado debe ser consistente independientemente de la posición del cursor
	h.cursorVisible = false
	resultNoCursor := h.renderFooterInput()

	if lipgloss.Width(result) != lipgloss.Width(resultNoCursor) {
		t.Errorf("El ancho del input no es consistente: conCursor=%d, sinCursor=%d",
			lipgloss.Width(result), lipgloss.Width(resultNoCursor))
	}
}
