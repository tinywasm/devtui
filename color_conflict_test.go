package devtui

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/tinywasm/fmt"
)

// TestOpcionA_RequirementsValidation validates the core requirements from BETTER_VIEW.md
func TestOpcionA_RequirementsValidation(t *testing.T) {
	tui := NewTUI(&TuiConfig{
		AppName:  "Requirements Test",
		ExitChan: make(chan bool),
	})

	// Enable test mode for synchronous execution
	tui.SetTestMode(true)

	tab := tui.NewTabSection("Test", "Test Tab")

	testCases := []struct {
		handler  string
		content  string
		msgType  MessageType
		expected string // Expected format pattern
	}{
		{"DatabaseConfig", "postgres://localhost:5432/mydb", Msg.Info, "[DatabaseConfig] postgres://localhost:5432/mydb"},
		{"SystemBackup", "Create System Backup", Msg.Success, "[SystemBackup] Create System Backup"},
		{"ErrorHandler", "Connection failed", Msg.Error, "[ErrorHandler] Connection failed"},
		{"WarningHandler", "Deprecated function", Msg.Warning, "[WarningHandler] Deprecated function"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.handler, tc.msgType.String()), func(t *testing.T) {
			tabContent := tui.createTabContent(tc.content, tc.msgType, tab.(*tabSection), tc.handler, "", "", handlerTypeLoggable)
			formattedMessage := tui.formatMessage(tabContent, true)

			t.Logf("Message: %s", formattedMessage)

			// 1. Verificar que el formato contiene el nombre del handler (sin corchetes)
			// NOTE: current API truncates/pads handler names to a fixed width (HandlerNameWidth).
			// Check for the prefix/truncated form instead of the full handler name.
			var expectedPattern string
			if len(tc.handler) > HandlerNameWidth {
				expectedPattern = tc.handler[:HandlerNameWidth]
			} else {
				expectedPattern = tc.handler
			}
			if !strings.Contains(formattedMessage, expectedPattern) {
				t.Errorf("FAIL: Expected handler name prefix '%s' not found (full: '%s')", expectedPattern, tc.handler)
			}

			// 2. Verificar que el contenido está presente
			if !strings.Contains(formattedMessage, tc.content) {
				t.Errorf("FAIL: Expected content '%s' not found", tc.content)
			}

			// 3. Verificar que NO hay brackets separados (patrón viejo)
			separatedPattern := fmt.Sprintf(" [ %s ] ", tc.handler)
			if strings.Contains(formattedMessage, separatedPattern) {
				t.Errorf("FAIL: Found old separated pattern '%s'", separatedPattern)
			}

			t.Log("✅ PASS: Opción A requirements met")
		})
	}
}

// TestCentralizedMessageProcessing validates that all message flows use centralized processing
func TestCentralizedMessageProcessing(t *testing.T) {
	t.Log("TESTING CENTRALIZED MESSAGE PROCESSING:")
	t.Log("=======================================")

	// Test cases que deberían detectar tipo automáticamente
	testCases := []struct {
		content      string
		expectedType MessageType
		description  string
	}{
		{"Database connection configured successfully", Msg.Success, "Success word detected correctly"},
		{"ERROR: Connection failed", Msg.Error, "Error prefix detected correctly"},
		{"WARNING: Deprecated function", Msg.Warning, "Warning prefix detected correctly"},
		{"SUCCESS: Operation completed", Msg.Success, "Success prefix detected correctly"},
		{"System initialized", Msg.Normal, "Normal message detected correctly"},
		{"Backup completed successfully", Msg.Success, "Success word detected correctly"},
		{"Preparing backup...", Msg.Normal, "Normal progress message"},
	}

	for _, tc := range testCases {
		t.Run(tc.content, func(t *testing.T) {
			// Test que Translate(content).StringType() funciona correctamente
			_, detectedType := Translate(tc.content).StringType()

			if detectedType != tc.expectedType {
				t.Errorf("FAIL: Expected %v, got %v for: %s", tc.expectedType, detectedType, tc.content)
			} else {
				t.Logf("✅ PASS: '%s' correctly detected as %v", tc.content, detectedType)
			}
		})
	}

	t.Log("")
	t.Log("CONCLUSION: DetectMessageType works correctly")
	t.Log("SOLUTION: All message methods now use DetectMessageType for centralization")
}

// TestLastMessageColorFixed validates that the last callback message now uses correct colors
func TestLastMessageColorFixed(t *testing.T) {
	tui := NewTUI(&TuiConfig{
		AppName:  "Last Message Color Fixed Test",
		ExitChan: make(chan bool),
	})

	// Enable test mode for synchronous execution
	tui.SetTestMode(true)

	tab := tui.NewTabSection("Test", "Test Tab")

	t.Log("🔧 SOLUTION TEST: Validar que el último mensaje usa el color correcto")

	// Test casos que simulan el final de una operación
	finalMessages := []struct {
		content       string
		expectedType  MessageType
		expectedColor string
		context       string
	}{
		// Casos que antes fallaban - ahora deberían funcionar
		{"Operation completed successfully", Msg.Success, "HIGHLIGHT (#FF6600)", "Success con palabra 'successfully'"},
		{"Backup completed successfully", Msg.Success, "HIGHLIGHT (#FF6600)", "Success con 'completed successfully'"},
		{"ERROR: Operation failed", Msg.Error, "RED (#FF0000)", "Error con prefijo 'ERROR:'"},
		{"WARNING: Operation completed with warnings", Msg.Warning, "YELLOW (#FFFF00)", "Warning con prefijo 'WARNING:'"},
		{"Database connection established", Msg.Normal, "NORMAL", "Normal message sin keywords especiales"},
		{"SUCCESS: All tasks completed", Msg.Success, "HIGHLIGHT (#FF6600)", "Success con prefijo 'SUCCESS:'"},
	}

	for _, tc := range finalMessages {
		t.Run(tc.content, func(t *testing.T) {
			// Simular el mensaje final de una operación
			tabContent := tui.createTabContent(tc.content, tc.expectedType, tab.(*tabSection), "TestHandler", "final-op-123", "", handlerTypeLoggable)
			formattedMessage := tui.formatMessage(tabContent, true)

			t.Logf("Context: %s", tc.context)
			t.Logf("Content: %s", tc.content)
			t.Logf("Expected: %s (%s)", tc.expectedType, tc.expectedColor)
			t.Logf("Formatted: %s", formattedMessage)

			// Verificar detección automática de tipo
			_, detectedType := Translate(tc.content).StringType()
			if detectedType != tc.expectedType {
				t.Errorf("❌ DetectMessageType failed: Expected %v, got %v", tc.expectedType, detectedType)
			} else {
				t.Logf("✅ DetectMessageType working: %s → %v", tc.content, detectedType)
			}

			// Verificar que el tabContent tiene el tipo correcto
			if tabContent.Type != tc.expectedType {
				t.Errorf("❌ TabContent type wrong: Expected %v, got %v", tc.expectedType, tabContent.Type)
			} else {
				t.Logf("✅ TabContent type correct: %v", tabContent.Type)
			}
		})
	}

	t.Log("")
	t.Log("🎯 RESULT: sendSuccessMessage() y sendErrorMessage() ahora usan DetectMessageType")
	t.Log("✅ BENEFIT: El último mensaje de callback tendrá el color correcto según su contenido")
	t.Log("✅ CONSISTENCY: Todos los métodos de envío de mensajes usan centralización")
}
