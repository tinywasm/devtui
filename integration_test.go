package devtui

import (
	"sync"
	"testing"
	"time"
)

// TestRealWorldScenario simula el escenario exacto que causaba el error original
func TestRealWorldScenario(t *testing.T) {
	// Exactamente la misma configuración que en main.go
	tui := NewTUI(&TuiConfig{
		AppName:  "Ejemplo DevTUI",
		ExitChan: make(chan bool),
		Color: &ColorPalette{
			Foreground: "#F4F4F4",
			Background: "#000000",
			Primary:    "#FF6600",
			Secondary:  "#666666",
		},
		Logger: func(messages ...any) {
			t.Logf("Log: %v", messages)
		},
	})

	// Configurar la sección y los campos exactamente como en main.go
	nombreHandler := NewTestEditableHandler("Nombre", "")
	edadHandler := NewTestEditableHandler("Edad", "")
	emailHandler := NewTestEditableHandler("Email", "")

	tab := tui.NewTabSection("Datos personales", "Información básica")
	tui.AddHandler(nombreHandler, "", tab)
	tui.AddHandler(edadHandler, "", tab)
	tui.AddHandler(emailHandler, "", tab)

	// Asegurarnos de que no hay panic durante la inicialización
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("No se esperaba ningún panic, pero se obtuvo: %v", r)
		}
	}()

	// Usar un WaitGroup como en el ejemplo original
	var wg sync.WaitGroup
	wg.Add(1)

	// Simular la inicialización de la TUI en una goroutine separada
	go func() {
		defer wg.Done()

		// En lugar de inicializar la TUI completa (que bloquearía el test),
		// simplemente simulamos las operaciones que causaban el problema

		// Simular que se presiona Enter en un campo
		section := tui.TabSections[0] // Primera sección

		// Esto era lo que causaba el panic original
		section.addNewContent(0, "Test content from real scenario")

		t.Log("Operación completada sin panic")
	}()

	// Esperar un momento para que la goroutine termine
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("Test completado exitosamente")
	case <-time.After(2 * time.Second):
		t.Error("El test tardó demasiado en completarse")
	}
}
