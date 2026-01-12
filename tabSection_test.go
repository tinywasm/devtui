package devtui

import (
	"fmt"
	"testing"
	"time"

	. "github.com/tinywasm/fmt"
)

func TestTabSectionWriter(t *testing.T) {

	config := &TuiConfig{
		ExitChan: make(chan bool),
		Color:    &ColorPalette{}, // Usando un ColorPalette vacío
		Logger: func(messages ...any) {
			// Mock function for logging
		},
	}

	tui := NewTUI(config)

	// Enable test mode for synchronous execution
	tui.SetTestMode(true)

	// Crear tab section de prueba
	tab := tui.NewTabSection("TEST", "")

	// Usar handlerWriter en vez de tabSection como Writer
	tabSection := tab.(*tabSection)
	handlerWriter := &handlerWriter{tabSection: tabSection, handlerName: ""}
	testMsg := "Mensaje de prueba"
	n, err := fmt.Fprintln(handlerWriter, testMsg)
	if err != nil {
		t.Fatalf("Error escribiendo en el Writer: %v", err)
	}
	if n != len(testMsg)+1 { // +1 por el newline
		t.Errorf("Bytes escritos incorrectos: esperado %d, obtenido %d", len(testMsg)+1, n)
	}

	// Verificar que el mensaje llegó al canal
	select {
	case msg := <-tui.tabContentsChan:
		if msg.Content != testMsg {
			t.Errorf("Contenido incorrecto: esperado '%s', obtenido '%s'", testMsg, msg.Content)
		}
		if msg.Type != 0 { // 0 es el tipo por defecto para mensajes normales
			t.Errorf("Tipo de mensaje incorrecto: esperado 0, obtenido %v", msg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout: el mensaje no llegó al canal")
	}
}

func TestTabContentsIncrementWhenSendingMessages(t *testing.T) {

	config := &TuiConfig{
		ExitChan: make(chan bool),
		Color:    &ColorPalette{}, // Usando un ColorPalette vacío
		Logger: func(messages ...any) {
			// Mock function for logging
		},
	}

	tui := NewTUI(config)

	// Enable test mode for synchronous execution
	tui.SetTestMode(true)

	tab := tui.NewTabSection("TEST", "")

	// Usar handlerWriter en vez de tabSection como Writer
	tabSection := tab.(*tabSection)
	handlerWriter := &handlerWriter{tabSection: tabSection, handlerName: ""}

	// Test messages with different types and prefixes for detection
	messages := []struct {
		rawText      string // Text sent via Fprintln
		expectedText string // Text stored in tabContents (might be same or trimmed)
		expectedType MessageType
	}{
		{"First message", "First message", Msg.Normal}, // Normal message
		{"INFO: Second message", "INFO: Second message", Msg.Info},
		{"ERROR: Third message", "ERROR: Third message", Msg.Error},
		{"WARNING: Fourth message", "WARNING: Fourth message", Msg.Warning},
		{"Fifth message", "Fifth message", Msg.Normal}, // Normal message again
	}

	// Send messages and verify increment
	for i, message := range messages {
		// Send message using the raw text
		_, err := fmt.Fprintln(handlerWriter, message.rawText)
		if err != nil {
			t.Fatalf("Error writing message %d ('%s'): %v", i+1, message.rawText, err)
		}

		// Verify that the message arrived to the channel (sent by sendMessage)
		// AND that tabContents was updated correctly (also by sendMessage)
		select {
		case msg := <-tui.tabContentsChan:
			// Verify content received from channel matches expected stored text
			if msg.Content != message.expectedText {
				t.Errorf("Message %d: incorrect content from channel. Expected '%s', got '%s'",
					i+1, message.expectedText, msg.Content)
			}

			// Verify type received from channel matches expected type
			if msg.Type != message.expectedType {
				t.Errorf("Message %d: incorrect type from channel. Expected %v, got %v",
					i+1, message.expectedType, msg.Type)
			}

			// Verify that tabContents has the correct amount (should be updated by sendMessage)
			if len(tabSection.tabContents) != i+1 {
				t.Errorf("Message %d: incorrect amount in tabContents. Expected %d, got %d",
					i+1, i+1, len(tabSection.tabContents))
			}

			// Verify that the last message added to the slice matches
			if len(tabSection.tabContents) > 0 { // Check bounds
				last := tabSection.tabContents[len(tabSection.tabContents)-1]
				if last.Content != message.expectedText || last.Type != message.expectedType {
					t.Errorf("Message %d: last record in slice does not match. Expected ('%s', %v), got ('%s', %v)",
						i+1, message.expectedText, message.expectedType, last.Content, last.Type)
				}
			} else {
				t.Errorf("Message %d: tabContents is empty after message should have been added", i+1)
			}

		case <-time.After(1 * time.Second):
			t.Fatalf("Timeout: message %d ('%s') did not arrive to channel", i+1, message.rawText)
		}
	}

	// Final verification of all messages
	if len(tabSection.tabContents) != len(messages) {
		t.Fatalf("Incorrect final amount. Expected %d, got %d",
			len(messages), len(tabSection.tabContents))
	}

	// Verify message order
	for i, message := range messages {
		if tabSection.tabContents[i].Content != message.expectedText {
			t.Errorf("Incorrect order in message %d. Expected '%s', got '%s'",
				i+1, message.expectedText, tabSection.tabContents[i].Content)
		}
	}
}
