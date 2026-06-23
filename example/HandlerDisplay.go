package example

import (
	"github.com/tinywasm/fmt/lang"
)

type StatusHandler struct{}

func (h *StatusHandler) Name() string { return lang.Translate("information", "status", "system").String() }
func (h *StatusHandler) Content() string {
	return "Status: Running\nPID: 12345\nUptime: 2h 30m\nMemory: 45MB\nCPU: 12%"
}
