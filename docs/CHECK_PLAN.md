# PLAN: Clean Terminal Shutdown on Ctrl+C

## Problem

When the user presses Ctrl+C, the terminal is left with TUI artifacts (colors, boxes, log lines)
visible after exit. Root cause: `close(ExitChan)` fires before bubbletea executes `ExitAltScreen`,
so goroutines that listen on the channel write to stdout/Logger while the alternate screen is still
active or mid-restore.

Secondary bugs found during analysis:

- The SSE client goroutine blocks on `reader.ReadString('\n')` — the non-blocking `select` on
  ExitChan at line 95 is never reached while the read is pending. Can take seconds to unblock.
- Internal Logger calls in `sse_client.go` fire during the retry delay after ExitChan closes,
  writing to the terminal after it has been restored.
- `ExitChan` ownership is inverted: the channel is created by the caller (tinywasm/app) but
  closed by devtui — violates the Go principle that only the creator should close a channel.
- `tea.KeyCtrlC` is checked twice in `handleNormalModeKeyboard`: the ClientMode guard at lines
  171-178 duplicates the general guard at lines 181-184.

## Design Decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| ExitChan in TuiConfig | **Remove** | Eliminates inverted ownership; caller no longer needs to pass it |
| External stop API | `Shutdown()` method | Single entry point; caller and OS signals both use it |
| `Shutdown()` implementation | Sends internal `shutdownMsg{}` → handled in `Update()` | Only `Update()` can return a `tea.Cmd` sequence; calling `p.Quit()` directly skips `ClearScreen` |
| SSE context | Created in `Init()`, passed to `startSSEClient(ctx)` | Eliminates race between goroutine writing `h.sseCancel` and `Shutdown()` reading it |
| SSE cancellation | `http.NewRequestWithContext(ctx, ...)` | When context is cancelled, `client.Do()` returns immediately — unblocks `ReadString` without waiting for next byte |
| Caller notification | `Start()` blocks until full cleanup | No extra `Done()` channel needed in public API |
| Shutdown timeout | 2 seconds then `os.Exit(0)` | Terminal is already clean before timeout starts; `os.Exit(0)` is safe |
| Logger during shutdown | Guard internal uses with `isShuttingDown` flag | External Logger is caller's responsibility |
| Screen cleanup sequence | `ClearScreen → ExitAltScreen → Quit` | `ClearScreen` wipes alt-screen content before switching back to normal terminal |

## Shutdown Sequence (after fix)

```
Ctrl+C or Shutdown() called
  1. Update() receives shutdownMsg (or tea.KeyCtrlC)
  2. h.isShuttingDown.Store(true)       ← guards internal Logger calls
  3. h.sseCancel()                       ← cancels ctx → client.Do() returns immediately
  4. return tea.Sequence(
         tea.ClearScreen,                ← wipes alt-screen content
         tea.ExitAltScreen,              ← restores normal terminal
         tea.Quit,                       ← stops bubbletea loop
     )
     ← NO close(ExitChan) anywhere in devtui

h.tea.Run() returns in Start()          ← terminal is clean at this point
  5. h.sseWg.Wait() with 2s timeout     ← SSE goroutine already done (ctx cancelled)
  6. if timeout: os.Exit(0)             ← safe: terminal already restored
  7. Start() returns                    ← WaitGroup.Done() notifies caller

Caller (runClient in tinywasm/app)
  8. caller closes its own exitChan     ← server/watcher/browser stop
  9. POST "quit" to daemon (500ms)      ← best-effort, non-blocking
```

## Affected Files

| File | Change |
|------|--------|
| `models.go` | Add `isShuttingDown atomic.Bool`, `sseCancel context.CancelFunc`, `sseWg sync.WaitGroup` to `DevTUI`; remove `ExitChan` from `TuiConfig` |
| `init.go` | Create SSE context in `Init()`; update `Start()` to wait for SSE goroutine with timeout after `h.tea.Run()` returns; add `Shutdown()` method; add `shutdownMsg` type |
| `update.go` | Handle `shutdownMsg` in `Update()` with full cleanup sequence |
| `sse_client.go` | Accept `context.Context` param; use `http.NewRequestWithContext`; guard Logger with `isShuttingDown`; call `h.sseWg.Done()` on return |
| `userKeyboard.go` | Replace `close(h.ExitChan)` with `go h.tea.Send(shutdownMsg{})`; remove duplicate ClientMode Ctrl+C guard |

## Implementation Steps

### Step 1 — `models.go`: extend DevTUI struct and clean TuiConfig

Add to `DevTUI` struct:
```go
import (
    "context"
    "sync"
    "sync/atomic"
)

isShuttingDown atomic.Bool
sseCancel       context.CancelFunc // cancels SSE HTTP request context
sseWg           sync.WaitGroup     // tracks SSE goroutine
```

Remove `ExitChan chan bool` from `TuiConfig`.

Initialize `sseCancel` to a no-op in `NewTUI` (safety: Shutdown() callable before Init()):
```go
_, noopCancel := context.WithCancel(context.Background())
tui.sseCancel = noopCancel
```

### Step 2 — `init.go`: wire SSE context, update Start(), add Shutdown()

In `Init()`, create the cancellable context BEFORE launching the goroutine to avoid races:
```go
func (h *DevTUI) Init() tea.Cmd {
    if h.ClientMode && h.ClientURL != "" {
        ctx, cancel := context.WithCancel(context.Background())
        h.sseCancel = cancel   // set before goroutine starts — no race
        h.sseWg.Add(1)
        go h.startSSEClient(h.ClientURL, ctx)
    }
    return tea.Batch(
        h.listenToMessages(),
        h.tickEverySecond(),
        h.cursorTick(),
    )
}
```

In `Start()`, after `h.tea.Run()`:
```go
func (h *DevTUI) Start(args ...any) {
    // ... existing WaitGroup and setup logic ...

    if _, err := h.tea.Run(); err != nil {
        // errors go to file log (app's Logger responsibility)
    }

    // Terminal is restored here. Now drain the SSE goroutine.
    done := make(chan struct{})
    go func() {
        h.sseWg.Wait()
        close(done)
    }()
    select {
    case <-done:
        // clean exit
    case <-time.After(2 * time.Second):
        os.Exit(0) // terminal already clean; force exit
    }
}
```

Add `Shutdown()` and `shutdownMsg` (also needed in `update.go`):
```go
// shutdownMsg triggers a clean exit through the normal Update() path.
// This ensures the full ClearScreen → ExitAltScreen → Quit sequence runs.
type shutdownMsg struct{}

// Shutdown signals the TUI to stop gracefully.
// Safe to call from any goroutine (OS signal handlers, external callers).
func (h *DevTUI) Shutdown() {
    if h.tea != nil {
        go h.tea.Send(shutdownMsg{})
    }
}
```

### Step 3 — `update.go`: handle shutdownMsg in Update()

Add case to the `switch msg := msg.(type)` block:
```go
case shutdownMsg:
    h.isShuttingDown.Store(true)
    h.sseCancel()
    return h, tea.Sequence(tea.ClearScreen, tea.ExitAltScreen, tea.Quit)
```

### Step 4 — `sse_client.go`: cancellable HTTP request

Change signature and use context throughout:
```go
func (h *DevTUI) startSSEClient(url string, ctx context.Context) {
    defer h.sseWg.Done()

    // ... existing URL normalization ...

    client := &http.Client{Timeout: 0}
    retryDelay := 1 * time.Second

    for {
        // Check for cancellation before each connection attempt
        select {
        case <-ctx.Done():
            return
        default:
        }

        if !h.isShuttingDown.Load() && h.Logger != nil {
            h.Logger("Connecting to SSE stream at", url)
        }

        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            if ctx.Err() != nil {
                return // context cancelled
            }
            if !h.isShuttingDown.Load() && h.Logger != nil {
                h.Logger("Error creating SSE request:", err)
            }
            time.Sleep(retryDelay)
            continue
        }

        req.Header.Set("Accept", "text/event-stream")
        req.Header.Set("Cache-Control", "no-cache")
        req.Header.Set("Connection", "keep-alive")
        if h.apiKey != "" {
            req.Header.Set("Authorization", "Bearer "+h.apiKey)
        }

        resp, err := client.Do(req)
        if err != nil {
            if ctx.Err() != nil {
                return // context cancelled — clean exit, no log
            }
            if !h.isShuttingDown.Load() && h.Logger != nil {
                h.Logger("Error connecting to SSE server:", err)
            }
            time.Sleep(retryDelay)
            continue
        }

        reader := bufio.NewReader(resp.Body)
        var currentEvent string

        for {
            // Non-blocking cancellation check before each read
            select {
            case <-ctx.Done():
                resp.Body.Close()
                return
            default:
            }

            line, err := reader.ReadString('\n')
            if err != nil {
                resp.Body.Close()
                if ctx.Err() != nil {
                    return // context cancelled mid-stream
                }
                if !h.isShuttingDown.Load() && h.Logger != nil {
                    h.Logger("Error reading SSE stream:", err)
                }
                break // reconnect
            }

            // ... existing line processing logic (unchanged) ...
        }

        // resp.Body already closed above in all paths
        time.Sleep(retryDelay)
    }
}
```

Key change: `http.NewRequestWithContext(ctx, ...)` — when context is cancelled, `client.Do()`
returns immediately with `context.Canceled`. This eliminates the blocking `ReadString` problem:
the HTTP client closes the underlying connection, causing `ReadString` to return an error.
The `ctx.Err() != nil` check then returns cleanly without logging.

### Step 5 — `userKeyboard.go`: fix Ctrl+C handler

Replace both Ctrl+C blocks in `handleNormalModeKeyboard` with a single unified handler:
```go
func (h *DevTUI) handleNormalModeKeyboard(msg tea.KeyMsg) (bool, tea.Cmd) {
    if msg.Type == tea.KeyCtrlC {
        if h.ClientMode && h.ClientURL != "" {
            // Best-effort: tell daemon to stop the project
            h.mcpClient().Dispatch(context.Background(), "tinywasm/action", &ActionArgs{Key: "stop"})
        }
        // Trigger shutdown through Update() to get full screen cleanup sequence
        go h.tea.Send(shutdownMsg{})
        return false, nil
    }
    // ... rest of handler unchanged ...
}
```

Removed:
- `close(h.ExitChan)` — devtui no longer owns or closes ExitChan
- Duplicate `if h.ClientMode && h.ClientURL != ""` Ctrl+C guard (merged into single block)
- Direct `tea.Sequence(tea.ExitAltScreen, tea.Quit)` — moved to Update() via shutdownMsg

## Dependency

Self-contained within devtui. No external package changes required.
Consumers that currently pass `ExitChan` via `TuiConfig` must update their call sites
after this plan is applied and the module is published.

## Verification

```bash
go test ./...   # existing tests must pass
go vet ./...    # no new warnings
```

Manual: run `tinywasm`, press Ctrl+C — terminal must return completely clean with no artifacts,
cursor visible, previous terminal content restored, no delay.
