# Dynamic Compact Mode & MCP Log Control

This document describes the implementation plan for renaming "Tracking" to "Compact" and enabling dynamic log mode switching via MCP.

## Objective

1. Rename "Tracking" terminology to "Compact" for a more intuitive API
2. Enable dynamic switching between `compact` (clean terminal) and `history` (full debug) modes via MCP
3. Synchronize DevTUI's active tab with the LLM's current focus
4. Enhance `terminal_logs` tool with handler filtering

## Architecture Overview

### Current Flow (Tracking)
```
AddLogger(name, enableTracking=true) → MessageTracker.GetLastOperationID() → Reuses line
AddLogger(name, enableTracking=false) → Always new line
```

### New Flow (Compact)
```
AddLogger(name, compact=true) → anyHandler.currentCompactMode=true → GetCompactID() returns ID → Reuses line
AddLogger(name, compact=false) → anyHandler.currentCompactMode=false → GetCompactID() returns "" → New line

MCP: terminal_logs(mode="history") → Sets currentCompactMode=false dynamically → Full history shown
```

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Terminology** | `compact` instead of `tracking` | More intuitive: compact = clean, history = full |
| **Dynamic Control** | Via `anyHandler.currentCompactMode` | Mutable state per handler |
| **Persistence** | Volatile (reset on restart) | Avoids dirty state across sessions |
| **Tab Sync** | Auto-switch on MCP call | Developer sees what LLM sees |
| **Multi-handler filter** | Support `[]string` | Filter multiple handlers at once |

## Implementation Steps

Complete each step in order. Mark `[x]` when done.

### DevTUI Changes (this package)
- [ ] **Step 1**: [Rename Interfaces - MessageTracker to CompactProvider](COMPACT_STEP1_RENAME_INTERFACES.md)
- [ ] **Step 2**: [Update anyHandler for Dynamic Modes](COMPACT_STEP2_ANYHANDLER_DYNAMIC.md)
- [ ] **Step 3**: [Update AddLogger API](COMPACT_STEP3_ADDLOGGER_API.md)
- [ ] **Step 4**: [Enhance terminal_logs MCP Tool](COMPACT_STEP4_MCP_TOOL.md)
- [ ] **Step 5**: [Verification and Testing](COMPACT_STEP5_VERIFICATION.md)

### App Integration (tinywasm/app)
- [ ] **Step 6**: See `tinywasm/app/docs/issues/COMPACT_MODE_INTEGRATION.md`

## Files to Modify

### DevTUI Package (`tinywasm/devtui`)

| File | Action | Description |
|------|--------|-------------|
| `interfaces.go` | MODIFY | Rename `MessageTracker` to `CompactProvider` |
| `anyHandler.go` | MODIFY | Add `currentCompactMode` field, update methods |
| `handlerRegistration.go` | MODIFY | Rename `enableTracking` to `compact` |
| `tabSection.go` | MODIFY | Update references to new interface names |
| `print.go` | MODIFY | Update references if any |
| `mcp.go` | MODIFY | Add `handlers`, `mode` parameters, tab sync |

### App Package (`tinywasm/app`) - See separate issue

| File | Action | Description |
|------|--------|-------------|
| `section-build.go` | MODIFY | Rename `enableTracking` to `compact` in AddLogger calls |

## Success Criteria

1. All references to `Tracking`/`enableTracking` renamed to `Compact`/`compact`
2. LLM can switch handler modes dynamically via `terminal_logs(mode="history")`
3. Developer's terminal auto-switches to the section viewed by LLM
4. Multi-handler filtering works with `handlers: ["WASM", "SERVER"]`
5. All existing tests pass
6. New MCP tests pass
