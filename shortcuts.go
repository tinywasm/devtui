package devtui

// createShortcutsTab creates and registers the shortcuts tab with its handler
import (
	. "github.com/tinywasm/fmt"
)

func createShortcutsTab(tui *DevTUI) {
	shortcutsTab := tui.NewTabSection("SHORTCUTS", "Keyboard navigation instructions")

	handler := &shortcutsInteractiveHandler{
		appName:            tui.AppName,
		lang:               OutLang(), // Get current language automatically
		needsLanguageInput: false,     // Initially show help content
		tui:                tui,       // NEW: Reference to TUI for shortcut registry access
	}
	// Use AddHandler for all handler types
	tui.AddHandler(handler, "", shortcutsTab)
}

// shortcutsInteractiveHandler - Interactive handler for language selection and help display
type shortcutsInteractiveHandler struct {
	appName            string
	lang               string  // e.g. "EN", "ES", etc.
	needsLanguageInput bool    // Controls when to activate edit mode
	lastOpID           string  // Operation ID for tracking
	tui                *DevTUI // NEW: Reference to TUI for shortcut registry access
	log                func(message ...any)
}

func (h *shortcutsInteractiveHandler) Name() string {
	return "shortcutsGuide"
}

func (h *shortcutsInteractiveHandler) Label() string {
	return Translate("language", ":").String()
}

// MessageTracker implementation for operation tracking
func (h *shortcutsInteractiveHandler) GetLastOperationID() string   { return h.lastOpID }
func (h *shortcutsInteractiveHandler) SetLastOperationID(id string) { h.lastOpID = id }

func (h *shortcutsInteractiveHandler) Value() string { return Convert(h.lang).ToLower().String() }

func (h *shortcutsInteractiveHandler) SetLog(f func(message ...any)) {
	h.log = f
}

// Change handles both content display and user input via log()
func (h *shortcutsInteractiveHandler) Change(newValue string) {
	if newValue == "" && !h.needsLanguageInput {
		// Display help content when field is selected (not in edit mode)
		if h.log != nil {
			h.log(h.generateHelpContent())
		}
		return
	}

	// Handle language change
	lang := OutLang(newValue)
	h.lang = lang
	h.needsLanguageInput = false

	// Show updated help content
	if h.log != nil {
		h.log(h.generateHelpContent())
	}
}

func (h *shortcutsInteractiveHandler) WaitingForUser() bool {
	return h.needsLanguageInput
}

// generateHelpContent creates the help content string
func (h *shortcutsInteractiveHandler) generateHelpContent() string {
	content := Translate(h.appName, "shortcuts", "keyboard", `:`+"\n\n",
		"content", "tab", `:
  • Tab/Shift+Tab  -`, "switch", "content", "\n\n",
		"fields", `:
  • `, "arrow", "left", `/`, "right", `     -`, "switch", "field", `
  • Enter          				-`, "edit", `/`, "execute", `
  • Esc            				-`, "cancel", "\n\n",
		"edit", "text", `:
  • `, "arrow", "left", `/`, "right", `   -`, "move", `cursor
  • Backspace      			-`, "create", "space", `

Viewport:
  • `, "arrow", "up", "/", "down", `    - Scroll`, "line", "text", `
  • PgUp/PgDown    		- Scroll`, "page", `
  • Mouse Wheel    		- Scroll`, "page", "\n\n",
		`Scroll `, "status", "icons", `:
  •  ■  - `, "all", "content", "visible", `
  •  ▼  - `, "can", `scroll`, "down", `
  •  ▲  - `, "can", `scroll`, "up", `
  • ▼ ▲ - `, "can", `scroll`, "down", `/`, "up", "\n\n",
		"quit", `:
  • Ctrl+C         - `, "quit", `
`).String()

	// Add registered shortcuts section
	if h.tui != nil && h.tui.shortcutRegistry != nil {
		shortcuts := h.getRegisteredShortcuts()
		if len(shortcuts) > 0 {
			content += "\n\nRegistered Shortcuts:\n"
			for key, description := range shortcuts {
				content += Sprintf("  • %s - %s\n", key, description)
			}
		}
	}

	content += "\n" + Translate("language", "supported", `: en, es, zh, hi, ar, pt, fr, de, ru`).String()
	return content
}

// getRegisteredShortcuts returns all registered shortcuts with descriptions
func (h *shortcutsInteractiveHandler) getRegisteredShortcuts() map[string]string {
	shortcuts := make(map[string]string)
	if h.tui != nil && h.tui.shortcutRegistry != nil {
		allEntries := h.tui.shortcutRegistry.GetAll()
		for key, entry := range allEntries {
			shortcuts[key] = entry.Description
		}
	}
	return shortcuts
}
