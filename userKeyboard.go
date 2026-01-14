package devtui

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyboard processes keyboard input and updates the model state
// returns whether the update function should continue processing or return early
func (h *DevTUI) handleKeyboard(msg tea.KeyMsg) (bool, tea.Cmd) {
	if h.editModeActivated { // EDITING CONFIG IN SECTION
		return h.handleEditingConfigKeyboard(msg)
	} else {
		return h.handleNormalModeKeyboard(msg)
	}
}

// handleEditingConfigKeyboard handles keyboard input while in config editing mode
func (h *DevTUI) handleEditingConfigKeyboard(msg tea.KeyMsg) (bool, tea.Cmd) {
	currentTab := h.TabSections[h.activeTab]
	fieldHandlers := currentTab.fieldHandlers
	currentField := fieldHandlers[currentTab.indexActiveEditField]

	if currentField.editable() { // Si el campo es editable, permitir la edición
		// Calcular el ancho máximo disponible para el texto
		// Esto sigue la misma lógica que en footerInput.go
		_, availableTextWidth := h.calculateInputWidths(currentField.handler.Label())

		switch msg.Type {
		case tea.KeyEnter: // Guardar cambios o ejecutar acción
			// For interactive handlers, ALWAYS call Change() - user is confirming the value
			// For edit handlers, only call Change() if value changed
			shouldExecute := currentField.isInteractiveHandler() || currentField.tempEditValue != currentField.Value()

			if shouldExecute {
				if currentField.handler != nil {
					currentField.handleEnter()
					h.editingConfigOpen(false, currentField, "", false) // false = check WaitingForUser
				}
			} else {
				// No changes and not interactive, just exit edit mode
				h.editingConfigOpen(false, currentField, "", false)
			}

			// Only clear tempEditValue if edit mode actually closed
			if !h.editModeActivated {
				currentField.tempEditValue = ""
			}
			h.updateViewport() // Asegurar que se actualice la vista para mostrar el mensaje
			return false, nil

		case tea.KeyEsc: // Al presionar ESC, descartamos los cambios y salimos del modo edición
			currentField.tempEditValue = "" // Limpiar el valor temporal

			// Notify handler if it implements Cancelable
			if currentField.handler != nil && currentField.handler.origHandler != nil {
				if cancelable, ok := currentField.handler.origHandler.(Cancelable); ok {
					cancelable.Cancel()
				}
			}

			h.editingConfigOpen(false, currentField, "", true) // true = force close, bypass WaitingForUser
			h.updateViewport()                                 // Asegurar que se actualice la vista para mostrar el mensaje
			return false, nil

		case tea.KeyLeft: // Mover el cursor a la izquierda dentro del texto
			if currentField.cursor > 0 {
				currentField.cursor--
				currentField.viewport.AdjustViewForCursor(len([]rune(currentField.Value())), currentField.cursor, availableTextWidth-1)
			}

		case tea.KeyRight: // Mover el cursor a la derecha dentro del texto
			value := currentField.Value()
			if currentField.tempEditValue != "" {
				value = currentField.tempEditValue
			}
			if currentField.cursor < len([]rune(value)) {
				currentField.cursor++
				currentField.viewport.AdjustViewForCursor(len([]rune(value)), currentField.cursor, availableTextWidth-1)
			}

		case tea.KeyBackspace: // Borrar carácter a la izquierda
			if currentField.cursor > 0 {
				// Si aún no hay valor temporal, copiar el valor original solo la primera vez
				if currentField.tempEditValue == "" {
					currentField.tempEditValue = currentField.Value()
				}

				// Convert to runes to handle multi-byte characters correctly
				runes := []rune(currentField.tempEditValue)
				if currentField.cursor <= len(runes) {
					newRunes := slices.Delete(runes, currentField.cursor-1, currentField.cursor)
					currentField.tempEditValue = string(newRunes)
					currentField.cursor--
					currentField.viewport.AdjustViewForCursor(len(newRunes), currentField.cursor, availableTextWidth-1)
				}
			}

		case tea.KeySpace: // Manejar la tecla espacio como un carácter especial
			// Si aún no hay valor temporal, NO copiar el valor original automáticamente
			if currentField.tempEditValue == "" {
				currentField.tempEditValue = ""
			}

			runes := []rune(currentField.tempEditValue)
			if currentField.cursor > len(runes) {
				currentField.cursor = len(runes)
			}

			// Insert the space at cursor position
			newRunes := make([]rune, 0, len(runes)+1)
			newRunes = append(newRunes, runes[:currentField.cursor]...)
			newRunes = append(newRunes, ' ') // Agregar el espacio
			newRunes = append(newRunes, runes[currentField.cursor:]...)
			currentField.tempEditValue = string(newRunes)
			currentField.cursor++

			currentField.viewport.AdjustViewForCursor(len(newRunes), currentField.cursor, availableTextWidth-1)

		case tea.KeyRunes:
			// Handle normal character input - convert everything to runes for proper handling
			if len(msg.Runes) > 0 {
				// NOTA: No inicializar tempEditValue aquí si está vacío
				// Si está vacío, significa que el usuario limpió el campo intencionalmente
				runes := []rune(currentField.tempEditValue)
				if currentField.cursor > len(runes) {
					currentField.cursor = len(runes)
				}

				// Insert the new runes at cursor position
				newRunes := make([]rune, 0, len(runes)+len(msg.Runes))
				newRunes = append(newRunes, runes[:currentField.cursor]...)
				newRunes = append(newRunes, msg.Runes...)
				newRunes = append(newRunes, runes[currentField.cursor:]...)
				currentField.tempEditValue = string(newRunes)
				currentField.cursor += len(msg.Runes)

				currentField.viewport.AdjustViewForCursor(len(newRunes), currentField.cursor, availableTextWidth-1)
			}
		}
	} else { // Si el campo no es editable, solo ejecutar la acción
		switch msg.Type {
		case tea.KeyEnter:
			// content eg: "DevBrowser Opened"
			if currentField.handler != nil {
				// Trigger async operation for non-editable fields (action buttons)
				currentField.handleEnter()
			}
			h.editModeActivated = false
			h.updateViewport() // Asegurar que se actualice la vista para mostrar el mensaje
			return false, nil

		case tea.KeyEsc: // Permitir también salir con ESC para campos no editables
			h.editingConfigOpen(false, currentField, "", true) // true = force close
			h.updateViewport()                                 // Asegurar que se actualice la vista para mostrar el mensaje
			return false, nil
		}
	}

	return true, nil
}

// handleNormalModeKeyboard handles keyboard input in normal mode (not editing config)
func (h *DevTUI) handleNormalModeKeyboard(msg tea.KeyMsg) (bool, tea.Cmd) {
	currentTab := h.TabSections[h.activeTab]
	fieldHandlers := currentTab.fieldHandlers
	totalFields := len(fieldHandlers)

	switch msg.Type {
	case tea.KeyUp, tea.KeyDown:
		// Las teclas arriba y abajo controlan el scroll línea por línea del viewport
		// No modifican el campo activo, solo el scroll del contenido
		// No hacemos nada aquí para permitir que el manejo del viewport siga su curso normal

	case tea.KeyPgUp: // Page Up - scroll página completa hacia arriba
		h.viewport.PageUp()
		return false, nil

	case tea.KeyPgDown: // Page Down - scroll página completa hacia abajo
		h.viewport.PageDown()
		return false, nil

	case tea.KeyLeft: // Navegar al campo anterior (ciclo continuo)
		if totalFields > 0 {
			currentTab.indexActiveEditField = (currentTab.indexActiveEditField - 1 + totalFields) % totalFields
			h.updateViewport()
			h.checkAndTriggerInteractiveContent() // NEW: Auto-trigger content for interactive handlers
			return false, nil                     // Detener procesamiento adicional
		}

	case tea.KeyRight: // Navegar al campo siguiente (ciclo continuo)
		if totalFields > 0 {
			currentTab.indexActiveEditField = (currentTab.indexActiveEditField + 1) % totalFields
			h.updateViewport()
			h.checkAndTriggerInteractiveContent() // NEW: Auto-trigger content for interactive handlers
			return false, nil                     // Detener procesamiento adicional
		}

	case tea.KeyTab: // cambiar tabSection
		h.activeTab = (h.activeTab + 1) % len(h.TabSections)
		h.updateViewport()
		h.checkAndTriggerInteractiveContent() // NEW: Auto-trigger content for interactive handlers

	case tea.KeyShiftTab: // cambiar tabSection
		h.activeTab = (h.activeTab - 1 + len(h.TabSections)) % len(h.TabSections)
		h.updateViewport()
		h.checkAndTriggerInteractiveContent() // NEW: Auto-trigger content for interactive handlers

	case tea.KeyEnter: //Enter para entrar en modo edición, ejecuta la acción directamente si el campo no es editable
		if totalFields > 0 {
			fieldHandlers := currentTab.fieldHandlers
			field := fieldHandlers[currentTab.indexActiveEditField]
			if !field.editable() {
				// Trigger async operation for non-editable fields
				if field.handler != nil {
					field.handleEnter()
				}
			} else {
				// Para campos editables, activar modo de edición explícitamente
				field.tempEditValue = field.Value()
				field.setCursorAtEnd() // Always start cursor at end
				h.editModeActivated = true
				h.editingConfigOpen(true, field, "", false)
			}
			h.updateViewport()
		}

	case tea.KeyRunes: // NEW: Handle single character shortcuts
		if len(msg.Runes) == 1 {
			key := string(msg.Runes[0])
			if entry, exists := h.shortcutRegistry.Get(key); exists {
				return h.executeShortcut(entry)
			}
		}

	case tea.KeyCtrlC:
		close(h.ExitChan) // Cerrar el canal para señalizar a todas las goroutines
		// Usar tea.Sequence para asegurar que ExitAltScreen se ejecute antes de Quit
		return false, tea.Sequence(tea.ExitAltScreen, tea.Quit)
	}

	return true, nil
}

// checkAndTriggerInteractiveContent checks if the active field is interactive and triggers content display automatically
func (h *DevTUI) checkAndTriggerInteractiveContent() {
	if h.activeTab >= len(h.TabSections) {
		return
	}

	activeTab := h.TabSections[h.activeTab]
	fieldHandlers := activeTab.fieldHandlers

	if len(fieldHandlers) == 0 || activeTab.indexActiveEditField >= len(fieldHandlers) {
		return
	}

	activeField := fieldHandlers[activeTab.indexActiveEditField]
	if activeField != nil && activeField.isInteractiveHandler() {
		// Auto-activate edit mode if handler requested it
		if !h.editModeActivated && activeField.shouldAutoActivateEditMode() {
			h.editingConfigOpen(true, activeField, activeField.handler.Value(), false)
			return
		}

		// Otherwise just trigger content display (default behavior)
		if !h.editModeActivated {
			activeField.handler.Change("")
		}
	}
}

// executeShortcut executes a registered shortcut action
func (h *DevTUI) executeShortcut(entry *ShortcutEntry) (bool, tea.Cmd) {
	// Validate indexes are still valid
	if entry.TabIndex >= len(h.TabSections) {
		if h.Logger != nil {
			h.Logger("Shortcut error: invalid tab index", entry.TabIndex)
		}
		return false, nil // Stop processing for invalid shortcuts
	}

	targetTab := h.TabSections[entry.TabIndex]
	fieldHandlers := targetTab.fieldHandlers
	if entry.FieldIndex >= len(fieldHandlers) {
		if h.Logger != nil {
			h.Logger("Shortcut error: invalid field index", entry.FieldIndex)
		}
		return false, nil // Stop processing for invalid shortcuts
	}

	targetField := fieldHandlers[entry.FieldIndex]

	// Navigate to target tab if not already there
	if h.activeTab != entry.TabIndex {
		h.activeTab = entry.TabIndex
	}

	// Set active field
	targetTab.indexActiveEditField = entry.FieldIndex

	// Execute the Change method with shortcut value
	if targetField.handler != nil {
		// Use Change() without channel - messages flow through h.log()
		// Execute synchronously to ensure deterministic behavior for shortcuts
		targetField.handler.Change(entry.Value)
	}

	// Update viewport to show changes
	h.updateViewport()

	return false, nil // Stop further processing
}
