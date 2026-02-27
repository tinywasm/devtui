package devtui

// StateEntry is the JSON wire format for a single handler registered in the daemon TUI.
// Produced by app.HeadlessTUI, consumed by DevTUI client mode.
// JSON tags are the published contract — any producer must match them exactly.
type StateEntry struct {
	TabTitle     string `json:"tab_title"`
	HandlerName  string `json:"handler_name"`
	HandlerColor string `json:"handler_color"`
	HandlerType  int    `json:"handler_type"` // HandlerType* constant below
	Label        string `json:"label"`
	Value        string `json:"value"`
	Shortcut     string `json:"shortcut"` // keyboard key that controls this handler
}

// HandlerType constants — mirror the private handlerType iota in anyHandler.go.
// These values are part of the published wire protocol. Do not reorder.
const (
	HandlerTypeDisplay     = 0
	HandlerTypeEdit        = 1
	HandlerTypeExecution   = 2
	HandlerTypeInteractive = 3
	HandlerTypeLoggable    = 4
)
