# PLAN: Fix SSE State Refresh Not Triggering UI Re-render

## Problem

When the daemon sends a `TypeStateRefresh` signal (HandlerType == 0) via SSE, the client TUI
calls `RefreshUI()` **before** the async state fetch callback completes. The footer shows stale
state (empty) because the re-render happens before `FieldHandlers` is updated.

This is a secondary issue related to the primary bug in `tinywasm/app` (callback ordering).
Even after the primary bug is fixed, this timing gap can cause a brief flash of an empty footer,
or in slow-network conditions, permanently empty handlers.

## Root Cause

**File:** `sse_client.go` — two locations

### Location 1: `handleLogEvent` (HandlerType == 0 path)

```go
// sse_client.go — handleLogEvent
if dto.HandlerType == 0 {
    h.fetchAndReconstructState()  // async: callback runs later
    h.RefreshUI()                  // called NOW, before callback runs → stale render
    return
}
```

`fetchAndReconstructState` uses `h.mcpClient().Call(...)` which is async (callback-based).
`RefreshUI()` fires immediately after the async call is dispatched, before the state has been
fetched and `FieldHandlers` updated. The footer renders with the old (possibly empty) state.

### Location 2: `fetchAndReconstructState` callback

```go
func (h *DevTUI) fetchAndReconstructState() {
    h.mcpClient().Call(context.Background(), "tinywasm/state", nil, func(result []byte, err error) {
        ...
        h.clearRemoteHandlers()
        h.reconstructRemoteHandlers(entries)
        // ← No RefreshUI() call here!
    })
}
```

After `reconstructRemoteHandlers` populates `FieldHandlers`, `RefreshUI()` is never called from
within the callback. The footer only updates on the next tick event (up to 1 second delay) or
the next incoming log message.

## Affected Files

| File | Lines | Change |
|---|---|---|
| `sse_client.go` | `fetchAndReconstructState` callback | Add `h.RefreshUI()` after `reconstructRemoteHandlers` |
| `sse_client.go` | `handleLogEvent` HandlerType==0 block | Remove the premature `h.RefreshUI()` call |

## Fix

### `fetchAndReconstructState` — add RefreshUI inside the callback

```go
func (h *DevTUI) fetchAndReconstructState() {
    h.mcpClient().Call(context.Background(), "tinywasm/state", nil, func(result []byte, err error) {
        if err != nil || result == nil {
            return
        }
        var entries []StateEntry
        if err := json.Unmarshal(result, &entries); err != nil {
            return
        }
        h.clearRemoteHandlers()
        h.reconstructRemoteHandlers(entries)
        h.RefreshUI()  // ← trigger re-render after FieldHandlers is populated
    })
}
```

### `handleLogEvent` — remove the premature RefreshUI

```go
if dto.HandlerType == 0 {
    h.fetchAndReconstructState()
    // h.RefreshUI() is now called inside fetchAndReconstructState callback
    return
}
```

## Why This Works

- `RefreshUI()` is now called inside the async callback, after `FieldHandlers` has been updated
- The footer re-renders immediately when state arrives, not on the next tick
- Removing the premature `RefreshUI()` from `handleLogEvent` avoids a redundant empty render

## Execution Steps

1. Open `sse_client.go`
2. In `fetchAndReconstructState`: add `h.RefreshUI()` as the last line of the async callback,
   after `h.reconstructRemoteHandlers(entries)`
3. In `handleLogEvent`: remove the `h.RefreshUI()` call from the `HandlerType == 0` block
4. Run existing tests: `go test ./...`
5. Verify with integration test or manual run that footer appears immediately on connect

## Dependency

This fix should be applied **after** the primary fix in `tinywasm/app` (`start.go` callback
ordering). Both fixes together ensure:

1. The daemon sends `PublishStateRefresh()` only after handlers are registered (app fix)
2. The client re-renders the footer immediately after receiving the state (devtui fix)
