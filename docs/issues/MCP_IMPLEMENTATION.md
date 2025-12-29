# MCP Integration for DevTUI

This document describes the implementation plan for integrating MCP (Model Context Protocol) tools into the DevTUI package, allowing LLMs to view the logs displayed in each registered tab section.

## Objective

Create a single MCP tool `devtui_get_section_logs` that:
1. Lists all available sections (tabs) by title when no parameter is provided
2. Returns the logs of a specific section when section title is provided
3. Returns logs in **plain text format without ANSI styles** (same content as terminal but clean for LLM context)

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Coupling** | Decoupled | DevTUI does NOT import `mcpserve`. Uses reflection-compatible structs |
| **File Location** | `/devtui/mcp.go` | Single file, no subpackage |
| **Registration Method** | `GetMCPToolsMetadata()` | Consistent with existing mcpserve pattern |
| **Log Limit** | No limit | Handlers use `MessageTracker` to avoid log growth |
| **Output Format** | Plain text | Same as `ContentView()` but without ANSI escape codes |

## Critical: Message Storage and Rendering

> **IMPORTANT**: Messages are stored ONCE without ANSI styles. Styles are applied ON-DEMAND at render time.

```
┌─────────────────────────────────────────────────┐
│  tabContent.Content = "Compiling main.go..."    │  ← Stored ONCE (no ANSI)
│  tabContent.Type = Msg.Success                  │  ← Message type for styling
└──────────────────────┬──────────────────────────┘
                       │
           ┌───────────┴───────────┐
           ↓                       ↓
 formatMessage(styled=true)    formatMessage(styled=false)
           ↓                       ↓
   "\x1b[32m..."                "12:30:45 [WASM] Compiling..."
   (terminal display)           (MCP/LLM output)
```

**Key points:**
- **NO memory duplication** - message stored once without styles
- **NO code duplication** - single `formatMessage()` function with `styled` parameter
- `styled=true` → applies ANSI colors via `applyMessageTypeStyle()` (for terminal)
- `styled=false` → returns plain text (for MCP tool output)

## Implementation Steps

Complete each step in order. Mark `[x]` when done.

### DevTUI Changes (this package)
- [ ] **Step 1**: [Refactor formatMessage for styled/unstyled output](MCP_STEP1_REFACTOR_FORMAT.md)
- [ ] **Step 2**: [Create mcp.go with tool metadata and implementation](MCP_STEP2_CREATE_MCP_GO.md)  
- [ ] **Step 3**: [Verification and testing](MCP_STEP3_VERIFICATION.md)

### App Integration (tinywasm/app)
- [ ] **Step 4**: See `tinywasm/app/docs/issues/MCP_DEVTUI_INTEGRATION.md`

## Files to Modify/Create

### DevTUI Package (`tinywasm/devtui`)

| File | Action | Description |
|------|--------|-------------|
| `print.go` | MODIFY | Refactor `formatMessage` to support styled/unstyled output |
| `view.go` | MODIFY | Add `ContentViewPlain()` method for unstyled output |
| `mcp.go` | CREATE | MCP tool metadata and implementation |

### App Package (`tinywasm/app`) - See separate issue

| File | Action | Description |
|------|--------|-------------|
| `start.go` | MODIFY | Pass DevTUI as tool handler to mcpserve |

## Expected Tool Behavior

### Tool: `devtui_get_section_logs`

**Description** (dynamically generated): 
```
Get logs from a specific DevTUI terminal section (tab). Available sections: 'SHORTCUTS', 'BUILD', 'DEPLOY'. Pass empty section parameter to list sections with descriptions.
```

> **Note**: The description is generated dynamically at runtime to include all registered sections. This allows LLMs to know valid section names directly from the tool listing.

**Parameters**:
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `section` | string | No | Section title to get logs from. Leave empty to list sections. EnumValues = section titles |

**Example Responses**:

When `section` is empty:
```
Available sections:
- SHORTCUTS
- BUILD
- DEPLOY
```

When `section` is "BUILD":
```
12:30:45 [WASM    ] Compiling main.go...
12:30:46 [WASM    ] Build successful
12:30:46 [Server  ] Reloading browser...
```

## Success Criteria

1. Tool `devtui_get_section_logs` appears in MCP server tools list
2. Calling tool without parameters lists all registered sections
3. Calling tool with valid section name returns logs in plain text
4. No ANSI escape codes in output (verified by checking for `\x1b[` sequences)
5. All existing devtui tests pass
6. All existing app tests pass
