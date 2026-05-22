# PLAN: Input Section Disappears on StateRefresh with Empty Response

## Status: PENDING

See primary root cause in `tinywasm/app/docs/PLAN_tui_contamination.md`.

## This Library's Issue

`fetchAndReconstructState` in `sse_client.go` calls `clearRemoteHandlers()` even when
the state response is empty (`[]`). This means any transient empty state (caused by a
race condition in the daemon's project lifecycle) permanently wipes the input section
visible to the user.

## Fix

In `sse_client.go`, `fetchAndReconstructState`:

```go
if len(entries) == 0 { return }  // add this guard before clearRemoteHandlers
```

## Test

File: `sse_client_empty_state_test.go`  
Test: `TestFetchAndReconstructState_EmptyResponseDoesNotClearHandlers`

Verifies that existing remote fields survive a `fetchAndReconstructState` call
that returns an empty entries slice.
