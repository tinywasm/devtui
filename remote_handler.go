//go:build !wasm

package devtui

import (
	"context"

	"github.com/tinywasm/mcp"
)

// newRemoteField constructs a *field populated from a StateEntry.
// Uses anyHandler closures directly — no intermediate interface types needed.
// The entry pointer is captured so optimistic value updates stay in sync.
func newRemoteField(entry StateEntry, client *mcp.Client, ts *tabSection) *field {
	e := entry // local copy captured by closures
	var anyH *anyHandler

	switch handlerType(e.HandlerType) {
	case handlerTypeDisplay:
		anyH = &anyHandler{
			handlerType:  handlerTypeDisplay,
			handlerColor: e.HandlerColor,
			nameFunc:     func() string { return e.HandlerName },
			valueFunc:    func() string { return e.Value },
			contentFunc:  func() string { return e.Value },
			editableFunc: func() bool { return false },
		}
	case handlerTypeEdit:
		anyH = &anyHandler{
			handlerType:  handlerTypeEdit,
			handlerColor: e.HandlerColor,
			nameFunc:     func() string { return e.HandlerName },
			labelFunc:    func() string { return e.Label },
			valueFunc:    func() string { return e.Value },
			editableFunc: func() bool { return true },
			changeFunc: func(v string) {
				e.Value = v // optimistic update
				postAction(client, e.Shortcut, v)
			},
		}
	case handlerTypeExecution:
		anyH = &anyHandler{
			handlerType:  handlerTypeExecution,
			handlerColor: e.HandlerColor,
			nameFunc:     func() string { return e.HandlerName },
			labelFunc:    func() string { return e.Label },
			valueFunc:    func() string { return e.Label },
			editableFunc: func() bool { return false },
			executeFunc:  func() { postAction(client, e.Shortcut, "") },
			changeFunc:   func(_ string) { postAction(client, e.Shortcut, "") },
		}
	case handlerTypeInteractive:
		anyH = &anyHandler{
			handlerType:  handlerTypeInteractive,
			handlerColor: e.HandlerColor,
			nameFunc:     func() string { return e.HandlerName },
			labelFunc:    func() string { return e.Label },
			valueFunc:    func() string { return e.Value },
			editableFunc: func() bool { return true },
			editModeFunc: func() bool { return false },
			changeFunc: func(v string) {
				postAction(client, e.Shortcut, v)
			},
		}
	default:
		return nil // HandlerTypeLoggable — no field, logs arrive via SSE
	}

	return &field{handler: anyH, parentTab: ts, isRemote: true}
}

// postAction sends a tinywasm/action JSON-RPC call to the daemon (fire-and-forget).
func postAction(client *mcp.Client, shortcut, value string) {
	if shortcut == "" || client == nil {
		return
	}
	client.Dispatch(context.Background(), "tinywasm/action", map[string]string{
		"key":   shortcut,
		"value": value,
	})
}
