package devtui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestInputBackgroundConsistency verifica que el fondo del input sea consistente con h.Secondary
func TestInputBackgroundConsistency(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	h := DefaultTUIForTest(func(messages ...any) {})
	h.viewport.Width = 80
	h.editModeActivated = true
	h.cursorVisible = true

	tab := h.TabSections[h.activeTab]
	tab.setFieldHandlers([]*field{})
	testHandler := NewTestEditableHandler("Path", "myapp")
	h.AddHandler(testHandler, 0, "", tab)

	f := tab.fieldHandlers[0]
	f.setTempEditValueForTest("myapp")
	f.setCursorForTest(5) // Cursor al final

	result := h.renderFooterInput()

	// El fondo del input DEBE ser h.Secondary (del DefaultPalette)
	// Verificamos que el texto "myapp" tenga el fondo correcto
	expectedBgStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(h.Secondary)).
		Foreground(lipgloss.Color(h.Foreground)).
		Render("myapp")

	if !strings.Contains(result, expectedBgStyle) {
		t.Errorf("El fondo del input no tiene el color Secondary esperado (%s)", h.Secondary)
		t.Logf("Esperado: %q", expectedBgStyle)
		t.Logf("Resultado parcial: %q", result[:min(len(result), 300)])
	}
}

// TestInputWidthInEditMode verifica que el input ocupe todo el ancho disponible
func TestInputWidthInEditMode(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	h := DefaultTUIForTest(func(messages ...any) {})
	h.viewport.Width = 80
	h.editModeActivated = true
	h.cursorVisible = false // cursor apagado para medir solo texto

	tab := h.TabSections[h.activeTab]
	tab.setFieldHandlers([]*field{})
	testHandler := NewTestEditableHandler("Path", "myapp")
	h.AddHandler(testHandler, 0, "", tab)

	f := tab.fieldHandlers[0]
	f.setTempEditValueForTest("myapp")
	f.setCursorForTest(5)

	// Render con cursor (modo edición)
	resultEdit := h.renderFooterInput()
	widthEdit := lipgloss.Width(resultEdit)

	// Render sin cursor (modo normal simulado)
	h.editModeActivated = false
	resultNormal := h.renderFooterInput()
	widthNormal := lipgloss.Width(resultNormal)

	// Ambos anchos deben ser iguales
	if widthEdit != widthNormal {
		t.Errorf("El ancho del input cambia entre modos: edit=%d, normal=%d", widthEdit, widthNormal)
	}
}

// TestCursorBlinkSingleColor verifica que el cursor use colores consistentes durante el parpadeo
func TestCursorBlinkSingleColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	h := DefaultTUIForTest(func(messages ...any) {})
	h.viewport.Width = 80
	h.editModeActivated = true

	tab := h.TabSections[h.activeTab]
	tab.setFieldHandlers([]*field{})
	testHandler := NewTestEditableHandler("Path", "myapp")
	h.AddHandler(testHandler, 0, "", tab)

	f := tab.fieldHandlers[0]
	f.setTempEditValueForTest("myapp")
	f.setCursorForTest(2) // Cursor en 'a': my[a]pp

	// Render con cursor visible
	h.cursorVisible = true
	resultOn := h.renderFooterInput()

	// Render con cursor invisible (parpadeo apagado)
	h.cursorVisible = false
	resultOff := h.renderFooterInput()

	// Cuando el cursor está visible, debe haber exactamente 1 carácter con fondo invertido
	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(h.Foreground)).
		Foreground(lipgloss.Color(h.Secondary)).
		Render("a")

	if !strings.Contains(resultOn, cursorStyle) {
		t.Errorf("Cursor visible: el carácter 'a' no tiene el estilo invertido esperado")
		t.Logf("Esperado: %q", cursorStyle)
	}

	// Cuando el cursor está apagado, no debería haber estilo invertido en 'a'
	if strings.Contains(resultOff, cursorStyle) {
		t.Errorf("Cursor apagado: el carácter 'a' todavía tiene el estilo invertido (debería ser normal)")
	}

	// Verificar que "myapp" aparece completo cuando cursor está apagado
	if !strings.Contains(resultOff, "myapp") {
		t.Errorf("Cursor apagado: el texto 'myapp' debería estar visible completo")
	}
}
