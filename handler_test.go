package devtui

import (
	"strconv"
	"strings"
	"sync"
	"time"
)

// GetFirstTestTabIndex returns the index of the first test tab
// This centralizes the index calculation to avoid test failures when tabs are added/removed
// Currently, NewTUI always adds SHORTCUTS tab at index 0, so test tabs start at index 1
func GetFirstTestTabIndex() int {
	return 1 // SHORTCUTS tab is always at index 0, so first test tab is at index 1
}

// GetSecondTestTabIndex returns the index of the second test tab
func GetSecondTestTabIndex() int {
	return GetFirstTestTabIndex() + 1 // Second test tab follows first test tab
}

// TestEditableHandler - Handler para campos editables (input fields)
type TestEditableHandler struct {
	mu           sync.RWMutex
	label        string
	currentValue string
	lastOpID     string
	updateMode   bool // Para controlar si actualiza mensajes existentes
	log          func(message ...any)
}

func NewTestEditableHandler(label, value string) *TestEditableHandler {
	return &TestEditableHandler{
		label:        label,
		currentValue: value,
	}
}

func (h *TestEditableHandler) Label() string { return h.label }

func (h *TestEditableHandler) Value() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentValue
}

func (h *TestEditableHandler) editable() bool         { return true }
func (h *TestEditableHandler) Timeout() time.Duration { return 0 }

func (h *TestEditableHandler) Change(newValue string) {
	h.mu.Lock()
	h.currentValue = newValue
	h.mu.Unlock()
	if h.log != nil {
		h.log("Saved: " + newValue)
	}
}

func (h *TestEditableHandler) SetLog(f func(message ...any)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.log = f
}

// MessageTracker methods
func (h *TestEditableHandler) Name() string { return h.label + "Handler" }

func (h *TestEditableHandler) SetLastOperationID(lastOpID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastOpID = lastOpID
}

func (h *TestEditableHandler) GetLastOperationID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestEditableHandler) SetUpdateMode(update bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updateMode = update
}

// TestNonEditableHandler - Handler para botones de acción (action buttons)
type TestNonEditableHandler struct {
	mu         sync.RWMutex
	label      string
	actionText string
	lastOpID   string
	updateMode bool
	log        func(message ...any)
}

func NewTestNonEditableHandler(label, actionText string) *TestNonEditableHandler {
	return &TestNonEditableHandler{
		label:      label,
		actionText: actionText,
	}
}

func (h *TestNonEditableHandler) Label() string { return h.label }

func (h *TestNonEditableHandler) Value() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.actionText
}

func (h *TestNonEditableHandler) Timeout() time.Duration { return 0 }

func (h *TestNonEditableHandler) Change(newValue string) {
	if h.log != nil {
		h.log("Action executed: " + h.actionText)
	}
}

// HandlerExecution interface
func (h *TestNonEditableHandler) Execute() {
	if h.log != nil {
		h.log("Action executed: " + h.actionText)
	}
}

func (h *TestNonEditableHandler) SetLog(f func(message ...any)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.log = f
}

// Name returns the name of the handler.
func (h *TestNonEditableHandler) Name() string {
	return h.label
}

// MessageTracker methods
func (h *TestNonEditableHandler) SetLastOperationID(lastOpID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastOpID = lastOpID
}
func (h *TestNonEditableHandler) GetLastOperationID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestNonEditableHandler) SetUpdateMode(update bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updateMode = update
}

// TestWriterHandler - Handler solo para escribir (no es field, para componentes externos)
type TestWriterHandler struct {
	mu         sync.RWMutex
	name       string
	lastOpID   string
	updateMode bool
}

func NewTestWriterHandler(name string) *TestWriterHandler {
	return &TestWriterHandler{name: name}
}

// Solo implementa MessageTracker
func (h *TestWriterHandler) Name() string { return h.name }

func (h *TestWriterHandler) SetLastOperationID(lastOpID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastOpID = lastOpID
}

func (h *TestWriterHandler) GetLastOperationID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestWriterHandler) SetUpdateMode(update bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updateMode = update
}

// PortTestHandler - Handler específico para tests de puerto con validación
type PortTestHandler struct {
	mu          sync.RWMutex
	currentPort string
	lastOpID    string
	updateMode  bool
	log         func(message ...any)
}

func NewPortTestHandler(initialPort string) *PortTestHandler {
	return &PortTestHandler{currentPort: initialPort}
}

func (h *PortTestHandler) Label() string { return "Port" }

func (h *PortTestHandler) Value() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.currentPort
}

func (h *PortTestHandler) Editable() bool         { return true }
func (h *PortTestHandler) Timeout() time.Duration { return 3 * time.Second }

func (h *PortTestHandler) Change(newValue string) {
	portStr := strings.TrimSpace(newValue)
	if portStr == "" {
		if h.log != nil {
			h.log("port cannot be empty")
		}
		return
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		if h.log != nil {
			h.log("port must be a number")
		}
		return
	}
	if port < 1 || port > 65535 {
		if h.log != nil {
			h.log("port must be between 1 and 65535")
		}
		return
	}
	h.mu.Lock()
	h.currentPort = portStr
	h.mu.Unlock()
	if h.log != nil {
		h.log("Port configured: " + strconv.Itoa(port))
	}
}

func (h *PortTestHandler) SetLog(f func(message ...any)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.log = f
}

// MessageTracker methods
func (h *PortTestHandler) Name() string { return "PortHandler" }

func (h *PortTestHandler) SetLastOperationID(lastOpID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastOpID = lastOpID
}

func (h *PortTestHandler) GetLastOperationID() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *PortTestHandler) SetUpdateMode(update bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updateMode = update
}

// TestErrorHandler - Handler que siempre genera errores para testing
type TestErrorHandler struct {
	label      string
	value      string
	lastOpID   string
	updateMode bool
	log        func(message ...any)
}

func NewTestErrorHandler(label, value string) *TestErrorHandler {
	return &TestErrorHandler{
		label: label,
		value: value,
	}
}

func (h *TestErrorHandler) Label() string          { return h.label }
func (h *TestErrorHandler) Value() string          { return h.value }
func (h *TestErrorHandler) Editable() bool         { return true }
func (h *TestErrorHandler) Timeout() time.Duration { return 0 }

func (h *TestErrorHandler) Change(newValue string) {
	if h.log != nil {
		h.log("simulated error occurred")
	}
}

func (h *TestErrorHandler) SetLog(f func(message ...any)) {
	h.log = f
}

// MessageTracker methods
func (h *TestErrorHandler) Name() string                       { return h.label + "ErrorHandler" }
func (h *TestErrorHandler) SetLastOperationID(lastOpID string) { h.lastOpID = lastOpID }
func (h *TestErrorHandler) GetLastOperationID() string {
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestErrorHandler) SetUpdateMode(update bool) {
	h.updateMode = update
}

// TestRequiredFieldHandler - Handler que rechaza valores vacíos
type TestRequiredFieldHandler struct {
	label        string
	currentValue string
	lastOpID     string
	updateMode   bool
	log          func(message ...any)
}

func NewTestRequiredFieldHandler(label, initialValue string) *TestRequiredFieldHandler {
	return &TestRequiredFieldHandler{
		label:        label,
		currentValue: initialValue,
	}
}

func (h *TestRequiredFieldHandler) Label() string          { return h.label }
func (h *TestRequiredFieldHandler) Value() string          { return h.currentValue }
func (h *TestRequiredFieldHandler) Editable() bool         { return true }
func (h *TestRequiredFieldHandler) Timeout() time.Duration { return 0 }

func (h *TestRequiredFieldHandler) Change(newValue string) {
	if newValue == "" {
		if h.log != nil {
			h.log("Field cannot be empty")
		}
		return
	}
	h.currentValue = newValue
	if h.log != nil {
		h.log("Accepted: " + newValue)
	}
}

func (h *TestRequiredFieldHandler) SetLog(f func(message ...any)) {
	h.log = f
}

// MessageTracker methods
func (h *TestRequiredFieldHandler) Name() string                       { return h.label + "RequiredHandler" }
func (h *TestRequiredFieldHandler) SetLastOperationID(lastOpID string) { h.lastOpID = lastOpID }
func (h *TestRequiredFieldHandler) GetLastOperationID() string {
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestRequiredFieldHandler) SetUpdateMode(update bool) {
	h.updateMode = update
}

// TestOptionalFieldHandler - Handler que acepta valores vacíos
type TestOptionalFieldHandler struct {
	label        string
	currentValue string
	lastOpID     string
	updateMode   bool
	log          func(message ...any)
}

func NewTestOptionalFieldHandler(label, initialValue string) *TestOptionalFieldHandler {
	return &TestOptionalFieldHandler{
		label:        label,
		currentValue: initialValue,
	}
}

func (h *TestOptionalFieldHandler) Label() string          { return h.label }
func (h *TestOptionalFieldHandler) Value() string          { return h.currentValue }
func (h *TestOptionalFieldHandler) Editable() bool         { return true }
func (h *TestOptionalFieldHandler) Timeout() time.Duration { return 0 }

func (h *TestOptionalFieldHandler) Change(newValue string) {
	h.currentValue = newValue
	if newValue == "" {
		h.currentValue = "Default Value" // Para el test que espera esta transformación
		if h.log != nil {
			h.log("Default Value")
		}
	} else {
		if h.log != nil {
			h.log("Updated: " + newValue)
		}
	}
}

func (h *TestOptionalFieldHandler) SetLog(f func(message ...any)) {
	h.log = f
}

// MessageTracker methods
func (h *TestOptionalFieldHandler) Name() string                       { return h.label + "OptionalHandler" }
func (h *TestOptionalFieldHandler) SetLastOperationID(lastOpID string) { h.lastOpID = lastOpID }
func (h *TestOptionalFieldHandler) GetLastOperationID() string {
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestOptionalFieldHandler) SetUpdateMode(update bool) {
	h.updateMode = update
}

// TestClearableFieldHandler - Handler que preserva valores vacíos tal como son
type TestClearableFieldHandler struct {
	label        string
	currentValue string
	lastOpID     string
	updateMode   bool
	log          func(message ...any)
}

func NewTestClearableFieldHandler(label, initialValue string) *TestClearableFieldHandler {
	return &TestClearableFieldHandler{
		label:        label,
		currentValue: initialValue,
	}
}

func (h *TestClearableFieldHandler) Label() string          { return h.label }
func (h *TestClearableFieldHandler) Value() string          { return h.currentValue }
func (h *TestClearableFieldHandler) Editable() bool         { return true }
func (h *TestClearableFieldHandler) Timeout() time.Duration { return 0 }

func (h *TestClearableFieldHandler) Change(newValue string) {
	h.currentValue = newValue
	if h.log != nil {
		h.log(newValue)
	}
}

func (h *TestClearableFieldHandler) SetLog(f func(message ...any)) {
	h.log = f
}

// MessageTracker methods
func (h *TestClearableFieldHandler) Name() string                       { return h.label + "ClearableHandler" }
func (h *TestClearableFieldHandler) SetLastOperationID(lastOpID string) { h.lastOpID = lastOpID }
func (h *TestClearableFieldHandler) GetLastOperationID() string {
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestClearableFieldHandler) SetUpdateMode(update bool) {
	h.updateMode = update
}

// TestCapturingHandler - Handler que captura valores recibidos para testing
type TestCapturingHandler struct {
	label         string
	currentValue  string
	capturedValue *string // Puntero para capturar valores en tests
	lastOpID      string
	updateMode    bool
	log           func(message ...any)
}

func NewTestCapturingHandler(label, initialValue string, capturedValue *string) *TestCapturingHandler {
	return &TestCapturingHandler{
		label:         label,
		currentValue:  initialValue,
		capturedValue: capturedValue,
	}
}

func (h *TestCapturingHandler) Label() string          { return h.label }
func (h *TestCapturingHandler) Value() string          { return h.currentValue }
func (h *TestCapturingHandler) Editable() bool         { return true }
func (h *TestCapturingHandler) Timeout() time.Duration { return 0 }

func (h *TestCapturingHandler) Change(newValue string) {
	if h.capturedValue != nil {
		*h.capturedValue = newValue // Captura el valor para el test
	}
	if newValue == "" {
		h.currentValue = "Field was cleared" // Actualizar el valor interno también
		return
	}
	h.currentValue = newValue
}

func (h *TestCapturingHandler) SetLog(f func(message ...any)) {
	h.log = f
}

// MessageTracker methods
func (h *TestCapturingHandler) Name() string                       { return h.label + "CapturingHandler" }
func (h *TestCapturingHandler) SetLastOperationID(lastOpID string) { h.lastOpID = lastOpID }
func (h *TestCapturingHandler) GetLastOperationID() string {
	if h.updateMode {
		return h.lastOpID
	}
	return ""
}

// SetUpdateMode permite controlar si actualiza mensajes para tests
func (h *TestCapturingHandler) SetUpdateMode(update bool) {
	h.updateMode = update
}

// DefaultTUIForTest creates a DevTUI instance with configurable handlers
// Usage examples:
//   - DefaultTUIForTest() // Empty TUI, no handlers
//   - DefaultTUIForTest(handler1, handler2) // TUI with specified handlers
//   - DefaultTUIForTest(handler1, func(messages...any){}) // TUI with handlers + logger
func DefaultTUIForTest(handlersAndLogger ...any) *DevTUI {
	var logFunc func(messages ...any)
	// UPDATED: Removed FieldHandler support - use specific handler types

	// Parse variadic arguments: handlers and optional logger (func)
	for _, arg := range handlersAndLogger {
		switch v := arg.(type) {
		case func(messages ...any):
			logFunc = v
			// NOTE: Specific handler types should be added via tab.AddEditHandler, etc.
		}
	}

	// Default no-op logger if none provided
	if logFunc == nil {
		logFunc = func(messages ...any) {
			// No-op logger for tests
		}
	}

	// Initialize the UI with TestMode enabled for synchronous execution
	h := NewTUI(&TuiConfig{
		ExitChan: make(chan bool), // Channel to signal exit
		Color:    nil,             // Use default colors
		Logger:   logFunc,
	})

	// Enable test mode for synchronous execution
	h.SetTestMode(true)

	// NOTE: For test tabs with handlers, use:
	// tab := h.NewTabSection("Test Tab", "Tab description")
	// tab.AddEditHandler(yourHandler).WithTimeout(timeout)
	// tab.NewDisplayHandler(yourDisplayHandler)
	// etc.

	return h
}
