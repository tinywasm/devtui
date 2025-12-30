# Unified Handler Architecture

This document describes the new unified architecture for DevTUI where all handlers use `AddHandler` and implement `SetLog` for automatic logging with clean terminal display.

## Vision

- **One entry point**: Only `AddHandler` - remove `AddLogger`
- **Automatic tracking**: By handler `Name()` - remove `MessageTracker`
- **Clean terminal**: Show only the most recent log per handler (ordered newest first)
- **MCP history**: When LLM requests specific handlers, show full history temporarily
- **Transparent logging**: Handlers call their internal `log()`, DevTUI intercepts

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Registration** | `AddHandler` only | Simpler API, one entry point |
| **Loggable interface** | Separate interface | Optional capability, not forced |
| **Tracking** | By handler Name() | Automatic, no MessageTracker needed |
| **Default display** | Last log per handler | Clean terminal, less noise |
| **MCP history** | Implicit when filtering | No `mode` param needed |
| **Developer sync** | Terminal updates with LLM | Both see same state |

## New Interface: Loggable

```go
// Loggable defines optional logging capability for handlers.
// Handlers implementing this receive a logger function from DevTUI.
type Loggable interface {
    Name() string
    SetLog(func(message ...any))
}
```

## Architecture Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Handler (WasmClient, Server, etc.)                         │
│  - Implements Loggable interface                            │
│  - Has internal log func (never nil, starts as no-op)       │
│  - Calls w.log("message") when needed                       │
└───────────────────────────┬─────────────────────────────────┘
                            │ SetLog(devtuiLogger)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  DevTUI (anyHandler)                                        │
│  - Intercepts all log calls                                 │
│  - Stores ALL logs internally (full history)                │
│  - Displays only LAST log per handler (clean terminal)      │
│  - Each handler identified by Name() - no duplicates        │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  MCP Tool: terminal_logs                                    │
│  - section="" → List sections with handlers                 │
│  - section="BUILD" → Last log per handler (default)         │
│  - section="BUILD", handlers=["WASM"] → Full history        │
│  - Temporarily shows history, then resets to clean          │
│  - Developer terminal syncs automatically                   │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Steps

Complete each step in order. Mark `[x]` when done.

### DevTUI Changes (this package)
- [ ] **Step 1**: [Add Loggable Interface](UNIFIED_STEP1_LOGGABLE_INTERFACE.md)
- [ ] **Step 2**: [Update AddHandler for Loggable](UNIFIED_STEP2_ADDHANDLER_LOGGABLE.md)
- [ ] **Step 3**: [Remove AddLogger and MessageTracker](UNIFIED_STEP3_REMOVE_ADDLOGGER.md)
- [ ] **Step 4**: [Implement Clean Terminal Display](UNIFIED_STEP4_CLEAN_DISPLAY.md)
- [ ] **Step 5**: [Update MCP Tool for History](UNIFIED_STEP5_MCP_HISTORY.md)
- [ ] **Step 6**: [Verification and Testing](UNIFIED_STEP6_VERIFICATION.md)

### App Integration (tinywasm/app)
- [ ] **Step 7**: See `tinywasm/app/docs/issues/UNIFIED_HANDLER_INTEGRATION.md`

## Files to Modify

### DevTUI Package (`tinywasm/devtui`)

| File | Action | Description |
|------|--------|-------------|
| `interfaces.go` | MODIFY | Add `Loggable` interface, remove `MessageTracker`, `HandlerLogger` |
| `anyHandler.go` | MODIFY | Add log interception, tracking by Name() |
| `handlerRegistration.go` | MODIFY | Update `AddHandler` for Loggable, remove `AddLogger` |
| `view.go` | MODIFY | Show only last log per handler |
| `mcp.go` | MODIFY | Show full history when handlers specified |

### External Handlers (breaking change)

| Package | File | Description |
|---------|------|-------------|
| `client` | `client.go` | Add `SetLog`, remove Logger from Config |
| `server` | `server.go` | Add `SetLog`, remove Logger from Config |
| `assetmin` | `assetmin.go` | Add `SetLog`, remove Logger from Config |
| `devwatch` | `watcher.go` | Add `SetLog`, remove Logger from Config |
| `devbrowser` | `browser.go` | Add `SetLog`, remove Logger from Config |

## Success Criteria

1. `AddLogger` removed - only `AddHandler` exists
2. `MessageTracker` and `HandlerLogger` interfaces removed
3. All handlers implement `Loggable` with `SetLog`
4. Terminal shows only last log per handler by default
5. MCP shows full history when specific handlers requested
6. Developer terminal syncs with LLM view
7. All tests pass
