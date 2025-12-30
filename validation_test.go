package devtui

import (
	"testing"
	"time"
)

func TestValidateTabSection_Nil(t *testing.T) {
	tui := NewTUI(&TuiConfig{})

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for nil tabSection")
		}
	}()

	tui.AddHandler(&validationTestHandler{}, time.Second, "", nil)
}

func TestValidateTabSection_WrongType(t *testing.T) {
	tui := NewTUI(&TuiConfig{})

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for wrong type")
		}
	}()

	tui.AddHandler(&validationTestHandler{}, time.Second, "", "not a tabSection")
}

func TestValidateTabSection_WrongDevTUI(t *testing.T) {
	tui1 := NewTUI(&TuiConfig{})
	tui2 := NewTUI(&TuiConfig{})

	tab := tui1.NewTabSection("TEST", "test")

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for tabSection from different DevTUI")
		}
	}()

	tui2.AddHandler(&validationTestHandler{}, time.Second, "", tab)
}

func (h *validationTestDisplayHandler) Content() string {
	return "display value"
}

func TestValidateTabSection_Success(t *testing.T) {
	tui := NewTUI(&TuiConfig{})
	tab := tui.NewTabSection("TEST", "test")

	// Should not panic
	tui.AddHandler(&validationTestDisplayHandler{name: "test"}, 0, "", tab)

	// Loggable handlers are registered via AddHandler
	h := &validationTestLoggableHandler{name: "logger"}
	tui.AddHandler(h, 0, "", tab)

	if h.logFunc == nil {
		t.Error("Expected logger function to be injected, got nil")
	}
}

// validationTestLoggableHandler is a minimal loggable handler for testing purposes
type validationTestLoggableHandler struct {
	name    string
	logFunc func(message ...any)
}

func (h *validationTestLoggableHandler) Name() string {
	return h.name
}

func (h *validationTestLoggableHandler) SetLog(f func(message ...any)) {
	h.logFunc = f
}

// validationTestHandler is a minimal handler for testing purposes
type validationTestHandler struct{}

func (h *validationTestHandler) Name() string {
	return "test"
}

func (h *validationTestHandler) Value() string {
	return "test value"
}

// validationTestDisplayHandler is a minimal display handler for testing purposes
type validationTestDisplayHandler struct {
	name string
}

func (h *validationTestDisplayHandler) Name() string {
	return h.name
}

func (h *validationTestDisplayHandler) Value() string {
	return "display value"
}
