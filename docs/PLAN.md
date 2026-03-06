# Plan: DevTUI — Fix Action URL Bug + Ctrl+C Stop Everything

## References

- `userKeyboard.go` — keyboard handler (both bugs are here)
- `sse_client.go` — `actionBaseURL()` helper (already correct)
- `remote_handler.go` — `postAction()` (already uses correct base URL)
- `init.go` — `TuiConfig.ClientURL` field

---

## Development Rules

- **SRP:** Every file must have a single, well-defined purpose.
- **Max 500 lines per file.**
- **Standard library only** in test assertions.
- **Test runner:** `gotest`. **Publish:** `gopush`.
- **Language:** Plans in English, chat in Spanish.
- **No code changes** until the user says "ejecuta" or "ok".

---

## Problem 1: Wrong Action URL (critical bug)

In `userKeyboard.go:182`, client mode sends key presses to a wrong URL:

```go
// ACTUAL — BROKEN:
targetURL := h.ClientURL + "/action?key=" + url.QueryEscape(key)
// h.ClientURL = "http://localhost:3030/logs"
// Result:  "http://localhost:3030/logs/action?key=q" → 404 ALWAYS

// CORRECT:
targetURL := h.actionBaseURL() + "/action?key=" + url.QueryEscape(key)
// actionBaseURL() = strings.TrimSuffix(h.ClientURL, "/logs") → "http://localhost:3030"
// Result:  "http://localhost:3030/action?key=q" ← correct
```

`actionBaseURL()` already exists in `sse_client.go:31` and is used correctly by
`fetchAndReconstructState` and `newRemoteField`. The keyboard handler just missed it.

---

## Problem 2: TUI not cleaned up on exit

Currently in both standalone and client mode, the TUI content is left printed on
the terminal when the app exits. The alt-screen must be exited properly.

In client mode (`Ctrl+C`), the code does:
```go
return false, tea.Sequence(tea.ExitAltScreen, tea.Quit)
```
This is correct. However the TUI may not reach this handler if `ExitChan` is closed
externally (e.g. daemon dies), because `Update()` receives `tea.Quit` from the channel
listener but `ExitAltScreen` is not issued. This needs to be verified and fixed if needed.

---

## Problem 3: `q` as stop key — wrong design

The current `KeyRunes` interception forwards ALL single-char keys to the daemon.
This is fragile: any key typed accidentally gets forwarded, and single letters can
conflict with future handler shortcuts.

**Correct design:** `Ctrl+C` is the universal "kill" convention. In client mode it should:
1. Signal the daemon to stop the project (`POST /action?key=stop`)
2. Exit alt-screen cleanly (restore terminal)
3. Close the TUI process

No other keys should be forwarded generically. Handler shortcuts use `remoteField.postAction()`
which already works correctly via the shortcut key system.

---

## Problem 4: Daemon action key mismatch

The daemon's `OnUIAction` currently handles `"q"` → `stopProject()`.
After this fix, client sends `key=stop`. This is a coordinated change with `app/daemon.go`.

---

## Files to Modify

### `devtui/userKeyboard.go` — client mode interception block

**Before** (lines 173-198):
```go
if h.ClientMode && h.ClientURL != "" {
    switch msg.Type {
    case tea.KeyRunes:
        if len(msg.Runes) == 1 {
            key := string(msg.Runes[0])
            go func() {
                targetURL := h.ClientURL + "/action?key=" + url.QueryEscape(key) // BUG
                resp, err := http.Post(targetURL, "application/json", nil)
                ...
            }()
            return false, nil
        }
    case tea.KeyCtrlC:
        return false, tea.Sequence(tea.ExitAltScreen, tea.Quit)
    }
}
```

**After**:
```go
if h.ClientMode && h.ClientURL != "" {
    switch msg.Type {
    case tea.KeyCtrlC:
        // Stop everything: signal daemon to stop project, clean terminal, close TUI
        go http.PostForm(h.actionBaseURL()+"/action",
            url.Values{"key": {"stop"}, "value": {""}})
        close(h.ExitChan)
        return false, tea.Sequence(tea.ExitAltScreen, tea.Quit)
    }
}
```

Changes:
- Remove the `KeyRunes` block entirely (no more generic key forwarding)
- `Ctrl+C` now: sends stop action to daemon + exits alt-screen + quits

### `devtui/shortcuts.go` — update help content

Replace the quit section in `generateHelpContent()`:
```go
// Before:
"quit", `:
  • Ctrl+C         - `, "quit", `
`

// After:
"quit", `:
  • Ctrl+C  - `, "stop", " & quit\n",
```

Only one entry. No Ctrl+L, no "TUI only" variant.

### `app/daemon.go` — coordinated action key change

```go
// Before:
case "q":
    dtp.stopProject()

// After:
case "stop":
    dtp.stopProject()
```

---

## Execution Steps

### Step 1 — Fix URL bug in `userKeyboard.go`
Change `h.ClientURL + "/action?key="` → `h.actionBaseURL() + "/action?key="`.
This alone fixes all existing action routing (remote handler shortcuts also benefit).

### Step 2 — Replace `KeyRunes` block with clean `Ctrl+C` handler

### Step 3 — Update quit help text in `shortcuts.go`

### Step 4 — Update `app/daemon.go`: `case "q"` → `case "stop"`

### Step 5 — Run tests and publish devtui
```bash
gotest
gopush 'fix: correct action URL, Ctrl+C stops project and cleans terminal'
```

### Step 6 — Run tests and publish app
```bash
cd ../app && gotest && gopush 'fix: handle stop action key from devtui Ctrl+C'
```

---

## Test Strategy

- `TestClientMode_CtrlC_SendsStopAction` — mock HTTP server receives `POST /action?key=stop`
  when `Ctrl+C` is pressed in client mode
- `TestClientMode_KeyRunes_NoLongerForwarded` — regular key presses do NOT trigger HTTP calls
- Existing keyboard tests must continue passing
