//go:build !wasm

package devtui

import (
	"github.com/tinywasm/context"
	"github.com/tinywasm/mcp"
)

// newRemoteField constructs a *field populated from a StateEntry.
// Uses anyHandler closures directly — no intermediate interface types needed.
// The entry pointer is captured so optimistic value updates stay in sync.
// The tui reference is used to register shortcuts in the ShortcutRegistry.
func newRemoteField(entry StateEntry, client *mcp.Client, ts *tabSection, tui *DevTUI) *field {
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

	f := &field{handler: anyH, parentTab: ts, isRemote: true}

	// Register shortcuts from StateEntry.Shortcuts in the TUI's registry
	if tui != nil && tui.shortcutRegistry != nil && len(e.Shortcuts) > 0 {
		fieldIndex := len(ts.FieldHandlers)
		tabIndex := ts.Index
		for _, m := range e.Shortcuts {
			for key := range m {
				entry := &ShortcutEntry{
					Key:         key,
					Description: key, // Use key as description for remote shortcuts
					TabIndex:    tabIndex,
					FieldIndex:  fieldIndex,
					HandlerName: e.HandlerName,
					Value:       key,
				}
				tui.shortcutRegistry.Register(key, entry)
			}
		}
	}

	return f
}

// postAction sends a tinywasm/action JSON-RPC call to the daemon (fire-and-forget).
func postAction(client *mcp.Client, shortcut, value string) {
	if shortcut == "" || client == nil {
		return
	}
	client.Dispatch(context.Background(), "tinywasm/action", &ActionArgs{
		Key:   shortcut,
		Value: value,
	})
}
