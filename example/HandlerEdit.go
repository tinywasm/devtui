package example

import (
	"fmt"
	"time"
)

type DatabaseHandler struct {
	ConnectionString string
	LastAction       string
	log              func(message ...any)
}

func (h *DatabaseHandler) Name() string  { return "DatabaseConfig" }
func (h *DatabaseHandler) Label() string { return "Database Connection" }
func (h *DatabaseHandler) Value() string { return h.ConnectionString }

// SetLog receives the logger from DevTUI
func (h *DatabaseHandler) SetLog(logger func(message ...any)) {
	h.log = logger
}

func (h *DatabaseHandler) Change(newValue string) {
	// Ensure log is not nil (safe guard)
	if h.log == nil {
		h.log = func(message ...any) { fmt.Println(message...) }
	}

	switch newValue {
	case "t":
		h.LastAction = "test"
		h.log("Testing database connection...")
		time.Sleep(500 * time.Millisecond)
		h.log("Connection test completed successfully")

	case "b":
		h.LastAction = "backup"
		h.log("Starting database backup...")
		time.Sleep(1000 * time.Millisecond)
		h.log("Database backup completed successfully")

	default:
		// Regular connection string update
		h.log("Validating Connection " + newValue)
		time.Sleep(500 * time.Millisecond)
		h.log("Testing database connectivity... " + newValue)
		time.Sleep(500 * time.Millisecond)
		h.log("Connection Database configured successfully " + newValue)
		h.ConnectionString = newValue
	}
}

// NEW: Add shortcut support
func (h *DatabaseHandler) Shortcuts() []map[string]string {
	return []map[string]string{
		{"t": "test connection"},
		{"b": "backup database"},
	}
}
