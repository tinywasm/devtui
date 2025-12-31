package devtui

import (
	"time"
)

// ============================================================================
// PRIVATE IMPLEMENTATION - anyHandler Structure
// ============================================================================

type handlerType int

const (
	handlerTypeDisplay handlerType = iota
	handlerTypeEdit
	handlerTypeExecution
	handlerTypeInteractive // NEW: Interactive content handler
	handlerTypeLoggable    // NEW: For Loggable-only handlers
)

// anyHandler - Estructura privada que unifica todos los handlers
type anyHandler struct {
	handlerType handlerType
	timeout     time.Duration // Solo edit/execution

	origHandler any // Store original handler for type assertions

	handlerColor string // NEW: Handler-specific color for message formatting

	// Function pointers - solo los necesarios poblados
	nameFunc     func() string        // Todos
	labelFunc    func() string        // Display/Edit/Execution
	valueFunc    func() string        // Edit/Display
	contentFunc  func() string        // Display únicamente
	editableFunc func() bool          // Por tipo
	editModeFunc func() bool          // NEW: Auto edit mode activation
	changeFunc   func(string)         // Edit/Execution (nueva firma)
	executeFunc  func()               // Execution únicamente (nueva firma)
	timeoutFunc  func() time.Duration // Edit/Execution
}

// ============================================================================
// anyHandler Methods - Replaces fieldHandler interface
// ============================================================================

func (a *anyHandler) Name() string {
	if a.nameFunc != nil {
		return a.nameFunc()
	}
	return ""
}

func (a *anyHandler) Label() string {
	if a.labelFunc != nil {
		return a.labelFunc()
	}
	return ""
}

func (a *anyHandler) Value() string {
	if a.valueFunc != nil {
		return a.valueFunc()
	}
	return ""
}

func (a *anyHandler) editable() bool {
	if a.editableFunc != nil {
		return a.editableFunc()
	}
	return false
}

func (a *anyHandler) Change(newValue string) {
	if a.changeFunc != nil {
		a.changeFunc(newValue)
	}
}

func (a *anyHandler) Execute() {
	if a.executeFunc != nil {
		a.executeFunc()
	}
}

func (a *anyHandler) Timeout() time.Duration {
	if a.timeoutFunc != nil {
		return a.timeoutFunc()
	}
	return a.timeout
}

// GetTrackingKey returns the handler name to be used for message tracking
func (a *anyHandler) GetTrackingKey() string {
	return a.Name()
}

func (a *anyHandler) WaitingForUser() bool {
	if a.editModeFunc != nil {
		return a.editModeFunc()
	}
	return false
}

// ============================================================================
// Factory Methods
// ============================================================================

func NewEditHandler(h HandlerEdit, timeout time.Duration, color string) *anyHandler {
	anyH := &anyHandler{
		handlerType:  handlerTypeEdit,
		timeout:      timeout,
		nameFunc:     h.Name,
		labelFunc:    h.Label,
		valueFunc:    h.Value,
		editableFunc: func() bool { return true },
		changeFunc:   h.Change,
		timeoutFunc:  func() time.Duration { return timeout },
		origHandler:  h,
		handlerColor: color, // NEW: Store handler color
	}

	// NEW: Check if handler also implements Value() method (like TestNonEditableHandler)
	if valuer, ok := h.(interface{ Value() string }); ok {
		anyH.valueFunc = valuer.Value
	} else {
		anyH.valueFunc = h.Label // Fallback to Label
	}

	return anyH
}

func NewDisplayHandler(h HandlerDisplay, color string) *anyHandler {
	return &anyHandler{
		handlerType:  handlerTypeDisplay,
		timeout:      0,         // Display no requiere timeout
		nameFunc:     h.Name,    // Solo Name()
		valueFunc:    h.Content, // Content como Value para compatibilidad interna
		contentFunc:  h.Content, // Solo Content()
		editableFunc: func() bool { return false },
		handlerColor: color, // NEW: Store handler color
	}
}

func NewExecutionHandler(h HandlerExecution, timeout time.Duration, color string) *anyHandler {
	anyH := &anyHandler{
		handlerType:  handlerTypeExecution,
		timeout:      timeout,
		nameFunc:     h.Name,
		labelFunc:    h.Label,
		editableFunc: func() bool { return false },
		executeFunc:  h.Execute,
		changeFunc: func(_ string) {
			h.Execute()
		},
		timeoutFunc:  func() time.Duration { return timeout },
		origHandler:  h,
		handlerColor: color, // NEW: Store handler color
	}

	// Check if handler also implements Value() method (like TestNonEditableHandler)
	if valuer, ok := h.(interface{ Value() string }); ok {
		anyH.valueFunc = valuer.Value
	} else {
		anyH.valueFunc = h.Label // Fallback to Label
	}

	return anyH
}

func NewInteractiveHandler(h HandlerInteractive, timeout time.Duration, color string) *anyHandler {
	anyH := &anyHandler{
		handlerType: handlerTypeInteractive,
		timeout:     timeout,
		nameFunc:    h.Name,
		labelFunc:   h.Label,
		valueFunc:   h.Value,
		// NO contentFunc - interactive handlers use progress() only
		editableFunc: func() bool { return true },
		changeFunc:   h.Change,
		timeoutFunc:  func() time.Duration { return timeout },
		editModeFunc: h.WaitingForUser, // NEW: Auto edit mode detection
		origHandler:  h,
		handlerColor: color, // NEW: Store handler color
	}

	return anyH
}
