# Plan: tinywasm/devtui — Migrate REST calls to JSON-RPC 2.0

← Requires: [mcp PLAN](../../mcp/docs/PLAN.md) + [app PLAN](../../app/docs/PLAN.md) executed first

## References
- `devtui/sse_client.go` — SSE client + `/state` + `/action` REST calls
- `devtui/remote_handler.go` — `postAction()` REST call
- `devtui/userKeyboard.go` — Ctrl+C REST call
- `devtui/init.go` — DevTUI struct + TuiConfig
- `devtui/mcp.go` — existing MCP tool integration (already imports `mcp` package)

---

## Development Rules
- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.**
- **No global state.** Use DI via interfaces.
- **Standard library only** in test assertions.
- **Test runner:** `gotest`. **Publish:** `gopush`.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".

---

## Problem Summary

`devtui` communicates with the daemon via plain REST calls that no longer exist
after mcp PLAN executes:

| Call site | Current (REST) | After mcp PLAN (JSON-RPC) |
|-----------|---------------|--------------------------|
| `userKeyboard.go` | `POST /action?key=stop` | `mcp.Client.Call("tinywasm/action", ...)` |
| `remote_handler.go` | `POST /action` form | `mcp.Client.Call("tinywasm/action", ...)` |
| `sse_client.go` | `GET /state` | `mcp.Client.Call("tinywasm/state", ...)` |
| `sse_client.go` | `GET /logs` SSE | **unchanged** — SSE stays HTTP |

Additionally: secured daemon connections require `Authorization: Bearer <apiKey>`
header on all `/mcp` requests and the SSE `/logs` request.

**Key design principle:** `devtui` is a pure library. It does NOT manage, generate,
or persist API keys. The `APIKey` field in `TuiConfig` is always set by the
consuming application (`tinywasm/app`), which orchestrates key lifecycle.

`devtui` already imports `tinywasm/mcp` (via `devtui/mcp.go`). The reusable
`mcp.Client` type (added in mcp PLAN) is used directly — no new JSON-RPC helper
files are created in devtui.

---

## `DevTUI` struct changes (`init.go`)

Add `APIKey` to `TuiConfig` and `apiKey` to `DevTUI`:

```go
type TuiConfig struct {
    // ... existing fields ...
    APIKey string // Bearer token for secured daemon; set by app, empty = open/local
}

type DevTUI struct {
    // ... existing fields ...
    apiKey string // stored from TuiConfig.APIKey
}
```

`apiKey` is set in the constructor from `TuiConfig.APIKey`.

---

## `mcpClient()` helper in `sse_client.go`

Derives base URL from `ClientURL` (which points to `/logs`) and builds an
`mcp.Client` with the stored API key. The `/mcp` path is added by `mcp.NewClient`
internally — not hardcoded in devtui.

```go
// mcpClient builds a stateless JSON-RPC client targeting the daemon's /mcp endpoint.
// ClientURL = "http://host:port/logs" → base URL = "http://host:port"
func (h *DevTUI) mcpClient() *mcp.Client {
    baseURL := strings.TrimSuffix(h.ClientURL, "/logs")
    return mcp.NewClient(baseURL, h.apiKey)
}
```

---

## MODIFY: `devtui/sse_client.go`

### `fetchAndReconstructState` — GET /state → JSON-RPC `tinywasm/state`

```go
// Before:
func (h *DevTUI) fetchAndReconstructState(baseURL string) {
    resp, err := http.Get(baseURL + "/state")
    if err != nil || resp.StatusCode != 200 { return }
    defer resp.Body.Close()
    var entries []StateEntry
    json.NewDecoder(resp.Body).Decode(&entries)
    ...
}

// After — callback-based (mcp.Client uses tinywasm/fetch, async in WASM + stdlib):
func (h *DevTUI) fetchAndReconstructState() {
    h.mcpClient().Call("tinywasm/state", map[string]string{}, func(result []byte, err error) {
        if err != nil || result == nil { return }
        var entries []StateEntry
        if err := json.Decode(result, &entries); err != nil { return }
        h.clearRemoteHandlers()
        h.reconstructRemoteHandlers(entries)
    })
}
```

### `handleLogEvent` — detect state-refresh signal

The daemon sends a lightweight `{"handler_type": 0}` signal (no state payload)
when handler state changes (project started/stopped). `handleLogEvent` checks for
this reserved marker and re-fetches state via JSON-RPC instead of rendering it as
a log entry. `handleStateEvent` (which parsed full state JSON from SSE) is removed.

```go
// handleLogEvent processes a plain SSE data line.
func (h *DevTUI) handleLogEvent(data string) {
    var dto tabContentDTO
    if err := json.Unmarshal([]byte(data), &dto); err != nil { return }

    // HandlerType 0 = TypeStateRefresh signal from daemon
    if dto.HandlerType == 0 {
        h.fetchAndReconstructState()
        h.RefreshUI()
        return
    }
    // ... existing log rendering logic unchanged ...
}
```

### Remove `handleStateEvent`

`handleStateEvent` is deleted — the SSE `event: state` branch in the event loop
switch is removed. All SSE messages now flow through `handleLogEvent`.
The `event:` SSE prefix is no longer used.

### SSE request — add API key header

```go
req.Header.Set("Accept", "text/event-stream")
req.Header.Set("Cache-Control", "no-cache")
req.Header.Set("Connection", "keep-alive")
if h.apiKey != "" {
    req.Header.Set("Authorization", "Bearer "+h.apiKey)
}
```

---

## MODIFY: `devtui/remote_handler.go`

### `postAction` — `http.PostForm /action` → JSON-RPC `tinywasm/action`

```go
// Before:
func postAction(baseURL, shortcut, value string) {
    if shortcut == "" { return }
    go http.PostForm(baseURL+"/action",
        url.Values{"key": {shortcut}, "value": {value}})
}

// After:
// postAction sends a tinywasm/action JSON-RPC call to the daemon (fire-and-forget).
// client is built by the caller from DevTUI.mcpClient().
func postAction(client *mcp.Client, shortcut, value string) {
    if shortcut == "" { return }
    client.Dispatch("tinywasm/action", map[string]string{
        "key":   shortcut,
        "value": value,
    })
}
```

### `newRemoteField` signature update

```go
// Before:
func newRemoteField(entry StateEntry, actionBase string, ts *tabSection) *field

// After:
func newRemoteField(entry StateEntry, client *mcp.Client, ts *tabSection) *field
```

All `postAction` calls inside closures pass the `client` argument.
Update `reconstructRemoteHandlers` in `sse_client.go` to pass `h.mcpClient()`
instead of `h.actionBaseURL()`.

---

## MODIFY: `devtui/userKeyboard.go`

### Ctrl+C in client mode — REST → JSON-RPC

```go
// Before:
go func() {
    targetURL := h.actionBaseURL() + "/action?key=stop&value="
    http.Post(targetURL, "application/json", nil)
}()
close(h.ExitChan)
return false, tea.Sequence(tea.ExitAltScreen, tea.Quit)

// After:
h.mcpClient().Dispatch("tinywasm/action", map[string]string{"key": "stop", "value": ""})
close(h.ExitChan)
return false, tea.Sequence(tea.ExitAltScreen, tea.Quit)
```

---

## Files to Create / Modify

| File | Action | Description |
|------|--------|-------------|
| `devtui/init.go` | **MODIFY** | Add `APIKey` to `TuiConfig`; `apiKey` to `DevTUI`; set in constructor |
| `devtui/sse_client.go` | **MODIFY** | Add `mcpClient()`; update `fetchAndReconstructState`; add auth header to SSE; update `reconstructRemoteHandlers` call |
| `devtui/remote_handler.go` | **MODIFY** | `postAction` → JSON-RPC via `mcp.Client`; update `newRemoteField` signature |
| `devtui/userKeyboard.go` | **MODIFY** | Ctrl+C → `mcpClient().Call("tinywasm/action", ...)` |
| `devtui/mcp_test.go` | **MODIFY** | Mock `/mcp` JSON-RPC instead of `/action`/`/state` |
| `devtui/client_mode_test.go` | **MODIFY** | Assert JSON-RPC body format; verify `APIKey` wires into auth header |

**No new files created.** `devtui` reuses `mcp.Client` from the `tinywasm/mcp`
package already imported by `devtui/mcp.go`.

---

## Execution Steps

### Step 1 — Prerequisite: update tinywasm/mcp to v0.0.17
```bash
go get github.com/tinywasm/mcp@v0.0.17
```
Confirm the following exist at v0.0.17:
- `mcp.NewClient(baseURL, apiKey string) *Client`
- `(*Client).Call(ctx context.Context, method string, params any, callback func([]byte, error))`
- `(*Client).Dispatch(ctx context.Context, method string, params any)`
- Daemon (app PLAN) serves `tinywasm/action` and `tinywasm/state` as JSON-RPC on POST `/mcp`

### Step 2 — Modify `devtui/init.go`
Add `APIKey` to config + struct.

### Step 3 — Modify `devtui/sse_client.go`
Add `mcpClient()`, update `fetchAndReconstructState`, add auth header, update
`reconstructRemoteHandlers` callers.

### Step 4 — Modify `devtui/remote_handler.go`
Update `postAction` + `newRemoteField`.

### Step 5 — Modify `devtui/userKeyboard.go`

### Step 6 — Update tests

### Step 7 — Run tests and publish
```bash
gotest
gopush 'feat: migrate REST action/state calls to JSON-RPC 2.0 via mcp.Client, add API key auth header'
```

---

## Test Strategy

| Test | Validates |
|------|-----------|
| `TestMCPClient_SendsCorrectBody` | `mcp.Client.Call` sends `jsonrpc:"2.0"`, method, params (tested in mcp pkg, not here) |
| `TestClientMode_CtrlC_SendsJSONRPCStop` | POST /mcp with `tinywasm/action` + `key=stop` |
| `TestFetchState_CallsJSONRPCState` | Calls `tinywasm/state` on /mcp, parses `[]StateEntry` |
| `TestHandleLogEvent_StateRefreshSignal_FetchesState` | `handler_type=0` in SSE data → calls `fetchAndReconstructState()` |
| `TestHandleLogEvent_NormalEntry_NoStateRefresh` | Normal log entry → no JSON-RPC call triggered |
| `TestPostAction_SendsJSONRPCAction` | `postAction` sends JSON-RPC body with key+value |
| `TestSSEConnect_WithAPIKey_SetsHeader` | GET /logs has `Authorization: Bearer` when APIKey set |
| `TestSSEConnect_NoAPIKey_NoAuthHeader` | No auth header when APIKey empty |
| `TestTuiConfig_APIKey_StoredInDevTUI` | `TuiConfig.APIKey` propagates to `DevTUI.apiKey` |
| Existing keyboard/shortcut tests | No regression |
