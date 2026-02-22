package devtui

import (
	"testing"

	. "github.com/tinywasm/fmt"
)

// TestCapitalizeFormatPreservation tests specific formatting issues
func TestCapitalizeFormatPreservation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Preserve newlines and indentation",
			input:    "hello world\n  • item one\n  • item two",
			expected: "Hello World\n  • Item One\n  • Item Two",
		},
		{
			name:     "Preserve bullet points formatting",
			input:    "section:\n  • first item\n  • second item",
			expected: "Section:\n  • First Item\n  • Second Item",
		},
		{
			name:     "Complex multiline with various symbols",
			input:    "tabs:\n  • tab/shift+tab  - switch tabs\n\nfields:\n  • left/right     - navigate",
			expected: "Tabs:\n  • Tab/Shift+Tab  - Switch Tabs\n\nFields:\n  • Left/Right     - Navigate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input).Capitalize().String()
			if result != tt.expected {
				t.Errorf("Capitalize() formatting test failed.\nInput:    %q\nExpected: %q\nGot:      %q", tt.input, tt.expected, result)
			}
		})
	}
}

// TestTranslationWithCapitalize tests the actual problem scenario
func TestTranslationWithCapitalize(t *testing.T) {
	// Test the exact pattern used in generateHelpContent
	tests := []struct {
		name     string
		lang     string
		expected string
	}{
		{
			name:     "English with proper formatting",
			lang:     "EN",
			expected: "Test Shortcuts Keyboard (\"En\"):\n\nTabs:\n  • Tab/Shift+Tab  - Switch Tabs",
		},
		{
			name:     "Spanish with proper formatting",
			lang:     "ES",
			expected: "Test Atajos Teclado (\"Es\"):\n\nPestañas:\n  • Tab/Shift+Tab  - Cambiar Pestañas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This mimics the exact problem pattern from generateHelpContent
			result := Translate("Test", "shortcuts", "keyboard", `("`+tt.lang+`"):

Tabs:
  • Tab/Shift+Tab  -`, "switch", ` tabs`).Capitalize().String()

			// Check if the basic structure is preserved
			if !Contains(result, "\n") {
				t.Errorf("Newlines should be preserved in result: %q", result)
			}

			if !Contains(result, "•") {
				t.Errorf("Bullet points should be preserved in result: %q", result)
			}

			// Log the actual result for debugging
			t.Logf("Language: %s\nResult: %q", tt.lang, result)
		})
	}
}
