//go:build !wasm

package devtui

import (
	"net/http"
	"net/url"
)

// newRemoteField constructs a *field populated from a StateEntry.
// Uses anyHandler closures directly — no intermediate interface types needed.
// The entry pointer is captured so optimistic value updates stay in sync.
func newRemoteField(entry StateEntry, actionBase string, ts *tabSection) *field {
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
				postAction(actionBase, e.Shortcut, v)
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
			executeFunc:  func() { postAction(actionBase, e.Shortcut, "") },
			changeFunc:   func(_ string) { postAction(actionBase, e.Shortcut, "") },
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
				postAction(actionBase, e.Shortcut, v)
			},
		}
	default:
		return nil // HandlerTypeLoggable — no field, logs arrive via SSE
	}

	return &field{handler: anyH, parentTab: ts, isRemote: true}
}

// postAction fires a non-blocking POST to the daemon action endpoint.
func postAction(baseURL, shortcut, value string) {
	if shortcut == "" {
		return
	}
	go http.PostForm(baseURL+"/action",
		url.Values{"key": {shortcut}, "value": {value}})
}
