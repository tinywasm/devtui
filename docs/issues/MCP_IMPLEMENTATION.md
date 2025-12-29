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

## Implementation Steps

Complete each step in order. Mark `[x]` when done.

- [ ] **Step 1**: [Refactor formatMessage for styled/unstyled output](MCP_STEP1_REFACTOR_FORMAT.md)
- [ ] **Step 2**: [Create mcp.go with tool metadata and implementation](MCP_STEP2_CREATE_MCP_GO.md)  
- [ ] **Step 3**: [Integrate devtui MCP tools in tinywasm/app](MCP_STEP3_APP_INTEGRATION.md)
- [ ] **Step 4**: [Verification and testing](MCP_STEP4_VERIFICATION.md)

## Files to Modify/Create

### DevTUI Package (`tinywasm/devtui`)

| File | Action | Description |
|------|--------|-------------|
| `print.go` | MODIFY | Refactor `formatMessage` to support styled/unstyled output |
| `view.go` | MODIFY | Add `ContentViewPlain()` method for unstyled output |
| `mcp.go` | CREATE | MCP tool metadata and implementation |

### App Package (`tinywasm/app`)

| File | Action | Description |
|------|--------|-------------|
| `start.go` | MODIFY | Pass DevTUI as tool handler to mcpserve |
| `interface.go` | MODIFY | Extend `TuiInterface` if needed |

## Expected Tool Behavior

### Tool: `devtui_get_section_logs`

**Description**: Get logs from a specific DevTUI section (tab). Lists available sections when called without parameters.

**Parameters**:
| Name | Type | Required | Description |
|------|------|----------|-------------|
| `section` | string | No | Section title (e.g., "BUILD", "DEPLOY"). If empty, lists available sections |

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
