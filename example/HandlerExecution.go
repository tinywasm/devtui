package example

import (
	"fmt"
	"time"
)

type BackupHandler struct {
	lastOpID string
	log      func(message ...any)
}

func (h *BackupHandler) Name() string  { return "SystemBackup" }
func (h *BackupHandler) Label() string { return "With Tracking" }

// SetLog receives the logger from DevTUI
func (h *BackupHandler) SetLog(logger func(message ...any)) {
	h.log = logger
}

func (h *BackupHandler) Execute() {
	if h.log == nil {
		h.log = func(message ...any) { fmt.Println(message...) }
	}

	h.log("Preparing backup... " + h.lastOpID)
	time.Sleep(500 * time.Millisecond)
	h.log("BackingUp database... " + h.lastOpID)
	time.Sleep(500 * time.Millisecond)
	h.log("BackingUp Files " + h.lastOpID)
	time.Sleep(500 * time.Millisecond)
	h.log("Backup End OK " + h.lastOpID)
}

func (h *BackupHandler) GetCompactID() string   { return h.lastOpID }
func (h *BackupHandler) SetCompactID(id string) { h.lastOpID = id }
