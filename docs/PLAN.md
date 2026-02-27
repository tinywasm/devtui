# Plan: Handler State Domain — StateEntry & Remote Handler Reconstruction

## SRP Contract

`devtui` **owns the handler state domain**. It defines `StateEntry` (the wire format
for handler metadata), the `HandlerType*` constants, and the client-side
reconstruction logic. No other package defines these concepts.

## Problem

In client mode, `DevTUI` creates tab sections (BUILD, DEPLOY) but they have no
`fieldHandlers`. The user sees a passive log viewer with no interactive controls,
because:

1. `sse_client.go` ignores `event:` lines in the SSE stream — state update events
   are silently dropped.
2. No one fetches the daemon's `/state` snapshot on connect.
3. There is no `RemoteHandler` type to represent a daemon-side handler in the
   client TUI.

## Solution

### 1. `StateEntry` Struct (new file `state_entry.go`)

`devtui` is the single source of truth for the handler state wire format:

```go
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
```

These constants replace the implicit magic numbers previously spread across mcpserve
and sse_client.go.

### 2. SSE Event Name Parsing in `sse_client.go`

SSE streams can carry named events:
```
event: state
data: {...StateEntry JSON...}

data: {...LogEntry JSON...}   ← no event name = default = log
```

The parser must track `currentEvent string` across line reads:

```go
var currentEvent string
for {
    line, err := reader.ReadString('\n')
    line = strings.TrimSpace(line)

    if strings.HasPrefix(line, "event:") {
        currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
        continue
    }
    if strings.HasPrefix(line, "data:") {
        data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
        switch currentEvent {
        case "state":
            h.handleStateEvent(data)
        default: // "" or "log"
            h.handleLogEvent(data) // existing log processing logic
        }
        currentEvent = "" // reset after data line
    }
}
```

### 3. Initial State Fetch on Connect

At the start of `startSSEClient`, before entering the read loop, fetch the daemon's
state snapshot and reconstruct the interactive fields.

`h.ClientURL` is the full SSE URL (e.g., `http://localhost:3030/logs`).
The base URL is derived by stripping the `/logs` suffix:

```go
func (h *DevTUI) startSSEClient(url string) {
    // url is "http://localhost:3030/logs"
    // Step 1: reconstruct interactive handlers from daemon state
    h.fetchAndReconstructState(h.actionBaseURL())
    // Step 2: subscribe to SSE stream for live events
    // ... existing connect loop ...
}

// actionBaseURL strips the /logs suffix from ClientURL to get the daemon base URL.
// Used for both GET /state and POST /action requests.
func (h *DevTUI) actionBaseURL() string {
    return strings.TrimSuffix(h.ClientURL, "/logs")
}

func (h *DevTUI) fetchAndReconstructState(baseURL string) {
    // baseURL is "http://localhost:3030"
    resp, err := http.Get(baseURL + "/state") // GET /state
    if err != nil || resp.StatusCode != 200 {
        return // daemon may not support state yet, degrade gracefully
    }
    defer resp.Body.Close()
    var entries []StateEntry
    if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
        return
    }
    h.reconstructRemoteHandlers(entries)
}
```

### 4. Handler Reconstruction

```go
func (h *DevTUI) reconstructRemoteHandlers(entries []StateEntry) {
    for _, entry := range entries {
        var section *tabSection
        for _, s := range h.TabSections {
            if s.title == entry.TabTitle {
                section = s
                break
            }
        }
        if section == nil {
            continue // section not registered in this client
        }
        f := newRemoteField(entry, h.actionBaseURL(), section)
        if f != nil {
            section.addFields(f) // bypasses addHandler type switch — field already built
        }
    }
}
```

`actionBaseURL()` is defined in §3 — strips `/logs` from `h.ClientURL`.

### 5. Live State Updates via SSE (Phase 2)

> **Scope note:** The snapshot fetch in §3 covers the primary use case (client connects
> and sees current state). Live push via `event: state` is a future enhancement —
> the SSE parser (§2) is already wired to call `handleStateEvent`, but no daemon-side
> trigger is defined in this iteration. The handler parses but takes no action until
> Phase 2 implements the push side.

When `handleStateEvent(data string)` is eventually called (live `event: state`):
- Unmarshal to `StateEntry`
- Find the matching field in `tabSection.fieldHandlers` by `HandlerName`
- Update `remoteBase.entry.Value` to the new value
- Call `h.RefreshUI()` to re-render

### 6. New File: `remote_handler.go`

> **Why no new types?**
> `remote_handler.go` lives inside the `devtui` package — it has direct access to
> `anyHandler` and `field`. Defining a `remoteBase` + 4 subtype hierarchy just to
> satisfy external interfaces (`HandlerDisplay`, `HandlerEdit`, etc.) adds an entire
> type layer whose only job is adapting a `StateEntry` to pass through `ts.addHandler()`.
>
> The correct approach is to construct `*anyHandler` with closures directly from
> `StateEntry` — the same pattern used internally by `NewEditHandler`, `NewDisplayHandler`,
> etc. This reuses the existing `anyHandler` + `field` + `ts.addFields()` infrastructure
> without any new types.

```go
//go:build !wasm

package devtui

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

    return &field{handler: anyH, parentTab: ts}
}

// postAction fires a non-blocking POST to the daemon action endpoint.
func postAction(baseURL, shortcut, value string) {
    if shortcut == "" {
        return
    }
    go http.PostForm(baseURL+"/action",
        url.Values{"key": {shortcut}, "value": {value}})
}
```

`reconstructRemoteHandlers` calls `ts.addFields(newRemoteField(entry, h.actionBaseURL(), section))`
directly — bypassing `ts.addHandler()` which would require implementing the external
interfaces unnecessarily.

## Files to Change

| File | Change |
|------|--------|
| `state_entry.go` | **New** — `StateEntry` struct + `HandlerType*` constants |
| `sse_client.go` | Parse `event:` lines; dispatch to `handleLogEvent`/`handleStateEvent` |
| `sse_client.go` | Call `fetchAndReconstructState` before SSE loop |
| `sse_client.go` | Add `handleStateEvent`, `fetchAndReconstructState`, `reconstructRemoteHandlers` |
| `remote_handler.go` | **New** (`//go:build !wasm`) — `newRemoteField` + `postAction` (no new types) |

## Constraints

- `state_entry.go` has **no build tag** — it is a plain data struct with no OS/network
  dependencies; it must be available in all builds (including WASM)
- `remote_handler.go` must be `//go:build !wasm` (uses `net/http` and `net/url`)
- Remote handlers do NOT implement `Loggable` — logs arrive via SSE log events
- `postAction` is always fire-and-forget (goroutine) — never blocks the UI loop
- `fetchAndReconstructState` degrades gracefully: if `/state` is unavailable, the
  client shows an empty-field TUI (same as today), never crashes

## Test Strategy

- `TestStateEntry_JSONRoundTrip` — marshal/unmarshal produces identical struct
- `TestFetchAndReconstructState_PopulatesFieldHandlers` — mock server returns
  `[]StateEntry`, verify `section.fieldHandlers` has correct count and types
- `TestHandleStateEvent_UpdatesRemoteHandlerValue` — live SSE state event updates
  matching field's `Value()`
- `TestSSEClientParsesEventName_RoutesCorrectly` — stream with `event: state`
  calls `handleStateEvent`, default calls `handleLogEvent`
- `TestRemoteEditHandler_Change_PostsAction` — `Change("8080")` sends
  `POST /action?key=c&value=8080`
- `TestNewRemoteHandler_ReturnsCorrectVariant` — each HandlerType constant

## References

- [DESCRIPTION.md](DESCRIPTION.md)
- [MCP_REFACTOR.md](MCP_REFACTOR.md)
- Wire format producer: `app/docs/PLAN.md`
- Transport: `mcpserve/docs/PLAN.md`
- Handler interfaces: `devtui/interfaces.go`
- Handler iota (reference only, not imported): `devtui/anyHandler.go`
